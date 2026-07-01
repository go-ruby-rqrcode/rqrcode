// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
)

// pngEncode is the PNG encoder, indirected through a variable so the (otherwise
// unreachable) encode-error branch can be exercised in tests.
var pngEncode = func(w io.Writer, img image.Image) error { return png.Encode(w, img) }

// PNGOptions configures AsPNG. This renderer is a pure-Go (stdlib image/png,
// CGO=0) implementation of rqrcode's "Original" as_png sizing: each module is
// ModulePxSize pixels square and BorderModules modules of quiet zone surround
// the code. (The gem's chunky_png "Google" resize mode is intentionally not
// reproduced byte-for-byte, as it depends on chunky_png's own zlib encoder; the
// module geometry here is deterministic and standard.)
type PNGOptions struct {
	ModulePxSize  int         // pixels per module (default 6)
	BorderModules int         // quiet-zone width in modules (default 4)
	Color         color.Color // foreground (default black)
	Fill          color.Color // background (default white)
}

// AsPNG renders the QR code to PNG bytes with the given options, using the
// deterministic module-per-pixel geometry.
func (q *QRCode) AsPNG(opts PNGOptions) ([]byte, error) {
	modulePx := opts.ModulePxSize
	if modulePx <= 0 {
		modulePx = 6
	}
	border := opts.BorderModules
	if border < 0 {
		border = 0
	}
	if opts.BorderModules == 0 {
		border = 4
	}
	fg := opts.Color
	if fg == nil {
		fg = color.Gray{Y: 0}
	}
	bg := opts.Fill
	if bg == nil {
		bg = color.Gray{Y: 255}
	}

	sideModules := q.ModuleCount + 2*border
	side := sideModules * modulePx
	img := image.NewNRGBA(image.Rect(0, 0, side, side))

	// Fill background.
	fr, fgc, fb, fa := bg.RGBA()
	bgN := color.NRGBA{R: uint8(fr >> 8), G: uint8(fgc >> 8), B: uint8(fb >> 8), A: uint8(fa >> 8)}
	for y := 0; y < side; y++ {
		for x := 0; x < side; x++ {
			img.SetNRGBA(x, y, bgN)
		}
	}

	// Paint dark modules.
	cr, cg, cb, ca := fg.RGBA()
	fgN := color.NRGBA{R: uint8(cr >> 8), G: uint8(cg >> 8), B: uint8(cb >> 8), A: uint8(ca >> 8)}
	for row := 0; row < q.ModuleCount; row++ {
		for col := 0; col < q.ModuleCount; col++ {
			if !q.Modules[row][col] {
				continue
			}
			x0 := (col + border) * modulePx
			y0 := (row + border) * modulePx
			for dy := 0; dy < modulePx; dy++ {
				for dx := 0; dx < modulePx; dx++ {
					img.SetNRGBA(x0+dx, y0+dy, fgN)
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := pngEncode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
