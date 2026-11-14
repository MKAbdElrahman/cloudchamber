package sim

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
)

// framePNG is a small image standing in for one extracted video frame.
func framePNG() ([]byte, error) {
	const width, height = 64, 36
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: uint8(20 + x*3), G: uint8(40 + y*4), B: 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
