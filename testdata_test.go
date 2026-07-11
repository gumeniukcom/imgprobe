package imgprobe_test

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"testing"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"

	"github.com/gumeniukcom/imgprobe"
)

// TestConfig_RealWebP exercises the webp DecodeConfig path with a real file.
// Go has no webp encoder, so a committed fixture (testdata/sample.webp) is the
// only way to cover this format end to end — synthetic bytes cannot.
func TestConfig_RealWebP(t *testing.T) {
	data := readFixture(t, "sample.webp")
	res, err := imgprobe.Config(data, imgprobe.WithFormats(imgprobe.WEBP))
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "webp" || res.Width != 12 || res.Height != 8 {
		t.Errorf("got %+v, want webp 12x8", res)
	}
}

// TestConfig_BigEndianTIFF exercises matchTIFF's big-endian ("MM") branch and
// the tiff decode path with a real MSB-ordered file (x/image/tiff.Encode only
// writes little-endian, so this branch is otherwise untested).
func TestConfig_BigEndianTIFF(t *testing.T) {
	data := readFixture(t, "sample_be.tiff")
	res, err := imgprobe.Config(data, imgprobe.WithFormats(imgprobe.TIFF))
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != "tiff" || res.Width != 13 || res.Height != 9 {
		t.Errorf("got %+v, want tiff 13x9", res)
	}
}

// TestConfig_BMPandLittleEndianTIFF covers the remaining opt-in decode paths via
// x/image encoders (no fixture needed — Go can produce these).
func TestConfig_BMPandLittleEndianTIFF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 20, 10))
	img.Set(0, 0, color.RGBA{1, 2, 3, 255})

	var bmpBuf bytes.Buffer
	if err := bmp.Encode(&bmpBuf, img); err != nil {
		t.Fatal(err)
	}
	if res, err := imgprobe.Config(bmpBuf.Bytes(), imgprobe.WithFormats(imgprobe.BMP)); err != nil || res.Format != "bmp" || res.Width != 20 || res.Height != 10 {
		t.Errorf("bmp: got %+v err=%v", res, err)
	}

	var tiffBuf bytes.Buffer
	if err := tiff.Encode(&tiffBuf, img, nil); err != nil {
		t.Fatal(err)
	}
	if res, err := imgprobe.Config(tiffBuf.Bytes(), imgprobe.WithFormats(imgprobe.TIFF)); err != nil || res.Format != "tiff" || res.Width != 20 || res.Height != 10 {
		t.Errorf("tiff-le: got %+v err=%v", res, err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
