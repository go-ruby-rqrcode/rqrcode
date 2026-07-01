<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-rqrcode/brand/main/social/go-ruby-rqrcode-rqrcode.png" alt="go-ruby-rqrcode/rqrcode" width="720"></p>

# rqrcode — go-ruby-rqrcode

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-rqrcode.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of Ruby's
[`rqrcode_core`](https://github.com/whomwah/rqrcode_core) and
[`rqrcode`](https://github.com/whomwah/rqrcode) gems** (MRI 4.0.5). It generates
the QR-code module matrix deterministically — modes, error-correction levels,
versions 1–40 with auto-selection, Reed–Solomon ECC, all eight mask patterns with
penalty scoring, finder / alignment / timing patterns and format / version info —
and renders it (SVG, ANSI, HTML, PNG) **module-for-module and byte-for-byte** as
the gems do, **without any Ruby runtime or a C QR library**.

It is a QR backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module — a sibling of
[go-ruby-regexp](https://github.com/go-ruby-regexp/regexp),
[go-ruby-erb](https://github.com/go-ruby-erb/erb) and
[go-ruby-yaml](https://github.com/go-ruby-yaml/yaml).

> **What it is — and isn't.** Building the QR matrix (segment encoding, RS error
> correction, mask selection, module layout) is fully deterministic and needs
> **no interpreter**, so it lives here as pure Go, matching the gems exactly.
> Binding `RQRCode::QRCode` to live Ruby objects is the host's job; this library
> hands back a plain `[][]bool` matrix and the renderers.

## Features

Faithful port of `rqrcode_core` + `rqrcode`, validated against the gems on every
supported platform:

- **All encoding modes** — numeric, alphanumeric, byte (8-bit), plus
  **multi-segment** codes — with the gems' automatic mode detection.
- **All four EC levels** — `:l` / `:m` / `:q` / `:h` (default `:h`, as in the gem).
- **Versions 1–40** with the gem's smallest-that-fits auto-selection, or an
  explicit `WithSize`.
- **Reed–Solomon ECC** over GF(256), block interleaving, and the exact
  `QRMAXBITS` / `RS_BLOCK_TABLE` capacity tables.
- **Mask selection** — all eight mask predicates and the four-part penalty
  (`get_lost_points`) scoring, choosing the same mask as the gem.
- **Function patterns** — finder + separators, alignment, timing, dark module,
  15-bit BCH format info and 18-bit BCH version info (v ≥ 7).
- **Renderers** — `AsSVG` (both `<rect>` and the size-optimised `<path>`),
  `AsANSI`, `AsHTML`, `ToString`, and a pure-Go `AsPNG` (stdlib `image/png`,
  still CGO-free).

CGO-free, dependency-free, **100% test coverage**, `gofmt` + `go vet` clean, and
green across the six 64-bit Go targets (amd64, arm64, riscv64, loong64, ppc64le,
s390x) and three OSes.

## Install

```sh
go get github.com/go-ruby-rqrcode/rqrcode
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-rqrcode/rqrcode"
)

func main() {
	// RQRCode::QRCode.new("hello world", level: :h)
	qr, err := rqrcode.New("hello world", rqrcode.WithLevelSymbol("h"))
	if err != nil {
		panic(err)
	}

	fmt.Println(qr.Version)     // 2  (auto-selected smallest version)
	fmt.Println(qr.ModuleCount) // 25

	// The boolean module matrix (qr.Modules[row][col] == qrcode.modules).
	dark, _ := qr.Checked(0, 0)
	fmt.Println(dark) // true — top-left finder pattern

	fmt.Println(qr.String())                    // to_s
	fmt.Println(qr.AsSVG(rqrcode.SVGOptions{}))  // as_svg
	fmt.Println(qr.AsANSIDefault())              // as_ansi
	fmt.Println(qr.AsHTML())                     // as_html

	png, _ := qr.AsPNG(rqrcode.PNGOptions{})     // as_png (pure-Go image/png)
	_ = png
}
```

Multi-segment codes mirror passing an array of `{data:, mode:}` hashes:

```go
qr, _ := rqrcode.NewMulti([]rqrcode.Segment{
	rqrcode.Seg("foo", rqrcode.ModeByte8bit),
	rqrcode.Seg("1234", rqrcode.ModeNumeric),
})
```

## API

```go
// New builds a QR code from a string (RQRCode::QRCode.new). Default level :h.
func New(data string, opts ...Option) (*QRCode, error)

// NewMulti builds a multi-segment code (array of {data:, mode:} hashes).
func NewMulti(segs []Segment, opts ...Option) (*QRCode, error)

type QRCode struct {
	Modules     [][]bool // dark/light matrix; Modules[row][col]
	ModuleCount int
	Version     int
	// ...
}
func (q *QRCode) Checked(row, col int) (bool, error) // checked?
func (q *QRCode) Mode() Mode
func (q *QRCode) ErrorCorrectionLevel() Level
func (q *QRCode) String() string                        // to_s (default x / space)
func (q *QRCode) ToString(opts StringOptions) string    // to_s(dark:, light:, quiet_zone_size:)
func (q *QRCode) AsSVG(opts SVGOptions) string          // as_svg (rect + path)
func (q *QRCode) AsANSI(opts ANSIOptions) string        // as_ansi
func (q *QRCode) AsANSIDefault() string                 // as_ansi with gem defaults
func (q *QRCode) AsHTML() string                        // as_html
func (q *QRCode) AsPNG(opts PNGOptions) ([]byte, error) // as_png (pure-Go)

// Options mirror the gems' keyword arguments.
func WithSize(v int) Option        // :size (QR version 1..40; 0 = auto)
func WithMaxSize(v int) Option     // :max_size
func WithLevel(l Level) Option     // :level (typed)
func WithLevelSymbol(s string) Option // :level as "l"/"m"/"q"/"h"
func WithMode(m Mode) Option       // :mode (typed)
func WithModeSymbol(s string) Option  // :mode as "number"/"alphanumeric"/"byte_8bit"

type Level int  // LevelL, LevelM, LevelQ, LevelH (values match QRERRORCORRECTLEVEL)
type Mode  int  // ModeNumeric, ModeAlphanumeric, ModeByte8bit
```

## Tests & coverage

The suite pairs **deterministic, gem-free golden tests** (which alone hold
coverage at 100%, so the qemu cross-arch and Windows lanes pass the gate) with a
**differential MRI oracle**: a corpus of numeric / alphanumeric / byte payloads
across every level and a spread of versions is generated here and compared —
module-for-module and byte-for-byte — against the `rqrcode_core` / `rqrcode` gems
run through the system `ruby` (matrix, `to_s`, `as_svg` rect + path, `as_ansi`,
`as_html`). The oracle skips itself where the gems are absent.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-rqrcode/rqrcode authors.
