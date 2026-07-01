// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package rqrcode is a pure-Go (CGO-free) faithful reimplementation of Ruby's
// rqrcode_core / rqrcode gems (MRI 4.0.5). It generates the QR-code module
// matrix deterministically and renders it (SVG / ANSI / HTML / PNG) exactly as
// the gems do, so a host such as go-embedded-ruby can bind RQRCode::QRCode
// without any Ruby runtime or a C QR library.
//
// # Value model
//
// The generated matrix is a [][]bool where true is a dark module. A QRCode is
// built with [New]; renderers are methods that mirror rqrcode's `as_svg`,
// `as_ansi`, `as_html` and `as_png`. Everything here is fully deterministic
// given input + version (size) + level + mode, so it matches the gems'
// `to_a` / module layout module-for-module.
package rqrcode

import (
	"errors"
	"fmt"
	"strings"
)

// Level is a QR error-correction level (rqrcode's :l/:m/:q/:h symbols).
type Level int

// The four error-correction levels, matching RQRCodeCore::QRERRORCORRECTLEVEL.
const (
	LevelM Level = 0 // 15% recovery (Psych-style value m => 0)
	LevelL Level = 1 // 7% recovery
	LevelH Level = 2 // 30% recovery
	LevelQ Level = 3 // 25% recovery
)

// Mode is a QR encoding mode.
type Mode int

// The encoding modes matching RQRCodeCore::QRMODE.
const (
	ModeNumeric      Mode = 1 << 0 // 1
	ModeAlphanumeric Mode = 1 << 1 // 2
	ModeByte8bit     Mode = 1 << 2 // 4
)

// Sentinel errors mirroring the gem's QRCodeArgumentError / QRCodeRunTimeError.
var (
	// ErrArgument corresponds to RQRCodeCore::QRCodeArgumentError.
	ErrArgument = errors.New("rqrcode: argument error")
	// ErrRunTime corresponds to RQRCodeCore::QRCodeRunTimeError.
	ErrRunTime = errors.New("rqrcode: runtime error")
)

const (
	qrPositionPatternLength = (7+1)*2 + 1
	qrFormatInfoLength      = 15
)

// levelFromSymbol maps a rqrcode level symbol (:l/:m/:q/:h, any case) to a Level.
func levelFromSymbol(s string) (Level, error) {
	switch strings.ToLower(s) {
	case "l":
		return LevelL, nil
	case "m":
		return LevelM, nil
	case "q":
		return LevelQ, nil
	case "h":
		return LevelH, nil
	default:
		return 0, fmt.Errorf("%w: Unknown error correction level `:%s`", ErrArgument, s)
	}
}

// Symbol returns the rqrcode level symbol name ("l"/"m"/"q"/"h"), matching
// RQRCodeCore::QRCode#error_correction_level's key.
func (l Level) Symbol() string {
	switch l {
	case LevelL:
		return "l"
	case LevelM:
		return "m"
	case LevelQ:
		return "q"
	default:
		return "h"
	}
}

// modeFromSymbol maps a rqrcode mode symbol to a Mode, or ok=false if empty/unknown.
func modeFromSymbol(s string) (Mode, bool) {
	switch strings.ToLower(s) {
	case "number":
		return ModeNumeric, true
	case "alphanumeric":
		return ModeAlphanumeric, true
	case "byte_8bit":
		return ModeByte8bit, true
	default:
		return 0, false
	}
}

// Options configures New, mirroring RQRCodeCore::QRCode.new's option hash.
type Options struct {
	// Size is the QR version (1..40). Zero means auto-select the smallest that fits.
	Size int
	// MaxSize caps auto-selection (default qrMaxSize, i.e. 40).
	MaxSize int
	// Level is the error-correction level; the zero value LevelM matches Ruby's
	// numeric m=0, so callers should use WithLevel / set it explicitly to get :h.
	Level Level
	// levelSet records whether Level was chosen (so the default can be :h like the gem).
	levelSet bool
	// Mode forces an encoding mode; zero means auto-detect from the data.
	Mode Mode
	// modeSet records whether Mode was chosen.
	modeSet bool
}

