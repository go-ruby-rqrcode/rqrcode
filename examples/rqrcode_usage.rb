# frozen_string_literal: true
#
# Pure-Ruby usage of the rqrcode QR-code generator, as provided by
# go-embedded-ruby (rbgo). Run it with:  rbgo examples/rqrcode_usage.rb
#
# Building the QR matrix (segment encoding, Reed-Solomon ECC, mask selection,
# module layout) is fully deterministic and needs no C library: RQRCode::QRCode
# picks the smallest fitting version and renders the same modules as the gem.

require "rqrcode"

# .new(data, level:, size:, mode:) builds a code. Default EC level is :h.
qr = RQRCode::QRCode.new("HELLO WORLD", level: :h)
puts "version=#{qr.version} module_count=#{qr.module_count}" # => version=2 module_count=25

# #modules / #to_a hand back the matrix as an Array of Arrays of booleans;
# #checked?(row, col) reports one dark module (the finder corner is dark).
p qr.modules[0][0]        # => true
p qr.checked?(0, 0)       # => true

# #to_s renders text; pass :dark / :light / :quiet_zone_size to style it.
puts qr.to_s(dark: "##", light: "  ", quiet_zone_size: 1)

# Every renderer is available: #as_svg, #as_ansi, #as_html all return a String.
puts "svg bytes=#{qr.as_svg(module_size: 4).bytesize}"

# Forcing :number mode on non-numeric data raises the gem's argument error.
begin
  RQRCode::QRCode.new("abc", mode: :number)
rescue RQRCode::QRCodeArgumentError => e
  puts "rescued: #{e.message}" # => rescued: Not a numeric string `abc`
end
