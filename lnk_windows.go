//go:build windows

package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	clctxInprocServer = 0x1
	stgmRead          = 0x0
	slrNoUI           = 0x1
	linkBufferLen     = 1024
)

var (
	clsidShellLink = windows.GUID{
		Data1: 0x00021401, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	iidIShellLinkW = windows.GUID{
		Data1: 0x000214F9, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	iidIPersistFile = windows.GUID{
		Data1: 0x0000010B, Data2: 0x0000, Data3: 0x0000,
		Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46},
	}
	procCoCreateInstance    = ole32.NewProc("CoCreateInstance")
	procCoTaskMemFree       = ole32.NewProc("CoTaskMemFree")
	procSHGetPathFromIDList = shell32.NewProc("SHGetPathFromIDListW")
)

// shellLinkWVtbl mirrors IShellLinkW (IUnknown + 18 methods), matching the
// layout documented for the Windows ShellLink COM object.
type shellLinkWVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	GetPath             uintptr
	GetIDList           uintptr
	SetIDList           uintptr
	GetDescription      uintptr
	SetDescription      uintptr
	GetWorkingDirectory uintptr
	SetWorkingDirectory uintptr
	GetArguments        uintptr
	SetArguments        uintptr
	GetHotkey           uintptr
	SetHotkey           uintptr
	GetShowCmd          uintptr
	SetShowCmd          uintptr
	GetIconLocation     uintptr
	SetIconLocation     uintptr
	SetRelativePath     uintptr
	Resolve             uintptr
	SetPath             uintptr
}

type shellLinkW struct{ lpVtbl *shellLinkWVtbl }

// persistFileVtbl mirrors IPersistFile (IUnknown + IPersist::GetClassID +
// IsDirty/Load/Save/SaveCompleted/GetCurFile).
type persistFileVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetClassID     uintptr
	IsDirty        uintptr
	Load           uintptr
	Save           uintptr
	SaveCompleted  uintptr
	GetCurFile     uintptr
}

type persistFile struct{ lpVtbl *persistFileVtbl }

func newShellLink() (*shellLinkW, error) {
	var sl *shellLinkW
	r1, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		uintptr(clctxInprocServer),
		uintptr(unsafe.Pointer(&iidIShellLinkW)),
		uintptr(unsafe.Pointer(&sl)),
	)
	if int32(r1) < 0 || sl == nil || sl.lpVtbl == nil {
		return nil, fmt.Errorf("CoCreateInstance(CLSID_ShellLink) failed: 0x%08x", uint32(int32(r1)))
	}
	return sl, nil
}

func (sl *shellLinkW) release() {
	syscall.SyscallN(sl.lpVtbl.Release, uintptr(unsafe.Pointer(sl)))
}

func (sl *shellLinkW) queryPersistFile() (*persistFile, error) {
	var pf *persistFile
	r1, _, _ := syscall.SyscallN(sl.lpVtbl.QueryInterface,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(&iidIPersistFile)),
		uintptr(unsafe.Pointer(&pf)),
	)
	if int32(r1) < 0 || pf == nil || pf.lpVtbl == nil {
		return nil, fmt.Errorf("IShellLinkW::QueryInterface(IPersistFile) failed: 0x%08x", uint32(int32(r1)))
	}
	return pf, nil
}

func (pf *persistFile) release() {
	syscall.SyscallN(pf.lpVtbl.Release, uintptr(unsafe.Pointer(pf)))
}

func (pf *persistFile) load(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r1, _, _ := syscall.SyscallN(pf.lpVtbl.Load,
		uintptr(unsafe.Pointer(pf)),
		uintptr(unsafe.Pointer(p)),
		uintptr(stgmRead),
	)
	if int32(r1) < 0 {
		return fmt.Errorf("IPersistFile::Load failed: 0x%08x", uint32(int32(r1)))
	}
	return nil
}

// resolve lets the shell relocate a stale/relative target without showing UI.
func (sl *shellLinkW) resolve() {
	syscall.SyscallN(sl.lpVtbl.Resolve, uintptr(unsafe.Pointer(sl)), 0, uintptr(slrNoUI))
}

// fieldString reads a string-returning IShellLinkW method (GetArguments,
// GetWorkingDirectory) whose vtable signature is (this, buffer, bufferLen).
func (sl *shellLinkW) fieldString(method uintptr) string {
	buf := make([]uint16, linkBufferLen)
	r1, _, _ := syscall.SyscallN(method,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if int32(r1) < 0 {
		return ""
	}
	return strings.TrimSpace(windows.UTF16ToString(buf))
}

func (sl *shellLinkW) getPath() string {
	buf := make([]uint16, linkBufferLen)
	r1, _, _ := syscall.SyscallN(sl.lpVtbl.GetPath,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0, // pfd (WIN32_FIND_DATAW*)
		0, // fFlags: default, returns the expanded target path
	)
	if int32(r1) < 0 {
		return ""
	}
	p := strings.TrimSpace(windows.UTF16ToString(buf))
	// Some shortcuts store the target with environment variables
	// (e.g. %LOCALAPPDATA%\...); expand them like the shell would.
	if strings.Contains(p, "%") {
		p = os.ExpandEnv(p)
	}
	return normalizePath(p)
}

// getIDListPath resolves the shortcut's IDList into a filesystem path,
// used as a fallback when no plain path string is stored.
func (sl *shellLinkW) getIDListPath() string {
	var pidl uintptr
	r1, _, _ := syscall.SyscallN(sl.lpVtbl.GetIDList,
		uintptr(unsafe.Pointer(sl)),
		uintptr(unsafe.Pointer(&pidl)),
	)
	if int32(r1) < 0 || pidl == 0 {
		return ""
	}
	defer procCoTaskMemFree.Call(pidl)
	buf := make([]uint16, windows.MAX_PATH)
	r2, _, _ := procSHGetPathFromIDList.Call(pidl, uintptr(unsafe.Pointer(&buf[0])))
	if r2 == 0 {
		return ""
	}
	return normalizePath(windows.UTF16ToString(buf))
}

// resolveShortcut parses a Windows .lnk file through the ShellLink COM object,
// returning the shortcut's target path, launch arguments and start-in folder.
func resolveShortcut(path string) (shortcutInfo, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ok, uninit := coInitCOM()
	if !ok {
		return shortcutInfo{}, fmt.Errorf("CoInitializeEx failed")
	}
	if uninit {
		defer procCoUninitialize.Call()
	}

	sl, err := newShellLink()
	if err != nil {
		return shortcutInfo{}, err
	}
	defer sl.release()

	pf, err := sl.queryPersistFile()
	if err != nil {
		return shortcutInfo{}, err
	}
	defer pf.release()

	if err := pf.load(path); err != nil {
		return shortcutInfo{}, err
	}

	sl.resolve()

	target := sl.getPath()
	if target == "" {
		target = sl.getIDListPath()
	}
	if target == "" {
		return shortcutInfo{}, fmt.Errorf("shortcut has no file target")
	}
	info := shortcutInfo{
		Target:  target,
		Args:    sl.fieldString(sl.lpVtbl.GetArguments),
		WorkDir: normalizePath(sl.fieldString(sl.lpVtbl.GetWorkingDirectory)),
	}
	return info, nil
}
