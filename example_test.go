package imgprobe_test

import (
	"errors"
	"fmt"

	"github.com/gumeniukcom/imgprobe"
)

// Probe an upload's header: format and post-orientation dimensions, without
// decoding the full image.
func ExampleConfig() {
	data := mustPNG(640, 480) // stand-in for the uploaded bytes

	res, err := imgprobe.Config(data)
	if err != nil {
		fmt.Println("rejected:", err)
		return
	}
	fmt.Printf("%s %dx%d\n", res.Format, res.Width, res.Height)
	// Output: png 640x480
}

// Reject a decompression bomb before allocating any pixels: only the header is
// read, and the declared dimensions blow the total-area cap.
func ExampleConfig_decompressionBomb() {
	bomb := mustPNG(8000, 7000) // 56 MP > the 50 MP default cap

	_, err := imgprobe.Config(bomb)
	fmt.Println(errors.Is(err, imgprobe.ErrTooLarge))
	// Output: true
}

// Accept an explicit allowlist that adds GIF to the defaults.
func ExampleWithFormats() {
	gif := mustGIF(16, 16)

	res, err := imgprobe.Config(gif, imgprobe.WithFormats(
		imgprobe.JPEG, imgprobe.PNG, imgprobe.WEBP, imgprobe.GIF,
	))
	if err != nil {
		fmt.Println("rejected:", err)
		return
	}
	fmt.Printf("%s %dx%d\n", res.Format, res.Width, res.Height)
	// Output: gif 16x16
}
