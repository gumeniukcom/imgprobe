// Package imgprobe inspects image bytes using only pure-Go decoders (no cgo):
// it sniffs the real format from magic bytes, reads dimensions cheaply (header
// only, never a full decode), applies EXIF orientation, and rejects
// decompression bombs via a total-pixel cap BEFORE any pixels are allocated.
//
// # Why
//
// Reading an image's dimensions by fully decoding it is a denial-of-service
// vector: a 15000×15000 PNG is a tiny file on disk but decodes to ~900 MB of
// pixels. imgprobe reads only the header via each format's DecodeConfig and
// rejects the image when its DECLARED dimensions exceed a total-area cap, so the
// bomb is refused before anything is allocated.
//
// # Usage
//
//	res, err := imgprobe.Config(data)
//	switch {
//	case errors.Is(err, imgprobe.ErrTooLarge):   // → 413 Payload Too Large
//	case errors.Is(err, imgprobe.ErrUnsupported): // → 415 Unsupported Media Type
//	case err != nil:
//	default:
//	    // res.Format == "jpeg", res.Width, res.Height ...
//	}
//
// The enabled format set is an allowlist (default {JPEG, PNG, WEBP}); pass
// WithFormats to change it, WithOrientation to supply the EXIF orientation, and
// WithMaxPixels/WithMaxSide to tune the caps.
//
// # Security
//
// imgprobe protects the "read dimensions" step and nothing more:
//
//   - It reads only headers (DecodeConfig), so a hostile image cannot force a
//     full-frame allocation here.
//   - The total-area cap (WithMaxPixels, default 50 MP) is the real bomb guard;
//     a per-side cap alone is insufficient because a square image passes it.
//   - Format detection is by magic bytes, ignoring the claimed Content-Type or
//     file extension.
//
// It is NOT a substitute for bounding the raw upload size before you buffer the
// bytes, and any later full decode (e.g. resizing) must apply its own limits.
package imgprobe
