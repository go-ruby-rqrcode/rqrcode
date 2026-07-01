// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"testing"
)

// TestNewDefaultsToLevelH verifies the default error-correction level is :h, as
// in the gem, and that the basic metadata is exposed.
func TestNewDefaultsToLevelH(t *testing.T) {
	q, err := New("hello world")
	if err != nil {
		t.Fatal(err)
	}
	if q.ErrorCorrectionLevel() != LevelH {
		t.Errorf("default level = %v, want LevelH", q.ErrorCorrectionLevel())
	}
	if q.Version != 2 || q.ModuleCount != 25 {
		t.Errorf("version/count = %d/%d, want 2/25", q.Version, q.ModuleCount)
	}
	if q.Mode() != ModeByte8bit {
		t.Errorf("mode = %v, want byte", q.Mode())
	}
	if got := LevelH.Symbol(); got != "h" {
		t.Errorf("Symbol = %q", got)
	}
}

// TestModeDetection checks automatic mode selection across data shapes.
func TestModeDetection(t *testing.T) {
	cases := []struct {
		data string
		want Mode
	}{
		{"12345", ModeNumeric},
		{"HELLO WORLD", ModeAlphanumeric},
		{"hello world", ModeByte8bit},
		{"", ModeNumeric}, // empty is valid numeric first in the gem's chain
	}
	for _, c := range cases {
		q, err := New(c.data)
		if err != nil {
			t.Fatalf("%q: %v", c.data, err)
		}
		if q.Mode() != c.want {
			t.Errorf("%q: mode = %v, want %v", c.data, q.Mode(), c.want)
		}
	}
}

// TestForcedMode exercises WithMode / WithModeSymbol and their symbol path.
func TestForcedMode(t *testing.T) {
	q, err := New("12345", WithMode(ModeAlphanumeric))
	if err != nil {
		t.Fatal(err)
	}
	if q.Mode() != ModeAlphanumeric {
		t.Errorf("forced mode = %v", q.Mode())
	}
	q2, err := New("HELLO", WithModeSymbol("alphanumeric"), WithLevel(LevelQ))
	if err != nil {
		t.Fatal(err)
	}
	if q2.Mode() != ModeAlphanumeric || q2.ErrorCorrectionLevel() != LevelQ {
		t.Errorf("mode/level = %v/%v", q2.Mode(), q2.ErrorCorrectionLevel())
	}
	// Unknown mode symbol is ignored (auto-detected instead).
	q3, err := New("hello", WithModeSymbol("bogus"))
	if err != nil {
		t.Fatal(err)
	}
	if q3.Mode() != ModeByte8bit {
		t.Errorf("bogus mode: %v", q3.Mode())
	}
	// number and byte_8bit symbols.
	if _, err := New("42", WithModeSymbol("number")); err != nil {
		t.Fatal(err)
	}
	if _, err := New("42", WithModeSymbol("byte_8bit")); err != nil {
		t.Fatal(err)
	}
}

// TestForcedModeValidationErrors covers the QRCodeArgumentError paths for a mode
// that does not match the data.
func TestForcedModeValidationErrors(t *testing.T) {
	if _, err := New("abc", WithMode(ModeNumeric)); !errors.Is(err, ErrArgument) {
		t.Errorf("numeric of non-numeric: %v", err)
	}
	if _, err := New("abc", WithMode(ModeAlphanumeric)); !errors.Is(err, ErrArgument) {
		t.Errorf("alphanumeric of invalid: %v", err)
	}
}

// TestLevels round-trips every level symbol and the typed constants.
func TestLevels(t *testing.T) {
	for _, s := range []string{"l", "m", "q", "h", "L", "M", "Q", "H"} {
		if _, err := New("test", WithLevelSymbol(s)); err != nil {
			t.Errorf("level %q: %v", s, err)
		}
	}
	for _, l := range []Level{LevelL, LevelM, LevelQ, LevelH} {
		q, err := New("test", WithLevel(l))
		if err != nil {
			t.Fatal(err)
		}
		if q.ErrorCorrectionLevel() != l {
			t.Errorf("level %v not round-tripped", l)
		}
		_ = l.Symbol()
	}
	// Unknown level symbol is ignored by WithLevelSymbol (falls back to :h default).
	if _, err := New("x", WithLevelSymbol("z")); err != nil {
		t.Fatal(err)
	}
	// The low-level parser reports the error for a bad symbol.
	if _, err := levelFromSymbol("z"); !errors.Is(err, ErrArgument) {
		t.Errorf("levelFromSymbol(z): %v", err)
	}
}

