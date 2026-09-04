//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"syscall"
	"unsafe"

	"github.com/fcjr/geticon"
	"golang.org/x/sys/windows"
)

const (
	shgfiIcon      = 0x000000100
	shgfiLargeIcon = 0x000000000
	dibRGBColors   = 0
	biRGB          = 0

	coinitApartmentThreaded = 0x2
	siigbfIconOnly          = 0x4
	maxIconSize             = 256
)

var iidShellItemImageFactory = windows.GUID{
	Data1: 0xbcc18b79,
	Data2: 0xba16,
	Data3: 0x442f,
	Data4: [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b},
}

type shellItemImageFactoryVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetImage       uintptr
}

type shellItemImageFactory struct {
	lpVtbl *shellItemImageFactoryVtbl
}

type sHFileInfoW struct {
	hIcon         windows.Handle
	iIcon         int32
	dwAttributes  uint32
	szDisplayName [260]uint16
	szTypeName    [80]uint16
}

type iconInfo struct {
	fIcon    int32
	xHotspot int32
	yHotspot int32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bm struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       unsafe.Pointer
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

type bitmapInfo struct {
	bmiHeader bitmapInfoHeader
	bmiColors [1]uint32
}

var (
	ole32              = syscall.NewLazyDLL("ole32.dll")
	shell32            = syscall.NewLazyDLL("shell32.dll")
	user32             = syscall.NewLazyDLL("user32.dll")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	procCoInitializeEx = ole32.NewProc("CoInitializeEx")
	procCoUninitialize = ole32.NewProc("CoUninitialize")
	procSHGetFileInfo  = shell32.NewProc("SHGetFileInfoW")
	procCreateItem     = shell32.NewProc("SHCreateItemFromParsingName")
	procGetIconInfo    = user32.NewProc("GetIconInfo")
	procDestroyIcon    = user32.NewProc("DestroyIcon")
	procGetObjectW     = gdi32.NewProc("GetObjectW")
	procGetDIBits      = gdi32.NewProc("GetDIBits")
	procDeleteObject   = gdi32.NewProc("DeleteObject")
	procCreateCompatDC = gdi32.NewProc("CreateCompatibleDC")
	procDeleteDC       = gdi32.NewProc("DeleteDC")
)

func readDIB(hbm windows.Handle, width, height int) ([]byte, error) {
	dc, _, _ := procCreateCompatDC.Call(0)
	if dc == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(dc)

	bih := bitmapInfoHeader{
		biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		biWidth:       int32(width),
		biHeight:      -int32(height),
		biPlanes:      1,
		biBitCount:    32,
		biCompression: biRGB,
	}
	buf := make([]byte, width*height*4)
	r1, _, _ := procGetDIBits.Call(
		dc,
		uintptr(hbm),
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&bitmapInfo{bmiHeader: bih})),
		uintptr(dibRGBColors),
	)
	if r1 == 0 {
		return nil, fmt.Errorf("GetDIBits failed")
	}
	return buf, nil
}

func iconForFile(path string) (image.Image, error) {
	img, err := geticon.FromPath(path)
	if err == nil && img != nil {
		return img, nil
	}
	img, err = shellIconLargeForFile(path)
	if err == nil && img != nil {
		return img, nil
	}
	return shellIconForFile(path)
}

func coInitCOM() (ok, uninit bool) {
	r1, _, _ := procCoInitializeEx.Call(0, uintptr(coinitApartmentThreaded))
	switch int32(r1) {
	case 0: // S_OK: this call initialized COM
		return true, true
	case 1: // S_FALSE: already initialized on this thread
		return true, false
	default: // RPC_E_CHANGED_MODE etc.
		return false, false
	}
}

func newShellItemImageFactory(path string) (*shellItemImageFactory, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var f *shellItemImageFactory
	r1, _, _ := procCreateItem.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(unsafe.Pointer(&iidShellItemImageFactory)),
		uintptr(unsafe.Pointer(&f)),
	)
	if int32(r1) < 0 || f == nil || f.lpVtbl == nil {
		return nil, fmt.Errorf("SHCreateItemFromParsingName failed: 0x%08x", uint32(int32(r1)))
	}
	return f, nil
}

