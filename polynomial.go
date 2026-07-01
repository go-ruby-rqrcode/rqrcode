// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "fmt"

// polynomial is RQRCodeCore::QRPolynomial: a GF(256) polynomial for Reed-Solomon.
type polynomial struct {
	num []int
}

// newPolynomial builds a polynomial from coefficients, stripping leading zeros
// and left-padding by shift (QRPolynomial#initialize).
func newPolynomial(num []int, shift int) *polynomial {
	if len(num) == 0 {
		panic(fmt.Errorf("%w: %d/%d", ErrRunTime, len(num), shift))
	}
	offset := 0
	for offset < len(num) && num[offset] == 0 {
		offset++
	}
	p := &polynomial{num: make([]int, len(num)-offset+shift)}
	for i := 0; i < len(num)-offset; i++ {
		p.num[i] = num[i+offset]
	}
	return p
}

// get is QRPolynomial#get.
func (p *polynomial) get(index int) int { return p.num[index] }

// length is QRPolynomial#get_length.
func (p *polynomial) length() int { return len(p.num) }

// multiply is QRPolynomial#multiply.
func (p *polynomial) multiply(e *polynomial) *polynomial {
	num := make([]int, p.length()+e.length()-1)
	for i := 0; i < p.length(); i++ {
		for j := 0; j < e.length(); j++ {
			num[i+j] ^= gexp(glog(p.get(i)) + glog(e.get(j)))
		}
	}
	return newPolynomial(num, 0)
}

// mod is QRPolynomial#mod (recursive polynomial remainder).
func (p *polynomial) mod(e *polynomial) *polynomial {
	if p.length()-e.length() < 0 {
		return p
	}
	ratio := glog(p.get(0)) - glog(e.get(0))
	num := make([]int, p.length())
	copy(num, p.num)
	for i := 0; i < e.length(); i++ {
		num[i] ^= gexp(glog(e.get(i)) + ratio)
	}
	return newPolynomial(num, 0).mod(e)
}
