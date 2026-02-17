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
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/containerd/nerdctl/mod/tigron/tig"

	"github.com/mycophonic/primordium/filesystem"
)

const (
	// dsdTestDuration is the default duration in seconds for generated DSD test data.
	// 100ms produces ~35KB at DSD64 — fast to generate, enough to exercise a pipeline.
	dsdTestDuration = 0.1

	// dsdBasePCMRate is the PCM sample rate used as the base for sigma-delta modulation.
	dsdBasePCMRate = 44100

	// dsdSineAmplitude is the peak amplitude of generated sine waves.
	// Kept below 1.0 to stay within the sigma-delta modulator's stable input range.
	dsdSineAmplitude = 0.5

	// dsdQuantizerPositive and dsdQuantizerNegative are the two output levels
	// of the 1-bit sigma-delta quantizer, representing DSD bit values 1 and 0.
	dsdQuantizerPositive = 1.0
	dsdQuantizerNegative = -1.0

	// bitsPerByte is used for bit-packing arithmetic.
	bitsPerByte = 8
)

// DSDSine writes raw DSD bytes for a sine wave at the given frequency.
// dsdRate is the DSD sample rate in Hz (e.g. 2822400 for DSD64).
// The file is written to dir, and helper is used for error reporting.
// Returns the path to the generated raw DSD file.
func DSDSine(dir string, helper tig.T, dsdRate int, freqHz float64) string {
	helper.Helper()

	numSamples := int(dsdTestDuration * dsdBasePCMRate)
	pcm := make([]float64, numSamples)

	for sample := range numSamples {
		pcm[sample] = dsdSineAmplitude * math.Sin(2*math.Pi*freqHz*float64(sample)/dsdBasePCMRate)
	}

	oversampleRatio := dsdRate / dsdBasePCMRate
	dsdBytes := sigmaDeltaModulate(pcm, oversampleRatio)

	outputPath := filepath.Join(dir, fmt.Sprintf("dsd-sine-%dhz-%d.raw", int(freqHz), dsdRate))
	writeDSDFile(helper, outputPath, dsdBytes)

	return outputPath
}

// DSDSilence writes raw DSD bytes for silence (zero signal).
// dsdRate is the DSD sample rate in Hz.
// The file is written to dir, and helper is used for error reporting.
// Returns the path to the generated raw DSD file.
func DSDSilence(dir string, helper tig.T, dsdRate int) string {
	helper.Helper()

	numSamples := int(dsdTestDuration * dsdBasePCMRate)
	pcm := make([]float64, numSamples) // all zeros

	oversampleRatio := dsdRate / dsdBasePCMRate
	dsdBytes := sigmaDeltaModulate(pcm, oversampleRatio)

	outputPath := filepath.Join(dir, fmt.Sprintf("dsd-silence-%d.raw", dsdRate))
	writeDSDFile(helper, outputPath, dsdBytes)

	return outputPath
}

// DSDDC writes raw DSD bytes for a DC (constant) signal at the given level.
// Level should be in the range [-0.8, 0.8] for modulator stability.
// dsdRate is the DSD sample rate in Hz.
// The file is written to dir, and helper is used for error reporting.
// Returns the path to the generated raw DSD file.
func DSDDC(dir string, helper tig.T, dsdRate int, level float64) string {
	helper.Helper()

	numSamples := int(dsdTestDuration * dsdBasePCMRate)
	pcm := make([]float64, numSamples)

	for sample := range numSamples {
		pcm[sample] = level
	}

	oversampleRatio := dsdRate / dsdBasePCMRate
	dsdBytes := sigmaDeltaModulate(pcm, oversampleRatio)

	outputPath := filepath.Join(dir, fmt.Sprintf("dsd-dc-%.2f-%d.raw", level, dsdRate))
	writeDSDFile(helper, outputPath, dsdBytes)

	return outputPath
}

// sigmaDeltaModulate converts PCM float64 samples to packed DSD bytes (MSB first).
//
// Uses a second-order CIFB (Cascade of Integrators, Feedback Form) sigma-delta
// modulator. Each PCM sample is repeated oversampleRatio times (zero-order hold)
// before modulation.
//
// The output is a byte slice with DSD bits packed MSB-first, matching the
// standard DSD convention used by DSF files and most DSD hardware.
func sigmaDeltaModulate(pcm []float64, oversampleRatio int) []byte {
	totalBits := len(pcm) * oversampleRatio

	// Pad to full bytes.
	totalBytes := (totalBits + bitsPerByte - 1) / bitsPerByte
	packed := make([]byte, totalBytes)

	// Second-order integrator state.
	var integrator1, integrator2 float64

	bitIndex := 0

	for _, sample := range pcm {
		for range oversampleRatio {
			// Quantizer: output +1 or -1 based on second integrator.
			var quantized float64
			if integrator2 >= 0 {
				quantized = dsdQuantizerPositive
				// Set bit (MSB first within each byte).
				packed[bitIndex/bitsPerByte] |= 1 << (bitsPerByte - 1 - bitIndex%bitsPerByte)
			} else {
				quantized = dsdQuantizerNegative
			}

			// Error feedback.
			err := sample - quantized

			// Update integrators (CIFB second-order).
			integrator1 += err
			integrator2 += integrator1 + err

			bitIndex++
		}
	}

	return packed
}

func writeDSDFile(helper tig.T, path string, data []byte) {
	helper.Helper()

	if err := os.WriteFile(path, data, filesystem.FilePermissionsPrivate); err != nil {
		helper.Log("writing DSD file: " + err.Error())
		helper.FailNow()
	}
}
