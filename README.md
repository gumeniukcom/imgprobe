# imgprobe

[![Go Reference](https://pkg.go.dev/badge/github.com/gumeniukcom/imgprobe.svg)](https://pkg.go.dev/github.com/gumeniukcom/imgprobe)
[![Go Report Card](https://goreportcard.com/badge/github.com/gumeniukcom/imgprobe)](https://goreportcard.com/report/github.com/gumeniukcom/imgprobe)
[![CI](https://github.com/gumeniukcom/imgprobe/actions/workflows/ci.yml/badge.svg)](https://github.com/gumeniukcom/imgprobe/actions/workflows/ci.yml)
[![CodeQL](https://github.com/gumeniukcom/imgprobe/actions/workflows/codeql.yml/badge.svg)](https://github.com/gumeniukcom/imgprobe/actions/workflows/codeql.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Pure-Go (zero cgo) image preflight for uploads: sniff the real format from magic
bytes, read dimensions from the header only, apply EXIF orientation, and
**reject decompression bombs before any pixels are allocated**.

```go
res, err := imgprobe.Config(uploadBytes)
switch {
case errors.Is(err, imgprobe.ErrTooLarge):    // → 413 Payload Too Large
    http.Error(w, "image too large", http.StatusRequestEntityTooLarge)
case errors.Is(err, imgprobe.ErrUnsupported): // → 415 Unsupported Media Type
    http.Error(w, "unsupported image", http.StatusUnsupportedMediaType)
case err != nil:
    http.Error(w, "bad upload", http.StatusBadRequest)
default:
    // res.Format == "jpeg", res.ContentType, res.Width, res.Height
}
```

## The problem it solves

Reading an image's dimensions by fully decoding it is a denial-of-service
vector: a `15000×15000` PNG is a tiny file on disk but decodes to **~900 MB** of
pixels. A handful of concurrent uploads like that will OOM a service.

`imgprobe` reads only the header (each format's `DecodeConfig`, never a full
decode) and rejects the image when its **declared** dimensions exceed a
total-area cap — so the bomb is refused before anything is allocated.

The load-bearing detail: a per-side cap alone is **not** enough — `15000×15000`
passes any reasonable per-side check yet is a bomb. The **total-pixel** cap
(`WithMaxPixels`, default 50 MP) is the real guard. Both ship, with safe
defaults.

It really is header-only — measured (`make bench`, Apple M-series):

```
BenchmarkConfig                12 MP JPEG      849 ns/op     13808 B/op    4 allocs/op
BenchmarkConfig_bombRejected   400 MP forged   401 ns/op      1416 B/op   10 allocs/op
BenchmarkSniff                 magic bytes      49 ns/op       192 B/op    2 allocs/op
```

A `20000×20000` PNG (≈1.6 GB fully decoded) is rejected in ~400 ns using ~1.4 KB
— the pixel buffer is never allocated. This is asserted, not just claimed:
`TestBombRejectionReadsHeaderOnly` fails if the rejection path allocates more
than header-sized memory.

## Why not just call `image.DecodeConfig` yourself?

You can, and you should keep doing so if that is all you need. `imgprobe` adds
the parts that are easy to get wrong or tedious to repeat:

- **The total-area bomb cap** as a first-class, tested guarantee.
- **Typed sentinels** (`ErrTooLarge` → 413, `ErrUnsupported` → 415) so an HTTP
  handler maps failures to the right status without string-matching.
- **Magic-byte format detection** that ignores the claimed `Content-Type`/extension.
- **A per-call format allowlist** — accept exactly the formats you want.
- **EXIF-orientation-aware dimensions** (you supply the orientation).

## Formats

Enabled by default: **JPEG, PNG, WebP**. Opt-in via `WithFormats`: **GIF, BMP,
TIFF**. Register your own with `NewFormat`. `WithFormats` sets the exact
allowlist (replace semantics), so you can also restrict to a single format:

```go
// defaults plus GIF:
imgprobe.Config(b, imgprobe.WithFormats(imgprobe.JPEG, imgprobe.PNG, imgprobe.WEBP, imgprobe.GIF))

// PNG only:
imgprobe.Config(b, imgprobe.WithFormats(imgprobe.PNG))
```

## Security

`imgprobe` protects the "read dimensions" step and nothing more. It is **not** a
substitute for bounding the raw upload size before you buffer the bytes, and any
later full decode (e.g. resizing) must apply its own limits. See
[SECURITY.md](SECURITY.md) and the package "Security" section on
[pkg.go.dev](https://pkg.go.dev/github.com/gumeniukcom/imgprobe).

## Install

```bash
go get github.com/gumeniukcom/imgprobe
```

Requires Go 1.25+. The only non-stdlib dependency is `golang.org/x/image` (for
WebP/BMP/TIFF header decoding). No cgo.

## License

MIT — see [LICENSE](LICENSE).
