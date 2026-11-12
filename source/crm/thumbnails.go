package crm

import (
	"bytes"
	"image"
	"image/jpeg"
)

var photoProcessing = make(chan struct{}, 1)

func thumbnail(data []byte) ([]byte, error) {
	img, _, e := image.Decode(bytes.NewReader(data))
	if e != nil {
		return nil, e
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	nw, nh := w, h
	if w > 1800 || h > 1800 {
		if w >= h {
			nw = 1800
			nh = max(1, h*1800/w)
		} else {
			nh = 1800
			nw = max(1, w*1800/h)
		}
	}
	out := image.NewRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			out.Set(x, y, img.At(b.Min.X+x*w/nw, b.Min.Y+y*h/nh))
		}
	}
	var buf bytes.Buffer
	e = jpeg.Encode(&buf, out, &jpeg.Options{Quality: 88})
	return buf.Bytes(), e
}
