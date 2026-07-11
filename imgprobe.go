package imgprobe

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

// Default caps. MaxSide is a per-side sanity bound; MaxPixels is the total-area
// cap that actually guards against decompression bombs — a 15000x15000 image
// passes a per-side check but decodes to ~900 MB.
const (
	DefaultMaxPixels = 50_000_000 // 50 MP, matching real camera output
	DefaultMaxSide   = 15000
)

// Sentinels — HTTP handlers can map these to distinct status codes (415 vs 413).
// Both are wrapped in the returned error, so errors.Is works.
var (
	// ErrUnsupported means the bytes are not one of the enabled formats, or the
	// header could not be decoded. Maps naturally to 415 Unsupported Media Type.
	ErrUnsupported = errors.New("imgprobe: unsupported or invalid image")
	// ErrTooLarge means the declared dimensions exceed the configured caps. Maps
	// naturally to 413 Payload Too Large.
	ErrTooLarge = errors.New("imgprobe: image exceeds size limits")
)

// Result is what a probe reveals about the bytes. Format is a neutral name
// string ("jpeg", "png", "webp", "gif", "bmp", "tiff").
type Result struct {
	Format      string
	ContentType string
	Width       int // post-orientation
	Height      int // post-orientation
}

// DecodeConfigFunc reads image dimensions from a header without a full decode.
type DecodeConfigFunc func(data []byte) (width, height int, err error)

// Format describes a recognized image format: its name, content type, magic-byte
// matcher, and pure-Go header decoder. Config dispatches on each enabled
// format's own decoder rather than Go's global image registry, so enabling is
// truly per-call: a disabled format is ErrUnsupported even though its decoder is
// linked into the binary.
type Format struct {
	name        string
	contentType string
	match       func(head []byte) bool
	decode      DecodeConfigFunc
}

// Name returns the neutral format name (e.g. "jpeg").
func (f Format) Name() string { return f.name }

// ContentType returns the canonical MIME type (e.g. "image/jpeg").
func (f Format) ContentType() string { return f.contentType }

// NewFormat registers a custom format. magic is matched against the leading
// bytes of the input; dc reads its dimensions.
func NewFormat(name, contentType string, magic []byte, dc DecodeConfigFunc) Format {
	return Format{name: name, contentType: contentType, match: prefixMatcher(magic), decode: dc}
}

// Built-in format descriptors. JPEG, PNG and WEBP are enabled by default;
// GIF, BMP and TIFF are opt-in via WithFormats.
var (
	JPEG = Format{"jpeg", "image/jpeg", prefixMatcher([]byte{0xFF, 0xD8, 0xFF}), decodeVia(jpeg.DecodeConfig)}
	PNG  = Format{"png", "image/png", prefixMatcher([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}), decodeVia(png.DecodeConfig)}
	WEBP = Format{"webp", "image/webp", matchWebP, decodeVia(webp.DecodeConfig)}
	GIF  = Format{"gif", "image/gif", matchGIF, decodeVia(gif.DecodeConfig)}
	BMP  = Format{"bmp", "image/bmp", prefixMatcher([]byte("BM")), decodeVia(bmp.DecodeConfig)}
	TIFF = Format{"tiff", "image/tiff", matchTIFF, decodeVia(tiff.DecodeConfig)}
)

func prefixMatcher(magic []byte) func([]byte) bool {
	m := append([]byte(nil), magic...)
	return func(head []byte) bool { return len(head) >= len(m) && bytes.Equal(head[:len(m)], m) }
}

func matchWebP(head []byte) bool {
	return len(head) >= 12 && bytes.Equal(head[:4], []byte("RIFF")) && bytes.Equal(head[8:12], []byte("WEBP"))
}

func matchGIF(head []byte) bool {
	return len(head) >= 6 && (bytes.Equal(head[:6], []byte("GIF87a")) || bytes.Equal(head[:6], []byte("GIF89a")))
}

func matchTIFF(head []byte) bool {
	return len(head) >= 4 && (bytes.Equal(head[:4], []byte{'I', 'I', 0x2A, 0x00}) || bytes.Equal(head[:4], []byte{'M', 'M', 0x00, 0x2A}))
}

