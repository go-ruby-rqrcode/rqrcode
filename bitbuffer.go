// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

// bitBuffer is RQRCodeCore::QRBitBuffer.
type bitBuffer struct {
	version int
	buffer  []int
	length  int
}

// Padding codewords (QRBitBuffer::PAD0 / PAD1).
const (
	pad0 = 0xEC
	pad1 = 0x11
)

// newBitBuffer builds a buffer for a version (QRBitBuffer#initialize).
func newBitBuffer(version int) *bitBuffer {
	return &bitBuffer{version: version}
}

// put appends num's low `length` bits, MSB first (QRBitBuffer#put).
func (b *bitBuffer) put(num, length int) {
	for i := 0; i < length; i++ {
		b.putBit((rszf(num, length-i-1) & 1) == 1)
	}
}

// lengthInBits is QRBitBuffer#get_length_in_bits.
func (b *bitBuffer) lengthInBits() int { return b.length }

// putBit appends a single bit (QRBitBuffer#put_bit).
func (b *bitBuffer) putBit(bit bool) {
	bufIndex := b.length / 8
	if len(b.buffer) <= bufIndex {
		b.buffer = append(b.buffer, 0)
	}
	if bit {
		b.buffer[bufIndex] |= rszf(0x80, b.length%8)
	}
	b.length++
}

// byteEncodingStart is QRBitBuffer#byte_encoding_start.
func (b *bitBuffer) byteEncodingStart(length int) {
	b.put(int(ModeByte8bit), 4)
	b.put(length, getLengthInBits(ModeByte8bit, b.version))
}

// alphanumericEncodingStart is QRBitBuffer#alphanumeric_encoding_start.
func (b *bitBuffer) alphanumericEncodingStart(length int) {
	b.put(int(ModeAlphanumeric), 4)
	b.put(length, getLengthInBits(ModeAlphanumeric, b.version))
}

// numericEncodingStart is QRBitBuffer#numeric_encoding_start.
func (b *bitBuffer) numericEncodingStart(length int) {
	b.put(int(ModeNumeric), 4)
	b.put(length, getLengthInBits(ModeNumeric, b.version))
}

// padUntil is QRBitBuffer#pad_until.
func (b *bitBuffer) padUntil(preferredSize int) {
	for b.lengthInBits()%8 != 0 {
		b.putBit(false)
	}
	for b.lengthInBits() < preferredSize {
		b.put(pad0, 8)
		if b.lengthInBits() < preferredSize {
			b.put(pad1, 8)
		}
	}
}

// endOfMessage is QRBitBuffer#end_of_message.
func (b *bitBuffer) endOfMessage(maxDataBits int) {
	if b.lengthInBits()+4 <= maxDataBits {
		b.put(0, 4)
	}
}