// Option mutates Options, matching rqrcode's keyword arguments.
type Option func(*Options)

// WithSize sets the QR version (:size). 0 keeps auto-selection.
func WithSize(v int) Option { return func(o *Options) { o.Size = v } }

// WithMaxSize sets the maximum auto-selected version (:max_size).
func WithMaxSize(v int) Option { return func(o *Options) { o.MaxSize = v } }

// WithLevel sets the error-correction level (:level).
func WithLevel(l Level) Option { return func(o *Options) { o.Level = l; o.levelSet = true } }

// WithLevelSymbol sets the level from a symbol string (:l/:m/:q/:h).
func WithLevelSymbol(s string) Option {
	return func(o *Options) {
		if l, err := levelFromSymbol(s); err == nil {
			o.Level = l
			o.levelSet = true
		}
	}
}

// WithMode forces an encoding mode (:mode).
func WithMode(m Mode) Option { return func(o *Options) { o.Mode = m; o.modeSet = true } }

// WithModeSymbol forces a mode from a symbol (:number/:alphanumeric/:byte_8bit).
func WithModeSymbol(s string) Option {
	return func(o *Options) {
		if m, ok := modeFromSymbol(s); ok {
			o.Mode = m
			o.modeSet = true
		}
	}
}

// Segment is one piece of multi-segment data (rqrcode's {data:, mode:} hash).
type Segment struct {
	Data string
	// Mode forces this segment's mode; zero auto-detects.
	Mode    Mode
	modeSet bool
}

// Seg builds a Segment with an explicit mode.
func Seg(data string, mode Mode) Segment { return Segment{Data: data, Mode: mode, modeSet: true} }

// SegAuto builds a Segment whose mode is auto-detected.
func SegAuto(data string) Segment { return Segment{Data: data} }

// QRCode is a generated QR code: the module matrix plus its metadata. It mirrors
// RQRCodeCore::QRCode's public attributes (modules, module_count, version).
type QRCode struct {
	Modules     [][]bool // the dark/light module matrix (Modules[row][col])
	ModuleCount int      // side length in modules
	Version     int      // QR version (1..40)
	level       Level
	segments    []*segment
	dataList    dataWriter
	dataCache   []int
	common      [][]bool
	// setMask tracks which modules have been assigned (Ruby uses nil vs bool for
	// the "already placed" check; Go bool cannot be nil, so we keep this parallel
	// matrix). commonSet is its snapshot after the fixed patterns are laid.
	setMask   [][]bool
	commonSet [][]bool
}

// segment is the internal parsed form of a Segment (RQRCodeCore::QRSegment).
type segment struct {
	data string
	mode Mode
}

// New builds a QR code from a string, matching RQRCode::QRCode.new(string, ...)
// and RQRCodeCore::QRCode.new. The default level is :h (as in the gems).
func New(data string, opts ...Option) (*QRCode, error) {
	return newFromSegments([]Segment{{Data: data}}, false, opts...)
}

// NewMulti builds a multi-segment QR code, matching passing an array of
// {data:, mode:} hashes to RQRCodeCore::QRCode.new.
func NewMulti(segs []Segment, opts ...Option) (*QRCode, error) {
	if len(segs) == 0 {
		return nil, fmt.Errorf("%w: data must be a String, QRSegment, or an Array", ErrArgument)
	}
	return newFromSegments(segs, true, opts...)
}

