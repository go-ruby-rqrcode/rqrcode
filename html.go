// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "strings"

// HTML table fragments, matching RQRCode::Export::HTML.
const (
	htmlTableOpen  = "<table>"
	htmlTableClose = "</table>"
	htmlTROpen     = "<tr>"
	htmlTRClose    = "</tr>"
	htmlTDBlack    = `<td class="black"></td>`
	htmlTDWhite    = `<td class="white"></td>`
)

// AsHTML renders the QR code as an HTML table, matching RQRCode's as_html.
func (q *QRCode) AsHTML() string {
	var b strings.Builder
	b.WriteString(htmlTableOpen)
	for row := 0; row < q.ModuleCount; row++ {
		b.WriteString(htmlTROpen)
		for col := 0; col < q.ModuleCount; col++ {
			if q.Modules[row][col] {
				b.WriteString(htmlTDBlack)
			} else {
				b.WriteString(htmlTDWhite)
			}
		}
		b.WriteString(htmlTRClose)
	}
	b.WriteString(htmlTableClose)
	return b.String()
}
