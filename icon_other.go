//go:build !windows

package main

import (
	"image"
	"image/color"
)

func iconForFile(path string) (image.Image, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 48, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 0x9e, G: 0x9e, B: 0x9e, A: 0xff})
		}
	}
	return img, nil
}