// newFromSegments is the shared constructor. multi selects QRMulti writing.
func newFromSegments(segs []Segment, multi bool, opts ...Option) (*QRCode, error) {
	// Default to level :h (as the gem does); WithLevel* overrides it.
	o := &Options{Level: LevelH, levelSet: true}
	for _, opt := range opts {
		opt(o)
	}
	maxSize := o.MaxSize
	if maxSize == 0 {
		maxSize = qrMaxSize()
	}

	parsed := make([]*segment, len(segs))
	for i, s := range segs {
		var forced Mode
		set := s.modeSet
		if set {
			forced = s.Mode
		}
		// For single-segment New, honour the top-level Options.Mode.
		if !multi && i == 0 && o.modeSet {
			forced = o.Mode
			set = true
		}
		ps, err := newSegment(s.Data, forced, set)
		if err != nil {
			return nil, err
		}
		parsed[i] = ps
	}

	q := &QRCode{level: o.Level, segments: parsed}

	size := o.Size
	if size == 0 {
		v, err := q.minimumVersion(maxSize)
		if err != nil {
			return nil, err
		}
		size = v
	}
	if size > maxSize {
		return nil, fmt.Errorf("%w: Given size greater than maximum possible size of %d", ErrArgument, qrMaxSize())
	}
	if size < 1 || size > 40 {
		return nil, fmt.Errorf("%w: bad version %d", ErrRunTime, size)
	}

	q.Version = size
	q.ModuleCount = q.Version*4 + qrPositionPatternLength
	q.Modules = make([][]bool, q.ModuleCount)

	if multi {
		q.dataList = &qrMulti{segs: parsed}
	} else {
		q.dataList = parsed[0].writer()
	}

	if err := q.make(); err != nil {
		return nil, err
	}
	return q, nil
}

// newSegment builds a parsed segment, choosing its mode like QRSegment#initialize.
func newSegment(data string, forced Mode, forcedSet bool) (*segment, error) {
	mode := forced
	if !forcedSet || forced == 0 {
		switch {
		case validNumeric(data):
			mode = ModeNumeric
		case validAlphanumeric(data):
			mode = ModeAlphanumeric
		default:
			mode = ModeByte8bit
		}
	}
	// Validate a forced numeric/alphanumeric mode as the writers would.
	switch mode {
	case ModeNumeric:
		if !validNumeric(data) {
			return nil, fmt.Errorf("%w: Not a numeric string `%s`", ErrArgument, data)
		}
	case ModeAlphanumeric:
		if !validAlphanumeric(data) {
			return nil, fmt.Errorf("%w: Not a alpha numeric uppercase string `%s`", ErrArgument, data)
		}
	}
	return &segment{data: data, mode: mode}, nil
}

// size returns the bit length this segment contributes at a given version
// (RQRCodeCore::QRSegment#size).
func (s *segment) size(version int) int {
	return 4 + s.headerSize(version) + s.contentSize()
}

// headerSize is the character-count-indicator bit length (QRSegment#header_size).
func (s *segment) headerSize(version int) int {
	return getLengthInBits(s.mode, version)
}

// contentSize is the payload bit length (QRSegment#content_size).
func (s *segment) contentSize() int {
	dataLength := len(s.data) // bytesize
	var chunkSize, bitLength, extra int
	switch s.mode {
	case ModeNumeric:
		chunkSize = 3
		bitLength = numberLength(3) // 10
		if v, ok := numberLengthOK(dataLength % 3); ok {
			extra = v
		} else {
			extra = 0
		}
	case ModeAlphanumeric:
		chunkSize, bitLength, extra = 2, 11, 6
	default: // ModeByte8bit
		chunkSize, bitLength, extra = 1, 8, 0
	}
	rem := 0
	if dataLength%chunkSize != 0 {
		rem = extra
	}
	return (dataLength/chunkSize)*bitLength + rem
}

// writer returns the dataWriter for this segment's mode (QRSegment#writer).
func (s *segment) writer() dataWriter {
	switch s.mode {
	case ModeNumeric:
		return &qrNumeric{data: s.data}
	case ModeAlphanumeric:
		return &qrAlphanumeric{data: s.data}
	default:
		return &qr8bitByte{data: s.data}
	}
}

// minimumVersion mirrors RQRCodeCore::QRCode#minimum_version.
func (q *QRCode) minimumVersion(limit int) (int, error) {
	for version := 1; ; version++ {
		if version > limit {
			return 0, fmt.Errorf("%w: Data length exceed maximum capacity of version %d", ErrRunTime, limit)
		}
		maxSizeBits := qrMaxBits[q.level][version-1]
		sizeBits := 0
		for _, s := range q.segments {
			sizeBits += s.size(version)
		}
		if sizeBits < maxSizeBits {
			return version, nil
		}
	}
}

