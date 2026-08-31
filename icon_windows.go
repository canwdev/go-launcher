//go:build windows

package main

import (
	"fmt"
	"image"
	"image/color"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	shgfiIcon      = 0x000000100
	shgfiLargeIcon = 0x000000000
	dibRGBColors   = 0
	biRGB          = 0
)

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
	shell32            = syscall.NewLazyDLL("shell32.dll")
	user32             = syscall.NewLazyDLL("user32.dll")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	procSHGetFileInfo  = shell32.NewProc("SHGetFileInfoW")
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