// TestExplicitSizeAndVersionInfo exercises pinned sizes including v>=7 (version
// info) and the maximum version.
func TestExplicitSizeAndVersionInfo(t *testing.T) {
	for _, v := range []int{1, 6, 7, 10, 40} {
		q, err := New("EXAMPLE", WithLevel(LevelL), WithSize(v))
		if err != nil {
			t.Fatalf("size %d: %v", v, err)
		}
		if q.Version != v {
			t.Errorf("version = %d, want %d", q.Version, v)
		}
	}
}

// TestSizeErrors covers the size/capacity error branches.
func TestSizeErrors(t *testing.T) {
	// Size greater than max_size.
	if _, err := New("x", WithSize(41), WithMaxSize(40)); !errors.Is(err, ErrArgument) {
		t.Errorf("oversize: %v", err)
	}
	// Explicit out-of-range version.
	if _, err := New("x", WithSize(99)); !errors.Is(err, ErrArgument) {
		t.Errorf("v99: %v", err)
	}
	// Data too large to ever fit within a tiny max_size => minimumVersion error.
	big := strings.Repeat("A", 200)
	if _, err := New(big, WithMaxSize(1)); !errors.Is(err, ErrRunTime) {
		t.Errorf("capacity: %v", err)
	}
	// A pinned size too small for the data => create_data overflow.
	if _, err := New(strings.Repeat("A", 100), WithLevel(LevelH), WithSize(1)); !errors.Is(err, ErrRunTime) {
		t.Errorf("overflow: %v", err)
	}
}

