// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "strings"

// ANSIOptions configures AsANSI, mirroring rqrcode's as_ansi option hash. Empty
// strings fall back to the gem defaults, and QuietZoneSize < 0 means "unset" so
// the default of 4 applies (0 is a meaningful value: no quiet zone).
type ANSIOptions struct {
	Light         string // foreground escape (default "\033[47m")
	Dark          string // background escape (default "\033[40m")
	FillCharacter string // written cell (default "  ")
	QuietZoneSize int    // quiet-zone width (default 4); negative selects the default
}

// AsANSI renders the QR code with ANSI background colors, matching RQRCode's
// as_ansi. The zero-value options reproduce the gem defaults (note: a zero
// QuietZoneSize means no quiet zone; use -1 or leave DefaultANSIOptions to get 4).
func (q *QRCode) AsANSI(opts ANSIOptions) string {
	light := opts.Light
	if light == "" {
		light = "\033[47m"
	}
	dark := opts.Dark
	if dark == "" {
		dark = "\033[40m"
	}
	fillCharacter := opts.FillCharacter
	if fillCharacter == "" {
		fillCharacter = "  "
	}
	quietZoneSize := opts.QuietZoneSize
	if quietZoneSize < 0 {
		quietZoneSize = 4
	}
	normal := "\033[m\n"

	var output []string
	for c := 0; c < q.ModuleCount; c++ {
		row := light + strings.Repeat(fillCharacter, quietZoneSize)
		previousDark := false
		for r := 0; r < q.ModuleCount; r++ {
			if q.Modules[c][r] {
				if !previousDark {
					row += dark
					previousDark = true
				}
			} else if previousDark {
				row += light
				previousDark = false
			}
			row += fillCharacter
		}
		if previousDark {
			row += light
		}
		row += strings.Repeat(fillCharacter, quietZoneSize)
		row += normal
		output = append(output, row)
	}

	// count the row width (number of fill_character occurrences in first row).
	width := strings.Count(output[0], fillCharacter)
	quietRow := light + strings.Repeat(fillCharacter, width) + normal
	quietRows := strings.Repeat(quietRow, quietZoneSize)

	return quietRows + strings.Join(output, "") + quietRows
}

// AsANSIDefault renders with the gem's default options (quiet zone 4).
func (q *QRCode) AsANSIDefault() string {
	return q.AsANSI(ANSIOptions{QuietZoneSize: -1})
}