// Checked reports whether the module at (row, col) is dark, matching
// RQRCodeCore::QRCode#checked? (note the row, col order).
func (q *QRCode) Checked(row, col int) (bool, error) {
	if row < 0 || row > q.ModuleCount-1 || col < 0 || col > q.ModuleCount-1 {
		return false, fmt.Errorf("%w: Invalid row/column pair: %d, %d", ErrRunTime, row, col)
	}
	return q.Modules[row][col], nil
}

// Mode returns the effective mode of the (first) data writer, mirroring
// RQRCodeCore::QRCode#mode.
func (q *QRCode) Mode() Mode {
	switch q.dataList.(type) {
	case *qrNumeric:
		return ModeNumeric
	case *qrAlphanumeric:
		return ModeAlphanumeric
	default:
		return ModeByte8bit
	}
}

// ErrorCorrectionLevel returns the QR code's level.
func (q *QRCode) ErrorCorrectionLevel() Level { return q.level }

// make prepares patterns and picks the best mask (RQRCodeCore::QRCode#make).
func (q *QRCode) make() error {
	q.prepareCommonPatterns()
	best, err := q.getBestMaskPattern()
	if err != nil {
		return err
	}
	return q.makeImpl(false, best)
}

// prepareCommonPatterns places the fixed patterns shared by every mask attempt.
func (q *QRCode) prepareCommonPatterns() {
	for i := range q.Modules {
		q.Modules[i] = make([]bool, q.ModuleCount)
	}
	// Track which cells are set (Ruby uses nil vs bool; Go bool can't be nil, so
	// keep a parallel "set" matrix for the adjust-pattern nil check).
	q.setMask = make([][]bool, q.ModuleCount)
	for i := range q.setMask {
		q.setMask[i] = make([]bool, q.ModuleCount)
	}

	q.placePositionProbePattern(0, 0)
	q.placePositionProbePattern(q.ModuleCount-7, 0)
	q.placePositionProbePattern(0, q.ModuleCount-7)
	q.placePositionAdjustPattern()
	q.placeTimingPattern()

	q.common = cloneMatrix(q.Modules)
	q.commonSet = cloneMatrix(q.setMask)
}

// makeImpl builds the matrix for a given mask (RQRCodeCore::QRCode#make_impl).
func (q *QRCode) makeImpl(test bool, maskPattern int) error {
	q.Modules = cloneMatrix(q.common)
	q.setMask = cloneMatrix(q.commonSet)

	q.placeFormatInfo(test, maskPattern)
	if q.Version >= 7 {
		q.placeVersionInfo(test)
	}
	if q.dataCache == nil {
		dc, err := createData(q.Version, q.level, q.dataList)
		if err != nil {
			return err
		}
		q.dataCache = dc
	}
	q.mapData(q.dataCache, maskPattern)
	return nil
}

// placePositionProbePattern places one finder pattern (place_position_probe_pattern).
func (q *QRCode) placePositionProbePattern(row, col int) {
	for r := -1; r <= 7; r++ {
		if row+r < 0 || row+r > q.ModuleCount-1 {
			continue
		}
		for c := -1; c <= 7; c++ {
			if col+c < 0 || col+c > q.ModuleCount-1 {
				continue
			}
			isVert := r >= 0 && r <= 6 && (c == 0 || c == 6)
			isHoriz := c >= 0 && c <= 6 && (r == 0 || r == 6)
			isSquare := r >= 2 && r <= 4 && c >= 2 && c <= 4
			part := isVert || isHoriz || isSquare
			q.Modules[row+r][col+c] = part
			q.setMask[row+r][col+c] = true
		}
	}
}

// getBestMaskPattern scores all 8 masks and returns the lowest-penalty one.
func (q *QRCode) getBestMaskPattern() (int, error) {
	minLost := 0.0
	pattern := 0
	for i := 0; i < 8; i++ {
		if err := q.makeImpl(true, i); err != nil {
			return 0, err
		}
		lost := getLostPoints(q.Modules)
		if i == 0 || minLost > lost {
			minLost = lost
			pattern = i
		}
	}
	return pattern, nil
}

