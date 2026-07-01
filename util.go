// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

// patternPositionTable is RQRCodeCore::QRUtil::PATTERN_POSITION_TABLE.
var patternPositionTable = [][]int{
	{},
	{6, 18},
	{6, 22},
	{6, 26},
	{6, 30},
	{6, 34},
	{6, 22, 38},
	{6, 24, 42},
	{6, 26, 46},
	{6, 28, 50},
	{6, 30, 54},
	{6, 32, 58},
	{6, 34, 62},
	{6, 26, 46, 66},
	{6, 26, 48, 70},
	{6, 26, 50, 74},
	{6, 30, 54, 78},
	{6, 30, 56, 82},
	{6, 30, 58, 86},
	{6, 34, 62, 90},
	{6, 28, 50, 72, 94},
	{6, 26, 50, 74, 98},
	{6, 30, 54, 78, 102},
	{6, 28, 54, 80, 106},
	{6, 32, 58, 84, 110},
	{6, 30, 58, 86, 114},
	{6, 34, 62, 90, 118},
	{6, 26, 50, 74, 98, 122},
	{6, 30, 54, 78, 102, 126},
	{6, 26, 52, 78, 104, 130},
	{6, 30, 56, 82, 108, 134},
	{6, 34, 60, 86, 112, 138},
	{6, 30, 58, 86, 114, 142},
	{6, 34, 62, 90, 118, 146},
	{6, 30, 54, 78, 102, 126, 150},
	{6, 24, 50, 76, 102, 128, 154},
	{6, 28, 54, 80, 106, 132, 158},
	{6, 32, 58, 84, 110, 136, 162},
	{6, 26, 54, 82, 110, 138, 166},
	{6, 30, 58, 86, 114, 142, 170},
}

// BCH generator polynomials and the format-info mask (QRUtil constants).
const (
	g15     = 1<<10 | 1<<8 | 1<<5 | 1<<4 | 1<<2 | 1<<1 | 1<<0
	g18     = 1<<12 | 1<<11 | 1<<10 | 1<<9 | 1<<8 | 1<<5 | 1<<2 | 1<<0
	g15Mask = 1<<14 | 1<<12 | 1<<10 | 1<<4 | 1<<1
)

// Penalty weights (QRUtil DEMERIT_POINTS_*).
const (
	demeritPoints1 = 3
	demeritPoints2 = 3
	demeritPoints3 = 40
	demeritPoints4 = 10
)

// archBits mirrors QRUtil::ARCH_BITS. Ruby computes 1.size * 8; on the 64-bit
// interpreters this org targets that is 64, so rszf uses 64 for byte-identical
// results with the oracle. (Go integers are always 64-bit here.)
const archBits = 64

// bitsForMode is QRUtil::BITS_FOR_MODE, keyed by Mode.
var bitsForMode = map[Mode][3]int{
	ModeNumeric:      {10, 12, 14},
	ModeAlphanumeric: {9, 11, 13},
	ModeByte8bit:     {8, 16, 16},
}

// qrMaxSize is QRUtil.max_size (the number of supported versions, 40).
func qrMaxSize() int { return len(patternPositionTable) }

// getBCHFormatInfo is QRUtil.get_bch_format_info.
func getBCHFormatInfo(data int) int {
	d := data << 10
	for getBCHDigit(d)-getBCHDigit(g15) >= 0 {
		d ^= g15 << uint(getBCHDigit(d)-getBCHDigit(g15))
	}
	return ((data << 10) | d) ^ g15Mask
}

// rszf is QRUtil.rszf (right-shift zero-fill) using archBits.
func rszf(num, count int) int {
	return (num >> uint(count)) & ((1 << uint(archBits-count)) - 1)
}

// getBCHVersion is QRUtil.get_bch_version.
func getBCHVersion(data int) int {
	d := data << 12
	for getBCHDigit(d)-getBCHDigit(g18) >= 0 {
		d ^= g18 << uint(getBCHDigit(d)-getBCHDigit(g18))
	}
	return (data << 12) | d
}

// getBCHDigit is QRUtil.get_bch_digit.
func getBCHDigit(data int) int {
	digit := 0
	for data != 0 {
		digit++
		data = rszf(data, 1)
	}
	return digit
}

// getPatternPositions is QRUtil.get_pattern_positions.
func getPatternPositions(version int) []int {
	return patternPositionTable[version-1]
}

// maskComputations is QRMASKCOMPUTATIONS: the eight mask predicates.
var maskComputations = [8]func(i, j int) bool{
	func(i, j int) bool { return (i+j)%2 == 0 },
	func(i, j int) bool { return i%2 == 0 },
	func(i, j int) bool { return j%3 == 0 },
	func(i, j int) bool { return (i+j)%3 == 0 },
	func(i, j int) bool { return (i/2+j/3)%2 == 0 },
	func(i, j int) bool { return (i*j)%2+(i*j)%3 == 0 },
	func(i, j int) bool { return ((i*j)%2+(i*j)%3)%2 == 0 },
	func(i, j int) bool { return ((i*j)%3+(i+j)%2)%2 == 0 },
}

