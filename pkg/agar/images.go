package agar

import (
	"path/filepath"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

// GenerateTestJPEG creates a test JPEG image with a solid color background.
// Returns the path to the generated image.
func GenerateTestJPEG(data test.Data, helpers test.Helpers, color string) string {
	helpers.T().Helper()

	outputPath := filepath.Join(data.Temp().Dir(), "test-cover.jpg")

	helpers.Custom(ffmpegBinary, "-y",
		"-f", "lavfi",
		"-i", "color=c="+color+":s=500x500:d=1",
		"-frames:v", "1",
		"-q:v", "2",
		outputPath,
	).Run(&test.Expected{})

	return outputPath
}

// GenerateTestPNG creates a test PNG image with a solid color background.
// Returns the path to the generated image.
func GenerateTestPNG(data test.Data, helpers test.Helpers, color string) string {
	helpers.T().Helper()

	outputPath := filepath.Join(data.Temp().Dir(), "test-cover.png")

	helpers.Custom(ffmpegBinary, "-y",
		"-f", "lavfi",
		"-i", "color=c="+color+":s=500x500:d=1",
		"-frames:v", "1",
		outputPath,
	).Run(&test.Expected{})

	return outputPath
}

// TestCoverJPEG returns path to a default test cover image (blue, JPEG).
func TestCoverJPEG(data test.Data, helpers test.Helpers) string {
	helpers.T().Helper()

	return GenerateTestJPEG(data, helpers, "blue")
}

// TestCoverPNG returns path to a default test cover image (red, PNG).
func TestCoverPNG(data test.Data, helpers test.Helpers) string {
	helpers.T().Helper()

	return GenerateTestPNG(data, helpers, "red")
}

// TestCoverAlternate returns path to an alternate test cover image (green, JPEG).
// Useful for testing cover replacement.
func TestCoverAlternate(data test.Data, helpers test.Helpers) string {
	helpers.T().Helper()

	return GenerateTestJPEG(data, helpers, "green")
}