// decodeVia adapts an image.Config decoder to a DecodeConfigFunc.
func decodeVia(dc func(io.Reader) (image.Config, error)) DecodeConfigFunc {
	return func(data []byte) (int, int, error) {
		cfg, err := dc(bytes.NewReader(data))
		if err != nil {
			return 0, 0, err
		}
		return cfg.Width, cfg.Height, nil
	}
}

type config struct {
	maxPixels   int64
	maxSide     int
	orientation int
	formats     []Format
}

func newConfig(opts []Option) config {
	c := config{
		maxPixels:   DefaultMaxPixels,
		maxSide:     DefaultMaxSide,
		orientation: 1,
		formats:     []Format{JPEG, PNG, WEBP},
	}
	for _, o := range opts {
		o(&c)
	}
	return c
}

// Option configures Sniff and Config. Safe defaults apply when omitted.
type Option func(*config)

// WithMaxPixels overrides the total-area cap (default DefaultMaxPixels).
// Non-positive values are ignored.
func WithMaxPixels(n int64) Option {
	return func(c *config) {
		if n > 0 {
			c.maxPixels = n
		}
	}
}

// WithMaxSide overrides the per-side cap (default DefaultMaxSide). Non-positive
// values are ignored.
func WithMaxSide(n int) Option {
	return func(c *config) {
		if n > 0 {
			c.maxSide = n
		}
	}
}

// WithOrientation supplies the EXIF orientation (1..8); values 5..8 swap width
// and height. The default is 1 (no adjustment). imgprobe does not parse EXIF —
// the caller extracts orientation separately.
func WithOrientation(orientation int) Option {
	return func(c *config) { c.orientation = orientation }
}

// WithFormats sets the EXACT enabled set (an allowlist), replacing the default
// {JPEG, PNG, WEBP}. "Defaults plus GIF" is spelled explicitly:
// WithFormats(imgprobe.JPEG, imgprobe.PNG, imgprobe.WEBP, imgprobe.GIF). An
// empty argument list is ignored (the default set stays in effect).
func WithFormats(formats ...Format) Option {
	return func(c *config) {
		if len(formats) > 0 {
			c.formats = append([]Format(nil), formats...)
		}
	}
}

// Sniff determines the real format from leading magic bytes, ignoring any
// claimed Content-Type or file extension. It considers only the enabled set
// (default {JPEG, PNG, WEBP}) and returns ErrUnsupported otherwise.
func Sniff(head []byte, opts ...Option) (format, contentType string, err error) {
	c := newConfig(opts)
	f, ok := matchFormat(head, c.formats)
	if !ok {
		return "", "", ErrUnsupported
	}
	return f.name, f.contentType, nil
}

// Config reads format + dimensions cheaply (header only, never a full decode),
// applies the EXIF orientation, and enforces the caps. It returns ErrTooLarge
// for a cap violation and ErrUnsupported for a bad or disabled format.
func Config(data []byte, opts ...Option) (Result, error) {
	c := newConfig(opts)

	f, ok := matchFormat(data, c.formats)
	if !ok {
		return Result{}, ErrUnsupported
	}

	w, h, err := f.decode(data)
	if err != nil {
		return Result{}, fmt.Errorf("%w: decode %s config: %w", ErrUnsupported, f.name, err)
	}

	// EXIF orientation 5..8 swap width and height.
	if c.orientation >= 5 && c.orientation <= 8 {
		w, h = h, w
	}

	if w <= 0 || h <= 0 {
		return Result{}, fmt.Errorf("%w: non-positive dimensions", ErrUnsupported)
	}
	if w > c.maxSide || h > c.maxSide {
		return Result{}, fmt.Errorf("%w: %dx%d exceeds per-side cap %d", ErrTooLarge, w, h, c.maxSide)
	}
	if int64(w)*int64(h) > c.maxPixels {
		return Result{}, fmt.Errorf("%w: %dx%d exceeds %d pixels", ErrTooLarge, w, h, c.maxPixels)
	}

	return Result{Format: f.name, ContentType: f.contentType, Width: w, Height: h}, nil
}

func matchFormat(head []byte, formats []Format) (Format, bool) {
	for _, f := range formats {
		if f.match(head) {
			return f, true
		}
	}
	return Format{}, false
}
