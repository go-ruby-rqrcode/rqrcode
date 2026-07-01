// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import "fmt"

// rsBlock is RQRCodeCore::QRRSBlock.
type rsBlock struct {
	totalCount int
	dataCount  int
}

// rsBlockTable is RQRCodeCore::QRRSBlock::RS_BLOCK_TABLE, four rows (L,M,Q,H) per
// version 1..40.
var rsBlockTable = [][]int{
	{1, 26, 19}, {1, 26, 16}, {1, 26, 13}, {1, 26, 9},
	{1, 44, 34}, {1, 44, 28}, {1, 44, 22}, {1, 44, 16},
	{1, 70, 55}, {1, 70, 44}, {2, 35, 17}, {2, 35, 13},
	{1, 100, 80}, {2, 50, 32}, {2, 50, 24}, {4, 25, 9},
	{1, 134, 108}, {2, 67, 43}, {2, 33, 15, 2, 34, 16}, {2, 33, 11, 2, 34, 12},
	{2, 86, 68}, {4, 43, 27}, {4, 43, 19}, {4, 43, 15},
	{2, 98, 78}, {4, 49, 31}, {2, 32, 14, 4, 33, 15}, {4, 39, 13, 1, 40, 14},
	{2, 121, 97}, {2, 60, 38, 2, 61, 39}, {4, 40, 18, 2, 41, 19}, {4, 40, 14, 2, 41, 15},
	{2, 146, 116}, {3, 58, 36, 2, 59, 37}, {4, 36, 16, 4, 37, 17}, {4, 36, 12, 4, 37, 13},
	{2, 86, 68, 2, 87, 69}, {4, 69, 43, 1, 70, 44}, {6, 43, 19, 2, 44, 20}, {6, 43, 15, 2, 44, 16},
	{4, 101, 81}, {1, 80, 50, 4, 81, 51}, {4, 50, 22, 4, 51, 23}, {3, 36, 12, 8, 37, 13},
	{2, 116, 92, 2, 117, 93}, {6, 58, 36, 2, 59, 37}, {4, 46, 20, 6, 47, 21}, {7, 42, 14, 4, 43, 15},
	{4, 133, 107}, {8, 59, 37, 1, 60, 38}, {8, 44, 20, 4, 45, 21}, {12, 33, 11, 4, 34, 12},
	{3, 145, 115, 1, 146, 116}, {4, 64, 40, 5, 65, 41}, {11, 36, 16, 5, 37, 17}, {11, 36, 12, 5, 37, 13},
	{5, 109, 87, 1, 110, 88}, {5, 65, 41, 5, 66, 42}, {5, 54, 24, 7, 55, 25}, {11, 36, 12, 7, 37, 13},
	{5, 122, 98, 1, 123, 99}, {7, 73, 45, 3, 74, 46}, {15, 43, 19, 2, 44, 20}, {3, 45, 15, 13, 46, 16},
	{1, 135, 107, 5, 136, 108}, {10, 74, 46, 1, 75, 47}, {1, 50, 22, 15, 51, 23}, {2, 42, 14, 17, 43, 15},
	{5, 150, 120, 1, 151, 121}, {9, 69, 43, 4, 70, 44}, {17, 50, 22, 1, 51, 23}, {2, 42, 14, 19, 43, 15},
	{3, 141, 113, 4, 142, 114}, {3, 70, 44, 11, 71, 45}, {17, 47, 21, 4, 48, 22}, {9, 39, 13, 16, 40, 14},
	{3, 135, 107, 5, 136, 108}, {3, 67, 41, 13, 68, 42}, {15, 54, 24, 5, 55, 25}, {15, 43, 15, 10, 44, 16},
	{4, 144, 116, 4, 145, 117}, {17, 68, 42}, {17, 50, 22, 6, 51, 23}, {19, 46, 16, 6, 47, 17},
	{2, 139, 111, 7, 140, 112}, {17, 74, 46}, {7, 54, 24, 16, 55, 25}, {34, 37, 13},
	{4, 151, 121, 5, 152, 122}, {4, 75, 47, 14, 76, 48}, {11, 54, 24, 14, 55, 25}, {16, 45, 15, 14, 46, 16},
	{6, 147, 117, 4, 148, 118}, {6, 73, 45, 14, 74, 46}, {11, 54, 24, 16, 55, 25}, {30, 46, 16, 2, 47, 17},
	{8, 132, 106, 4, 133, 107}, {8, 75, 47, 13, 76, 48}, {7, 54, 24, 22, 55, 25}, {22, 45, 15, 13, 46, 16},
	{10, 142, 114, 2, 143, 115}, {19, 74, 46, 4, 75, 47}, {28, 50, 22, 6, 51, 23}, {33, 46, 16, 4, 47, 17},
	{8, 152, 122, 4, 153, 123}, {22, 73, 45, 3, 74, 46}, {8, 53, 23, 26, 54, 24}, {12, 45, 15, 28, 46, 16},
	{3, 147, 117, 10, 148, 118}, {3, 73, 45, 23, 74, 46}, {4, 54, 24, 31, 55, 25}, {11, 45, 15, 31, 46, 16},
	{7, 146, 116, 7, 147, 117}, {21, 73, 45, 7, 74, 46}, {1, 53, 23, 37, 54, 24}, {19, 45, 15, 26, 46, 16},
	{5, 145, 115, 10, 146, 116}, {19, 75, 47, 10, 76, 48}, {15, 54, 24, 25, 55, 25}, {23, 45, 15, 25, 46, 16},
	{13, 145, 115, 3, 146, 116}, {2, 74, 46, 29, 75, 47}, {42, 54, 24, 1, 55, 25}, {23, 45, 15, 28, 46, 16},
	{17, 145, 115}, {10, 74, 46, 23, 75, 47}, {10, 54, 24, 35, 55, 25}, {19, 45, 15, 35, 46, 16},
	{17, 145, 115, 1, 146, 116}, {14, 74, 46, 21, 75, 47}, {29, 54, 24, 19, 55, 25}, {11, 45, 15, 46, 46, 16},
	{13, 145, 115, 6, 146, 116}, {14, 74, 46, 23, 75, 47}, {44, 54, 24, 7, 55, 25}, {59, 46, 16, 1, 47, 17},
	{12, 151, 121, 7, 152, 122}, {12, 75, 47, 26, 76, 48}, {39, 54, 24, 14, 55, 25}, {22, 45, 15, 41, 46, 16},
	{6, 151, 121, 14, 152, 122}, {6, 75, 47, 34, 76, 48}, {46, 54, 24, 10, 55, 25}, {2, 45, 15, 64, 46, 16},
	{17, 152, 122, 4, 153, 123}, {29, 74, 46, 14, 75, 47}, {49, 54, 24, 10, 55, 25}, {24, 45, 15, 46, 46, 16},
	{4, 152, 122, 18, 153, 123}, {13, 74, 46, 32, 75, 47}, {48, 54, 24, 14, 55, 25}, {42, 45, 15, 32, 46, 16},
	{20, 147, 117, 4, 148, 118}, {40, 75, 47, 7, 76, 48}, {43, 54, 24, 22, 55, 25}, {10, 45, 15, 67, 46, 16},
	{19, 148, 118, 6, 149, 119}, {18, 75, 47, 31, 76, 48}, {34, 54, 24, 34, 55, 25}, {20, 45, 15, 61, 46, 16},
}

