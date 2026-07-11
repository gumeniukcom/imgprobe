package imgprobe_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"testing"

	"github.com/gumeniukcom/imgprobe"
)

func BenchmarkConfig(b *testing.B) {
	data := mustJPEG(4000, 3000) // 12 MP, a typical camera upload
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := imgprobe.Config(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConfig_bombRejected(b *testing.B) {
	bomb := forgedPNG(20000, 20000) // ~45-byte file claiming 400 MP
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _ = imgprobe.Config(bomb)
	}
}

func BenchmarkSniff(b *testing.B) {
	data := mustJPEG(64, 64)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, _, _ = imgprobe.Sniff(data)
	}
}

// TestBombRejectionReadsHeaderOnly is the core security claim, measured: a tiny
// file whose IHDR declares 20000x20000 (400 MP, ~1.6 GB fully decoded) is
// rejected with ErrTooLarge while allocating only header-sized memory — the
// pixel buffer is never touched.
func TestBombRejectionReadsHeaderOnly(t *testing.T) {
	bomb := forgedPNG(20000, 20000)

	if _, err := imgprobe.Config(bomb); !errors.Is(err, imgprobe.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}

	res := testing.Benchmark(func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_, _ = imgprobe.Config(bomb)
		}
	})
	// A full decode would allocate ~1.6 GB; header-only is a few KB. 256 KB is a
	// generous ceiling that still separates the two by four orders of magnitude.
	if got := res.AllocedBytesPerOp(); got > 256*1024 {
		t.Errorf("bomb rejection allocated %d bytes/op; expected header-only (< 256 KiB)", got)
	}
}

// forgedPNG builds a minimal valid PNG — 8-byte signature + IHDR chunk only —
// declaring the given dimensions. image/png's DecodeConfig reads the IHDR and
// returns those dimensions without decoding any pixels, which is exactly the
// path imgprobe relies on to reject a bomb before allocation.
func forgedPNG(w, h uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], w)
	binary.BigEndian.PutUint32(ihdr[4:8], h)
	ihdr[8] = 8 // bit depth
	ihdr[9] = 0 // color type: grayscale
	// ihdr[10..12] = 0: compression, filter, interlace

	var length [4]byte
	binary.BigEndian.PutUint32(length[:], uint32(len(ihdr)))
	buf.Write(length[:])
	buf.WriteString("IHDR")
	buf.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write([]byte("IHDR"))
	crc.Write(ihdr)
	var sum [4]byte
	binary.BigEndian.PutUint32(sum[:], crc.Sum32())
	buf.Write(sum[:])

	return buf.Bytes()
}
