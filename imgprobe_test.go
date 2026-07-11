package imgprobe_test

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/gumeniukcom/imgprobe"
)

func TestSniff(t *testing.T) {
	tests := []struct {
		name    string
		head    []byte
		want    string
		wantErr bool
	}{
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, "jpeg", false},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, "png", false},
		{"webp", append([]byte("RIFF"), append([]byte{0, 0, 0, 0}, []byte("WEBP")...)...), "webp", false},
		{"gif not in default set", []byte("GIF89a"), "", true},
		{"bmp not in default set", []byte("BM\x00\x00"), "", true},
		{"svg rejected", []byte("<svg xmlns"), "", true},
		{"too short", []byte{0xFF}, "", true},
		{"empty", nil, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := imgprobe.Sniff(tt.head)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s", tt.name)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig(t *testing.T) {
	jpg := mustJPEG(640, 480)

	t.Run("reads dimensions and neutral format", func(t *testing.T) {
		res, err := imgprobe.Config(jpg)
		if err != nil {
			t.Fatal(err)
		}
		if res.Width != 640 || res.Height != 480 {
			t.Errorf("got %dx%d, want 640x480", res.Width, res.Height)
		}
		if res.Format != "jpeg" || res.ContentType != "image/jpeg" {
			t.Errorf("format %q / %q", res.Format, res.ContentType)
		}
	})

	t.Run("orientation 6 swaps width and height", func(t *testing.T) {
		res, err := imgprobe.Config(jpg, imgprobe.WithOrientation(6))
		if err != nil {
			t.Fatal(err)
		}
		if res.Width != 480 || res.Height != 640 {
			t.Errorf("orientation 6 should swap to 480x640, got %dx%d", res.Width, res.Height)
		}
	})

	t.Run("rejects non-image bytes with ErrUnsupported", func(t *testing.T) {
		_, err := imgprobe.Config([]byte("not an image"))
		if !errors.Is(err, imgprobe.ErrUnsupported) {
			t.Errorf("want ErrUnsupported, got %v", err)
		}
	})

	t.Run("rejects oversized pixel count with ErrTooLarge", func(t *testing.T) {
		big := mustPNG(8000, 7000) // 56 MP > 50 MP default cap
		_, err := imgprobe.Config(big)
		if !errors.Is(err, imgprobe.ErrTooLarge) {
			t.Errorf("want ErrTooLarge, got %v", err)
		}
	})

	t.Run("per-side cap is distinct from pixel cap", func(t *testing.T) {
		// 20000x10 = 200k pixels (under the pixel cap) but exceeds the 15000 per-side cap.
		wide := mustPNG(20000, 10)
		_, err := imgprobe.Config(wide)
		if !errors.Is(err, imgprobe.ErrTooLarge) {
			t.Errorf("want ErrTooLarge for per-side overflow, got %v", err)
		}
	})

	t.Run("WithMaxPixels tightens the cap", func(t *testing.T) {
		small := mustPNG(1000, 1000) // 1 MP
		if _, err := imgprobe.Config(small); err != nil {
			t.Fatalf("1MP should pass default: %v", err)
		}
		if _, err := imgprobe.Config(small, imgprobe.WithMaxPixels(500_000)); !errors.Is(err, imgprobe.ErrTooLarge) {
			t.Errorf("want ErrTooLarge under a 0.5MP cap, got %v", err)
		}
	})
}

func TestFormatAllowlist(t *testing.T) {
	g := mustGIF(32, 24)

	t.Run("GIF rejected under default set", func(t *testing.T) {
		if _, err := imgprobe.Config(g); !errors.Is(err, imgprobe.ErrUnsupported) {
			t.Errorf("want ErrUnsupported, got %v", err)
		}
	})

	t.Run("GIF probes when explicitly enabled", func(t *testing.T) {
		res, err := imgprobe.Config(g, imgprobe.WithFormats(imgprobe.JPEG, imgprobe.PNG, imgprobe.WEBP, imgprobe.GIF))
		if err != nil {
			t.Fatalf("enabled GIF should probe: %v", err)
		}
		if res.Format != "gif" || res.Width != 32 || res.Height != 24 {
			t.Errorf("got %+v", res)
		}
	})

	t.Run("allowlist can restrict to a single format", func(t *testing.T) {
		jpg := mustJPEG(10, 10)
		if _, err := imgprobe.Config(jpg, imgprobe.WithFormats(imgprobe.PNG)); !errors.Is(err, imgprobe.ErrUnsupported) {
			t.Errorf("JPEG must be rejected under a PNG-only allowlist, got %v", err)
		}
	})
}

func TestNewFormatCustom(t *testing.T) {
	// A custom format reusing the GIF decoder but a bespoke name/magic.
	custom := imgprobe.NewFormat("mygif", "image/x-mygif", []byte("GIF89a"), func(data []byte) (int, int, error) {
		cfg, err := gif.DecodeConfig(bytes.NewReader(data))
		return cfg.Width, cfg.Height, err
	})
	res, err := imgprobe.Config(mustGIF(8, 8), imgprobe.WithFormats(custom))
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "mygif" || res.ContentType != "image/x-mygif" {
		t.Errorf("custom format not honored: %+v", res)
	}
}

// FuzzSniff and FuzzConfig assert the probe never panics on arbitrary bytes.
func FuzzSniff(f *testing.F) {
	f.Add([]byte{0xFF, 0xD8, 0xFF})
	f.Add([]byte("GIF89a"))
	f.Add([]byte("not an image"))
	f.Fuzz(func(_ *testing.T, b []byte) {
		_, _, _ = imgprobe.Sniff(b)
		_, _ = imgprobe.Config(b)
	})
}

// --- test helpers ---

func mustJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func mustPNG(w, h int) []byte {
	img := image.NewGray(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func mustGIF(w, h int) []byte {
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	if err := gif.Encode(&buf, img, nil); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