func (f *shellItemImageFactory) getImage(size int) (windows.Handle, error) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		return 0, fmt.Errorf("IShellItemImageFactory unsupported on 32-bit")
	}
	packed := uint64(uint32(size))<<32 | uint64(uint32(size))
	var hbm windows.Handle
	r1, _, _ := syscall.SyscallN(f.lpVtbl.GetImage,
		uintptr(unsafe.Pointer(f)),
		uintptr(packed),
		uintptr(siigbfIconOnly),
		uintptr(unsafe.Pointer(&hbm)),
	)
	if int32(r1) < 0 {
		return 0, fmt.Errorf("GetImage failed: 0x%08x", uint32(int32(r1)))
	}
	return hbm, nil
}

func (f *shellItemImageFactory) release() {
	syscall.SyscallN(f.lpVtbl.Release, uintptr(unsafe.Pointer(f)))
}

func shellIconLargeForFile(path string) (image.Image, error) {
	if err := procCoInitializeEx.Find(); err != nil {
		return nil, err
	}
	if err := procCreateItem.Find(); err != nil {
		return nil, err
	}
	ok, uninit := coInitCOM()
	if !ok {
		return nil, fmt.Errorf("CoInitializeEx failed")
	}
	if uninit {
		defer procCoUninitialize.Call()
	}

	f, err := newShellItemImageFactory(path)
	if err != nil {
		return nil, err
	}
	defer f.release()

	hbm, err := f.getImage(maxIconSize)
	if err != nil {
		return nil, err
	}
	defer procDeleteObject.Call(uintptr(hbm))

	var b bm
	r1, _, _ := procGetObjectW.Call(uintptr(hbm), uintptr(unsafe.Sizeof(b)), uintptr(unsafe.Pointer(&b)))
	if r1 == 0 {
		return nil, fmt.Errorf("GetObject failed")
	}
	w := int(b.bmWidth)
	height := int(b.bmHeight)
	if height < 0 {
		height = -height
	}
	if w <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid bitmap size %dx%d", w, height)
	}

	pix, err := readDIB(hbm, w, height)
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, height))
	for y := 0; y < height; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			img.SetNRGBA(x, y, color.NRGBA{R: pix[i+2], G: pix[i+1], B: pix[i], A: pix[i+3]})
		}
	}
	return img, nil
}

func shellIconForFile(path string) (image.Image, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	var sf sHFileInfoW
	r1, _, callErr := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(p)),
		0,
		uintptr(unsafe.Pointer(&sf)),
		uintptr(unsafe.Sizeof(sf)),
		uintptr(shgfiIcon|shgfiLargeIcon),
	)
	if r1 == 0 {
		return nil, callErr
	}
	defer procDestroyIcon.Call(uintptr(sf.hIcon))

	var ii iconInfo
	r1, _, _ = procGetIconInfo.Call(uintptr(sf.hIcon), uintptr(unsafe.Pointer(&ii)))
	if r1 == 0 {
		return nil, fmt.Errorf("GetIconInfo failed")
	}
	defer procDeleteObject.Call(uintptr(ii.hbmColor))
	defer procDeleteObject.Call(uintptr(ii.hbmMask))

	var b bm
	r1, _, _ = procGetObjectW.Call(uintptr(ii.hbmColor), uintptr(unsafe.Sizeof(b)), uintptr(unsafe.Pointer(&b)))
	if r1 == 0 {
		return nil, fmt.Errorf("GetObject failed")
	}

	w := int(b.bmWidth)
	height := int(b.bmHeight)
	if height < 0 {
		height = -height
	}
	if w <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid bitmap size %dx%d", w, height)
	}

	pix, err := readDIB(ii.hbmColor, w, height)
	if err != nil {
		return nil, err
	}
	mask, err := readDIB(ii.hbmMask, w, height)
	if err != nil {
		return nil, err
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, height))
	for y := 0; y < height; y++ {
		for x := 0; x < w; x++ {
			i := (y*w + x) * 4
			a := pix[i+3]
			if a == 0 && mask[i] == 0 {
				a = 255
			}
			img.SetNRGBA(x, y, color.NRGBA{R: pix[i+2], G: pix[i+1], B: pix[i], A: a})
		}
	}
	return img, nil
}