// placeTimingPattern lays the timing rows/cols (place_timing_pattern).
func (q *QRCode) placeTimingPattern() {
	for i := 8; i < q.ModuleCount-8; i++ {
		v := i%2 == 0
		q.Modules[i][6] = v
		q.Modules[6][i] = v
		q.setMask[i][6] = true
		q.setMask[6][i] = true
	}
}

// placePositionAdjustPattern lays the alignment patterns (place_position_adjust_pattern).
func (q *QRCode) placePositionAdjustPattern() {
	positions := getPatternPositions(q.Version)
	for _, row := range positions {
		for _, col := range positions {
			if q.setMask[row][col] {
				continue // Ruby: next unless @modules[row][col].nil?
			}
			for r := -2; r <= 2; r++ {
				for c := -2; c <= 2; c++ {
					part := abs(r) == 2 || abs(c) == 2 || (r == 0 && c == 0)
					q.Modules[row+r][col+c] = part
					q.setMask[row+r][col+c] = true
				}
			}
		}
	}
}

// placeVersionInfo writes the version bits for v>=7 (place_version_info).
func (q *QRCode) placeVersionInfo(test bool) {
	bits := getBCHVersion(q.Version)
	for i := 0; i < 18; i++ {
		mod := !test && ((bits>>uint(i))&1) == 1
		a := i / 3
		b := i%3 + q.ModuleCount - 8 - 3
		q.Modules[a][b] = mod
		q.Modules[b][a] = mod
		q.setMask[a][b] = true
		q.setMask[b][a] = true
	}
}

// placeFormatInfo writes the 15-bit format info (place_format_info).
func (q *QRCode) placeFormatInfo(test bool, maskPattern int) {
	data := int(q.level)<<3 | maskPattern
	bits := getBCHFormatInfo(data)

	for i := 0; i < qrFormatInfoLength; i++ {
		mod := !test && ((bits>>uint(i))&1) == 1

		var row int
		switch {
		case i < 6:
			row = i
		case i < 8:
			row = i + 1
		default:
			row = q.ModuleCount - 15 + i
		}
		q.Modules[row][8] = mod
		q.setMask[row][8] = true

		var col int
		switch {
		case i < 8:
			col = q.ModuleCount - i - 1
		case i < 9:
			col = 15 - i - 1 + 1
		default:
			col = 15 - i - 1
		}
		q.Modules[8][col] = mod
		q.setMask[8][col] = true
	}

	q.Modules[q.ModuleCount-8][8] = !test
	q.setMask[q.ModuleCount-8][8] = true
}

// mapData places the encoded data bits under the given mask (map_data).
func (q *QRCode) mapData(data []int, maskPattern int) {
	inc := -1
	row := q.ModuleCount - 1
	bitIndex := 7
	byteIndex := 0

	// Ruby iterates `(@module_count-1).step(1, -2)` and then locally does
	// `col -= 1 if col <= 6`; the step counter itself is unaffected by that
	// adjustment, so we keep the loop variable separate from the adjusted column.
	for base := q.ModuleCount - 1; base >= 1; base -= 2 {
		col := base
		if col <= 6 {
			col--
		}
		for {
			for c := 0; c < 2; c++ {
				if !q.setMask[row][col-c] {
					dark := false
					if byteIndex < len(data) {
						dark = (rszf(data[byteIndex], bitIndex) & 1) == 1
					}
					mask := getMask(maskPattern, row, col-c)
					if mask {
						dark = !dark
					}
					q.Modules[row][col-c] = dark
					q.setMask[row][col-c] = true
					bitIndex--
					if bitIndex == -1 {
						byteIndex++
						bitIndex = 7
					}
				}
			}
			row += inc
			if row < 0 || row >= q.ModuleCount {
				row -= inc
				inc = -inc
				break
			}
		}
	}
}

// abs is the small integer absolute value used by the pattern placement.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// cloneMatrix deep-copies a bool matrix (Ruby's map(&:clone)).
func cloneMatrix(m [][]bool) [][]bool {
	out := make([][]bool, len(m))
	for i := range m {
		row := make([]bool, len(m[i]))
		copy(row, m[i])
		out[i] = row
	}
	return out
}
