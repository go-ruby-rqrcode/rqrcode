// Copyright (c) the go-ruby-rqrcode/rqrcode authors
//
// SPDX-License-Identifier: BSD-3-Clause

package rqrcode

import (
	"os/exec"
	"strings"
	"testing"
)

// rubyWithGems locates a `ruby` that can require rqrcode_core / rqrcode. The
// oracle tests skip themselves when it is absent (the qemu cross-arch lanes and
// the Windows lane), so the deterministic golden suite alone drives the 100%
// gate there.
func rubyWithGems(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("ruby")
	if err != nil {
		t.Skip("ruby not on PATH; skipping MRI oracle")
	}
	// Probe that both gems load; if not, skip rather than fail.
	probe := exec.Command(path, "-e", "require 'rqrcode_core'; require 'rqrcode'")
	if out, err := probe.CombinedOutput(); err != nil {
		t.Skipf("rqrcode gems not installed (%v): %s", err, out)
	}
	return path
}

// rubyRun executes a ruby script and returns stdout, failing on error. The script
// binmodes stdout so no text-mode translation pollutes the bytes.
func rubyRun(t *testing.T, bin, script string) string {
	t.Helper()
	cmd := exec.Command(bin, "-e", "$stdout.binmode\n"+script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ruby error: %v\nscript:\n%s\noutput:\n%s", err, script, out)
	}
	return string(out)
}

// oracleCorpus is the differential corpus: payloads spanning numeric,
// alphanumeric and byte modes, exercised across every level and both
// auto-selected and explicit versions (including v>=7 for version info).
var oracleCorpus = []struct {
	data string
	size int // 0 = auto
}{
	{"12345", 0},
	{"1234567890123456789012345", 0},
	{"HELLO WORLD", 0},
	{"HTTP://EXAMPLE.COM/PATH?A=1 B", 0},
	{"hello world", 0},
	{"Mixed 123 abc XYZ ~!@#", 0},
	{"a", 0},
	{"The quick brown fox 0123456789", 0},
	{"EXAMPLE", 7},
	{"EXAMPLE", 10},
	{"ABC", 1},
}

// TestOracleMatrix compares every generated matrix against rqrcode_core,
// module-for-module.
func TestOracleMatrix(t *testing.T) {
	bin := rubyWithGems(t)
	for _, c := range oracleCorpus {
		for _, lvl := range []string{"l", "m", "q", "h"} {
			c, lvl := c, lvl
			name := c.data + "/" + lvl
			t.Run(name, func(t *testing.T) {
				opts := []Option{WithLevelSymbol(lvl)}
				sizeArg := ""
				if c.size != 0 {
					opts = append(opts, WithSize(c.size))
					sizeArg = ", size: " + itoa(c.size)
				}
				q, err := New(c.data, opts...)
				if err != nil {
					t.Fatalf("New: %v", err)
				}
				script := `require 'rqrcode_core'
q = RQRCodeCore::QRCode.new(` + rubyStr(c.data) + `, level: :` + lvl + sizeArg + `)
q.modules.each { |r| $stdout.print(r.map { |x| x ? "1" : "0" }.join, "\n") }`
				want := rubyRun(t, bin, script)
				got := strings.Join(matrixToStrings(q), "\n") + "\n"
				if got != want {
					t.Fatalf("matrix mismatch for %q level %s size %d", c.data, lvl, c.size)
				}
			})
		}
	}
}

// TestOracleRenderers compares to_s / as_svg (rect + path) / as_ansi / as_html
// against the rqrcode gem byte-for-byte.
func TestOracleRenderers(t *testing.T) {
	bin := rubyWithGems(t)
	inputs := []string{"hello world", "HELLO WORLD", "12345", "Mixed ~ 42"}
	for _, data := range inputs {
		data := data
		t.Run(data, func(t *testing.T) {
			q, err := New(data, WithLevelSymbol("h"))
			if err != nil {
				t.Fatal(err)
			}
			ds := rubyStr(data)

			checks := []struct {
				label  string
				got    string
				script string
			}{
				{"to_s", q.String(),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).to_s`},
				{"to_s_qz", q.ToString(StringOptions{QuietZoneSize: 2, Dark: "E", Light: "Q"}),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).to_s(quiet_zone_size: 2, dark: "E", light: "Q")`},
				{"as_svg", q.AsSVG(SVGOptions{}),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_svg`},
				{"as_svg_path", q.AsSVG(SVGOptions{UsePath: true, ModuleSize: 7, Offset: 4, Color: "123456"}),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_svg(use_path: true, module_size: 7, offset: 4, color: "123456")`},
				{"as_svg_fill", q.AsSVG(SVGOptions{Fill: "eeeeee", ModuleSize: 5}),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_svg(fill: "eeeeee", module_size: 5)`},
				{"as_ansi", q.AsANSIDefault(),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_ansi`},
				{"as_ansi_opt", q.AsANSI(ANSIOptions{QuietZoneSize: 1, FillCharacter: "@@"}),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_ansi(quiet_zone_size: 1, fill_character: "@@")`},
				{"as_html", q.AsHTML(),
					`require 'rqrcode'; $stdout.print RQRCode::QRCode.new(` + ds + `, level: :h).as_html`},
			}
			for _, ch := range checks {
				want := rubyRun(t, bin, ch.script)
				if ch.got != want {
					t.Errorf("%s mismatch:\n got %q\nwant %q", ch.label, ch.got, want)
				}
			}
		})
	}
}

// TestOracleMulti compares a multi-segment code against the gem.
func TestOracleMulti(t *testing.T) {
	bin := rubyWithGems(t)
	q, err := NewMulti([]Segment{
		Seg("foo", ModeByte8bit),
		Seg("1234", ModeNumeric),
		Seg("BAR", ModeAlphanumeric),
	}, WithLevelSymbol("m"))
	if err != nil {
		t.Fatal(err)
	}
	script := `require 'rqrcode_core'
q = RQRCodeCore::QRCode.new([
  { data: "foo", mode: :byte_8bit },
  { data: "1234", mode: :number },
  { data: "BAR", mode: :alphanumeric }
], level: :m)
q.modules.each { |r| $stdout.print(r.map { |x| x ? "1" : "0" }.join, "\n") }`
	want := rubyRun(t, bin, script)
	got := strings.Join(matrixToStrings(q), "\n") + "\n"
	if got != want {
		t.Fatal("multi-segment matrix mismatch")
	}
}

// rubyStr renders a Go string as a Ruby double-quoted literal for the oracle
// scripts, escaping backslashes and quotes (the corpus is plain ASCII/UTF-8).
func rubyStr(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	return `"` + r.Replace(s) + `"`
}