// TestMulti covers multi-segment construction, including auto and forced modes.
func TestMulti(t *testing.T) {
	q, err := NewMulti([]Segment{
		Seg("foo", ModeByte8bit),
		Seg("1234", ModeNumeric),
		SegAuto("BAR"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if q.ModuleCount < 21 {
		t.Errorf("count = %d", q.ModuleCount)
	}
	// Empty segment list is an argument error.
	if _, err := NewMulti(nil); !errors.Is(err, ErrArgument) {
		t.Errorf("empty multi: %v", err)
	}
	// A forced-invalid segment errors.
	if _, err := NewMulti([]Segment{Seg("abc", ModeNumeric)}); !errors.Is(err, ErrArgument) {
		t.Errorf("invalid seg: %v", err)
	}
}

// TestChecked covers checked? including the out-of-range runtime error.
func TestChecked(t *testing.T) {
	q, _ := New("hello world")
	if d, err := q.Checked(0, 0); err != nil || !d {
		t.Errorf("checked(0,0) = %v,%v", d, err)
	}
	for _, rc := range [][2]int{{-1, 0}, {0, -1}, {q.ModuleCount, 0}, {0, q.ModuleCount}} {
		if _, err := q.Checked(rc[0], rc[1]); !errors.Is(err, ErrRunTime) {
			t.Errorf("checked(%d,%d) should error", rc[0], rc[1])
		}
	}
}

// TestContentSizeNumericRemainders drives the numeric content_size chunk-length
// branches (data lengths with %3 == 0/1/2).
func TestContentSizeNumericRemainders(t *testing.T) {
	for _, d := range []string{"123", "1234", "12345", "123456"} {
		if _, err := New(d, WithMode(ModeNumeric)); err != nil {
			t.Fatalf("%q: %v", d, err)
		}
	}
	// Alphanumeric odd/even lengths (last-char 6-bit path).
	for _, d := range []string{"A", "AB", "ABC"} {
		if _, err := New(d, WithMode(ModeAlphanumeric)); err != nil {
			t.Fatalf("%q: %v", d, err)
		}
	}
}

// TestString covers to_s with and without a quiet zone.
func TestString(t *testing.T) {
	q, _ := New("A", WithLevel(LevelL))
	s := q.String()
	if !strings.Contains(s, "x") || strings.Contains(s, "\t") {
		t.Errorf("to_s: %q", s)
	}
	qz := q.ToString(StringOptions{QuietZoneSize: 3, Dark: "#", Light: "."})
	lines := strings.Split(qz, "\n")
	if len(lines) != q.ModuleCount+6 {
		t.Errorf("quiet-zone rows = %d, want %d", len(lines), q.ModuleCount+6)
	}
}

// TestSVGVariants exercises the rect and path renderers with a spread of options
// (offsets, named colors, viewbox, standalone toggles, fill).
func TestSVGVariants(t *testing.T) {
	q, _ := New("hello world")
	st := true
	stF := false
	outs := []string{
		q.AsSVG(SVGOptions{}),
		q.AsSVG(SVGOptions{Standalone: &st}),
		q.AsSVG(SVGOptions{Standalone: &stF}),
		q.AsSVG(SVGOptions{UsePath: true}),
		q.AsSVG(SVGOptions{Offset: 5}),
		q.AsSVG(SVGOptions{OffsetX: 2, OffsetXSet: true, OffsetY: 3, OffsetYSet: true}),
		q.AsSVG(SVGOptions{Viewbox: true}),
		q.AsSVG(SVGOptions{Fill: "fff"}),
		q.AsSVG(SVGOptions{Color: "red", ColorNamed: true, Fill: "white", FillNamed: true}),
		q.AsSVG(SVGOptions{ShapeRendering: "auto", SVGAttributes: [][2]string{{"data-x", "1"}}}),
	}
	for i, o := range outs {
		if !strings.Contains(o, "svg") && !strings.Contains(o, "path") && !strings.Contains(o, "rect") {
			t.Errorf("svg variant %d looks empty: %q", i, o[:min(40, len(o))])
		}
	}
	// The embeddable (standalone=false) output has no <?xml prolog.
	if strings.Contains(q.AsSVG(SVGOptions{Standalone: &stF}), "<?xml") {
		t.Error("embeddable svg should not have xml prolog")
	}
}

// TestANSIVariants exercises as_ansi defaults and options including quiet zone 0.
func TestANSIVariants(t *testing.T) {
	q, _ := New("HI", WithLevel(LevelL))
	if !strings.Contains(q.AsANSIDefault(), "\033[") {
		t.Error("ansi missing escapes")
	}
	// Zero quiet zone and custom fill.
	out := q.AsANSI(ANSIOptions{QuietZoneSize: 0, FillCharacter: "@@", Light: "\033[47m", Dark: "\033[40m"})
	if !strings.Contains(out, "@@") {
		t.Error("custom fill missing")
	}
}

// TestHTML covers as_html output shape.
func TestHTML(t *testing.T) {
	q, _ := New("A", WithLevel(LevelL))
	h := q.AsHTML()
	if !strings.HasPrefix(h, "<table>") || !strings.HasSuffix(h, "</table>") {
		t.Errorf("html: %q", h[:min(40, len(h))])
	}
	if !strings.Contains(h, `<td class="black">`) || !strings.Contains(h, `<td class="white">`) {
		t.Error("html missing cell classes")
	}
}

// TestPNG covers the pure-Go PNG renderer, including custom colors and geometry,
// and decodes the result to confirm dimensions and pixel values.
func TestPNG(t *testing.T) {
	q, _ := New("hello world", WithLevel(LevelL))
	data, err := q.AsPNG(PNGOptions{})
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	side := (q.ModuleCount + 8) * 6 // default border 4, module 6
	if b := img.Bounds(); b.Dx() != side || b.Dy() != side {
		t.Errorf("png size = %dx%d, want %d", b.Dx(), b.Dy(), side)
	}
	// A dark module (0,0 finder) must be foreground; a border pixel must be fill.
	darkX, darkY := 4*6, 4*6 // module (0,0) top-left inside the border
	r, g, b, _ := img.At(darkX+1, darkY+1).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Errorf("dark pixel = %d,%d,%d, want black", r>>8, g>>8, b>>8)
	}
	br, bg, bb, _ := img.At(0, 0).RGBA()
	if br>>8 != 255 || bg>>8 != 255 || bb>>8 != 255 {
		t.Errorf("border pixel = %d,%d,%d, want white", br>>8, bg>>8, bb>>8)
	}

	// Custom colors and geometry.
	data2, err := q.AsPNG(PNGOptions{
		ModulePxSize: 3, BorderModules: 1,
		Color: color.NRGBA{R: 10, G: 20, B: 30, A: 255},
		Fill:  color.NRGBA{R: 200, G: 210, B: 220, A: 255},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(data2)); err != nil {
		t.Fatalf("decode2: %v", err)
	}
	// Negative border is clamped to the default (exercises the guard).
	if _, err := q.AsPNG(PNGOptions{BorderModules: -5}); err != nil {
		t.Fatal(err)
	}
}

// TestGexpGlogWrap covers the wrap-around and low branches of the GF helpers.
func TestGexpGlogWrap(t *testing.T) {
	if gexp(-1) != gexp(254) {
		t.Error("gexp negative wrap")
	}
	if gexp(256) != gexp(1) {
		t.Error("gexp high wrap")
	}
	if glog(1) != 0 {
		t.Errorf("glog(1) = %d", glog(1))
	}
	if got := recoverPanic(func() { glog(0) }); got == nil {
		t.Error("glog(0) should panic")
	}
}

// TestNewPolynomialEmptyPanics covers the empty-input guard of newPolynomial.
func TestNewPolynomialEmptyPanics(t *testing.T) {
	if got := recoverPanic(func() { newPolynomial(nil, 0) }); got == nil {
		t.Error("newPolynomial(nil) should panic")
	}
	// Leading-zero stripping path.
	p := newPolynomial([]int{0, 0, 5, 3}, 2)
	if p.length() != 4 { // (4-2)+2
		t.Errorf("length = %d", p.length())
	}
	if p.get(0) != 5 {
		t.Errorf("get(0) = %d", p.get(0))
	}
}

// TestGetRSBlocksAllVersions builds the RS-block layout for every version/level.
func TestGetRSBlocksAllVersions(t *testing.T) {
	for v := 1; v <= 40; v++ {
		for _, l := range []Level{LevelL, LevelM, LevelQ, LevelH} {
			if blocks := getRSBlocks(v, l); len(blocks) == 0 {
				t.Fatalf("v%d/%v: no blocks", v, l)
			}
		}
	}
}

// TestSizeAboveTableGuard reaches the version-range runtime guard, which sits
// after the max_size argument check (so a max_size above 40 is needed).
func TestSizeAboveTableGuard(t *testing.T) {
	if _, err := New("x", WithSize(50), WithMaxSize(50)); !errors.Is(err, ErrRunTime) {
		t.Errorf("size 50 with max 50 should hit the version-range guard: %v", err)
	}
}

// TestPNGEncodeError injects a failing encoder to cover the encode-error branch.
func TestPNGEncodeError(t *testing.T) {
	orig := pngEncode
	defer func() { pngEncode = orig }()
	pngEncode = func(io.Writer, image.Image) error { return errors.New("boom") }
	q, _ := New("x", WithLevel(LevelL))
	if _, err := q.AsPNG(PNGOptions{}); err == nil {
		t.Error("expected encode error")
	}
}

// TestCreateBytesZeroECBranch drives create_bytes' modIndex<0 branch, where the
// mod-polynomial is shorter than the EC block and the leading EC codewords are
// filled with an explicit zero. 46 "A"s at level L is a known trigger; the code
// must still build and match its own re-encode (determinism).
func TestCreateBytesZeroECBranch(t *testing.T) {
	data := strings.Repeat("A", 46)
	q, err := New(data, WithLevel(LevelL))
	if err != nil {
		t.Fatal(err)
	}
	q2, err := New(data, WithLevel(LevelL))
	if err != nil {
		t.Fatal(err)
	}
	for i := range q.Modules {
		for j := range q.Modules[i] {
			if q.Modules[i][j] != q2.Modules[i][j] {
				t.Fatalf("non-deterministic at %d,%d", i, j)
			}
		}
	}
}

// TestNumberLengthOK covers the defined and undefined chunk lengths.
func TestNumberLengthOK(t *testing.T) {
	for _, l := range []int{1, 2, 3} {
		if _, ok := numberLengthOK(l); !ok {
			t.Errorf("numberLengthOK(%d) not ok", l)
		}
	}
	if _, ok := numberLengthOK(0); ok {
		t.Error("numberLengthOK(0) should be undefined")
	}
	if numberLength(3) != 10 {
		t.Error("numberLength(3)")
	}
}

// recoverPanic runs f and returns the recovered panic value (nil if none).
func recoverPanic(f func()) (r any) {
	defer func() { r = recover() }()
	f()
	return nil
}

// min is a tiny helper for slicing in the tests.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
