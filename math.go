// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "fmt"

// expTable / logTable are the GF(256) exponent and log tables, built exactly as
// RQRCodeCore::QRMath's module_eval does.
var (
	expTable [256]int
	logTable [256]int
)

func init() {
	for i := 0; i < 8; i++ {
		expTable[i] = 1 << uint(i)
	}
	for i := 8; i < 256; i++ {
		expTable[i] = expTable[i-4] ^ expTable[i-5] ^ expTable[i-6] ^ expTable[i-8]
	}
	for i := 0; i < 255; i++ {
		logTable[expTable[i]] = i
	}
}

// glog is RQRCodeCore::QRMath.glog. It panics via ErrRunTime semantics only when
// n < 1, which the callers never do; kept for faithfulness.
func glog(n int) int {
	if n < 1 {
		panic(fmt.Errorf("%w: glog(%d)", ErrRunTime, n))
	}
	return logTable[n]
}

// gexp is RQRCodeCore::QRMath.gexp, wrapping the exponent into 0..254.
func gexp(n int) int {
	for n < 0 {
		n += 255
	}
	for n >= 256 {
		n -= 255
	}
	return expTable[n]
}