// rsBlockRowOffset maps a Level to the offset within a version's 4-row group.
// The gem indexes by QRERRORCORRECTLEVEL numeric value order l/m/q/h => rows
// 0/1/2/3, matched here.
func rsBlockRowOffset(level Level) int {
	switch level {
	case LevelL:
		return 0
	case LevelM:
		return 1
	case LevelQ:
		return 2
	default: // LevelH
		return 3
	}
}

// getRSBlockTable is QRRSBlock.get_rs_block_table.
func getRSBlockTable(version int, level Level) []int {
	return rsBlockTable[(version-1)*4+rsBlockRowOffset(level)]
}

// getRSBlocks is QRRSBlock.get_rs_blocks. The caller always passes a validated
// version (1..40), for which getRSBlockTable returns a non-nil row, so the gem's
// bad-rsblock guard cannot fire here.
func getRSBlocks(version int, level Level) []rsBlock {
	row := getRSBlockTable(version, level)
	length := len(row) / 3
	var list []rsBlock
	for i := 0; i < length; i++ {
		count := row[i*3+0]
		totalCount := row[i*3+1]
		dataCount := row[i*3+2]
		for j := 0; j < count; j++ {
			list = append(list, rsBlock{totalCount: totalCount, dataCount: dataCount})
		}
	}
	return list
}

// countMaxDataBits is QRCode.count_max_data_bits.
func countMaxDataBits(blocks []rsBlock) int {
	sum := 0
	for _, b := range blocks {
		sum += b.dataCount
	}
	return sum * 8
}

// createData is QRCode.create_data.
func createData(version int, level Level, dataList dataWriter) ([]int, error) {
	blocks := getRSBlocks(version, level)
	maxDataBits := countMaxDataBits(blocks)
	buffer := newBitBuffer(version)

	dataList.write(buffer)
	buffer.endOfMessage(maxDataBits)

	if buffer.lengthInBits() > maxDataBits {
		return nil, fmt.Errorf("%w: code length overflow. (%d>%d). (Try a larger size!)",
			ErrRunTime, buffer.lengthInBits(), maxDataBits)
	}
	buffer.padUntil(maxDataBits)
	return createBytes(buffer, blocks), nil
}

// createBytes is QRCode.create_bytes (data + EC interleaving).
func createBytes(buffer *bitBuffer, blocks []rsBlock) []int {
	offset := 0
	maxDCCount := 0
	maxECCount := 0
	dcdata := make([][]int, len(blocks))
	ecdata := make([][]int, len(blocks))

	for r, block := range blocks {
		dcCount := block.dataCount
		ecCount := block.totalCount - dcCount
		if dcCount > maxDCCount {
			maxDCCount = dcCount
		}
		if ecCount > maxECCount {
			maxECCount = ecCount
		}

		dcBlock := make([]int, dcCount)
		for i := 0; i < dcCount; i++ {
			dcBlock[i] = 0xff & buffer.buffer[i+offset]
		}
		dcdata[r] = dcBlock
		offset += dcCount

		rsPoly := getErrorCorrectPolynomial(ecCount)
		rawPoly := newPolynomial(dcdata[r], rsPoly.length()-1)
		modPoly := rawPoly.mod(rsPoly)

		ecBlock := make([]int, rsPoly.length()-1)
		for i := range ecBlock {
			modIndex := i + modPoly.length() - len(ecBlock)
			if modIndex >= 0 {
				ecBlock[i] = modPoly.get(modIndex)
			} else {
				ecBlock[i] = 0
			}
		}
		ecdata[r] = ecBlock
	}

	totalCodeCount := 0
	for _, block := range blocks {
		totalCodeCount += block.totalCount
	}

	data := make([]int, totalCodeCount)
	index := 0
	for i := 0; i < maxDCCount; i++ {
		for r := range blocks {
			if i < len(dcdata[r]) {
				data[index] = dcdata[r][i]
				index++
			}
		}
	}
	for i := 0; i < maxECCount; i++ {
		for r := range blocks {
			if i < len(ecdata[r]) {
				data[index] = ecdata[r][i]
				index++
			}
		}
	}
	return data
}
