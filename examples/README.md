# Ruby examples

Pure-Ruby examples for the `rqrcode` QR-code generator as provided by
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) (rbgo). The matrix
generation and every renderer are deterministic and need no C library. Run it
with the `rbgo` interpreter:

```sh
rbgo examples/rqrcode_usage.rb
```

| File | Shows |
| --- | --- |
| [`rqrcode_usage.rb`](rqrcode_usage.rb) | `RQRCode::QRCode.new(data, level:, mode:)`; `#version` / `#module_count`, `#modules` and `#checked?`, styled `#to_s` and `#as_svg` renderers, and rescuing `RQRCode::QRCodeArgumentError`. |

Each example is executed as-is under rbgo (`require "rqrcode"`).
