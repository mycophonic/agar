/*
   Copyright Mycophonic.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package agar

import (
	"path/filepath"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

// GenerateTestJPEG creates a test JPEG image with a solid color background.
// Returns the path to the generated image.
func GenerateTestJPEG(data test.Data, helpers test.Helpers, color string) string {
	helpers.T().Helper()

	ffmpeg := lookForOrFail(helpers.T(), ffmpegBinary)
	outputPath := filepath.Join(data.Temp().Dir(), "test-cover.jpg")

	helpers.Custom(ffmpeg, "-y",
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

	ffmpeg := lookForOrFail(helpers.T(), ffmpegBinary)
	outputPath := filepath.Join(data.Temp().Dir(), "test-cover.png")

	helpers.Custom(ffmpeg, "-y",
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