// getMask is QRUtil.get_mask.
func getMask(maskPattern, i, j int) bool {
	return maskComputations[maskPattern](i, j)
}

// getErrorCorrectPolynomial is QRUtil.get_error_correct_polynomial.
func getErrorCorrectPolynomial(errorCorrectLength int) *polynomial {
	a := newPolynomial([]int{1}, 0)
	for i := 0; i < errorCorrectLength; i++ {
		a = a.multiply(newPolynomial([]int{1, gexp(i)}, 0))
	}
	return a
}

// getLengthInBits is QRUtil.get_length_in_bits.
func getLengthInBits(mode Mode, version int) int {
	macro := 0
	switch {
	case version >= 1 && version <= 9:
		macro = 0
	case version <= 26:
		macro = 1
	default:
		macro = 2
	}
	return bitsForMode[mode][macro]
}

// getLostPoints is QRUtil.get_lost_points.
func getLostPoints(modules [][]bool) float64 {
	pts := 0.0
	pts += float64(demeritPoints1SameColor(modules))
	pts += float64(demeritPoints2FullBlocks(modules))
	pts += float64(demeritPoints3DangerousPatterns(modules))
	pts += demeritPoints4DarkRatio(modules)
	return pts
}

// demeritPoints1SameColor is QRUtil.demerit_points_1_same_color.
func demeritPoints1SameColor(modules [][]bool) int {
	pts := 0
	moduleCount := len(modules)
	maxIndex := moduleCount - 1
	for row := 0; row < moduleCount; row++ {
		modulesRow := modules[row]
		for col := 0; col < moduleCount; col++ {
			sameCount := 0
			dark := modulesRow[col]
			if row > 0 {
				rowAbove := modules[row-1]
				if col > 0 && dark == rowAbove[col-1] {
					sameCount++
				}
				if dark == rowAbove[col] {
					sameCount++
				}
				if col < maxIndex && dark == rowAbove[col+1] {
					sameCount++
				}
			}
			if col > 0 && dark == modulesRow[col-1] {
				sameCount++
			}
			if col < maxIndex && dark == modulesRow[col+1] {
				sameCount++
			}
			if row < maxIndex {
				rowBelow := modules[row+1]
				if col > 0 && dark == rowBelow[col-1] {
					sameCount++
				}
				if dark == rowBelow[col] {
					sameCount++
				}
				if col < maxIndex && dark == rowBelow[col+1] {
					sameCount++
				}
			}
			if sameCount > 5 {
				pts += demeritPoints1 + sameCount - 5
			}
		}
	}
	return pts
}

// demeritPoints2FullBlocks is QRUtil.demerit_points_2_full_blocks.
func demeritPoints2FullBlocks(modules [][]bool) int {
	pts := 0
	moduleCount := len(modules)
	maxRow := moduleCount - 1
	for row := 0; row < maxRow; row++ {
		rowCurr := modules[row]
		rowNext := modules[row+1]
		for col := 0; col < maxRow; col++ {
			val := rowCurr[col]
			if val == rowNext[col] && val == rowCurr[col+1] && val == rowNext[col+1] {
				pts += demeritPoints2
			}
		}
	}
	return pts
}

// demeritPoints3DangerousPatterns is QRUtil.demerit_points_3_dangerous_patterns.
func demeritPoints3DangerousPatterns(modules [][]bool) int {
	pts := 0
	moduleCount := len(modules)
	patternLen := 7
	maxStart := moduleCount - patternLen + 1
	for _, row := range modules {
		for col := 0; col < maxStart; col++ {
			if row[col] && !row[col+1] && row[col+2] &&
				row[col+3] && row[col+4] && !row[col+5] && row[col+6] {
				pts += demeritPoints3
			}
		}
	}
	for col := 0; col < moduleCount; col++ {
		for row := 0; row < maxStart; row++ {
			if modules[row][col] && !modules[row+1][col] && modules[row+2][col] &&
				modules[row+3][col] && modules[row+4][col] && !modules[row+5][col] && modules[row+6][col] {
				pts += demeritPoints3
			}
		}
	}
	return pts
}

// demeritPoints4DarkRatio is QRUtil.demerit_points_4_dark_ratio.
func demeritPoints4DarkRatio(modules [][]bool) float64 {
	darkCount := 0
	for _, col := range modules {
		for _, v := range col {
			if v {
				darkCount++
			}
		}
	}
	ratio := float64(darkCount) / float64(len(modules)*len(modules))
	// Ruby: ratio is a Float so (100*ratio-50).abs / 5 is float division (no
	// flooring), then * DEMERIT_POINTS_4.
	ratioDelta := absFloat(100*ratio-50) / 5
	return ratioDelta * demeritPoints4
}

// absFloat is the small float absolute value used by the dark-ratio penalty.
func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// numberLength is QRNumeric::NUMBER_LENGTH lookup for a chunk length.
func numberLength(length int) int {
	v, _ := numberLengthOK(length)
	return v
}

// numberLengthOK returns the numeric chunk bit length and whether it is defined.
func numberLengthOK(length int) (int, bool) {
	switch length {
	case 3:
		return 10, true
	case 2:
		return 7, true
	case 1:
		return 4, true
	default:
		return 0, false
	}
}
