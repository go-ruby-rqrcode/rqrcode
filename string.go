// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "strings"

// StringOptions configures ToString, mirroring RQRCodeCore::QRCode#to_s options.
type StringOptions struct {
	Dark          string // character(s) for a dark module (default "x")
	Light         string // character(s) for a light module (default " ")
	QuietZoneSize int    // quiet-zone width in modules (default 0)
}

// ToString renders the matrix as text, matching RQRCodeCore::QRCode#to_s. The
// zero-value options reproduce the gem's defaults (dark "x", light " ", no quiet
// zone).
func (q *QRCode) ToString(opts StringOptions) string {
	dark := opts.Dark
	if dark == "" {
		dark = "x"
	}
	light := opts.Light
	if light == "" {
		light = " "
	}
	quiet := opts.QuietZoneSize

	var rows []string
	for _, row := range q.Modules {
		cols := strings.Repeat(light, quiet)
		for _, col := range row {
			if col {
				cols += dark
			} else {
				cols += light
			}
		}
		rows = append(rows, cols)
	}

	for i := 0; i < quiet; i++ {
		// Ruby: light * (rows.first.length / light.size). length here is the byte
		// length of the first row divided by the light-string byte length.
		width := len(rows[0]) / len(light)
		pad := strings.Repeat(light, width)
		rows = append([]string{pad}, rows...)
		rows = append(rows, strings.Repeat(light, width))
	}
	return strings.Join(rows, "\n")
}

// String renders the code with the gem's default to_s (dark "x", light " ").
func (q *QRCode) String() string {
	return q.ToString(StringOptions{})
}
