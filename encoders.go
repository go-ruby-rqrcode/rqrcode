// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import (
	"strconv"
	"strings"
)

// dataWriter is the interface satisfied by the per-mode encoders (the Ruby
// duck-typed `write(buffer)` protocol shared by QRNumeric / QRAlphanumeric /
// QR8bitByte / QRMulti).
type dataWriter interface {
	write(b *bitBuffer)
}

// numericChars is RQRCodeCore::NUMERIC.
const numericChars = "0123456789"

// alphanumericChars is RQRCodeCore::ALPHANUMERIC, in index order.
const alphanumericChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ $%*+-./:"

// validNumeric reports whether data is purely numeric (QRNumeric.valid_data?).
// The empty string is valid (Ruby: `("".chars - NUMERIC).empty?` is true).
func validNumeric(data string) bool {
	for _, r := range data {
		if !strings.ContainsRune(numericChars, r) {
			return false
		}
	}
	return true
}

// validAlphanumeric reports whether data is valid alphanumeric
// (QRAlphanumeric.valid_data?). The empty string is valid.
func validAlphanumeric(data string) bool {
	for _, r := range data {
		if !strings.ContainsRune(alphanumericChars, r) {
			return false
		}
	}
	return true
}

// alphanumericIndex returns the code value of a single alphanumeric char.
func alphanumericIndex(c byte) int {
	return strings.IndexByte(alphanumericChars, c)
}

// qrNumeric is RQRCodeCore::QRNumeric.
type qrNumeric struct{ data string }

// write encodes the numeric data (QRNumeric#write).
func (q *qrNumeric) write(b *bitBuffer) {
	// @data.size is the character count; numeric data is ASCII digits so
	// len == char count.
	n := len(q.data)
	b.numericEncodingStart(n)
	for i := 0; i < n; i++ {
		if i%3 == 0 {
			end := i + 3
			if end > n {
				end = n
			}
			chars := q.data[i:end]
			bitLength := numberLength(len(chars))
			code, _ := strconv.Atoi(chars)
			b.put(code, bitLength)
		}
	}
}

// qrAlphanumeric is RQRCodeCore::QRAlphanumeric.
type qrAlphanumeric struct{ data string }

// write encodes the alphanumeric data (QRAlphanumeric#write).
func (q *qrAlphanumeric) write(b *bitBuffer) {
	n := len(q.data)
	b.alphanumericEncodingStart(n)
	for i := 0; i < n; i++ {
		if i%2 == 0 {
			if i == n-1 {
				value := alphanumericIndex(q.data[i])
				b.put(value, 6)
			} else {
				value := alphanumericIndex(q.data[i])*45 + alphanumericIndex(q.data[i+1])
				b.put(value, 11)
			}
		}
	}
}

// qr8bitByte is RQRCodeCore::QR8bitByte.
type qr8bitByte struct{ data string }

// write encodes the raw bytes (QR8bitByte#write).
func (q *qr8bitByte) write(b *bitBuffer) {
	b.byteEncodingStart(len(q.data)) // bytesize
	for i := 0; i < len(q.data); i++ {
		b.put(int(q.data[i]), 8)
	}
}

// qrMulti is RQRCodeCore::QRMulti.
type qrMulti struct{ segs []*segment }

// write encodes each segment in order (QRMulti#write).
func (q *qrMulti) write(b *bitBuffer) {
	for _, s := range q.segs {
		s.writer().write(b)
	}
}
