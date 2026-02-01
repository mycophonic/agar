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

//nolint:tagliatelle // ffprobe JSON uses snake_case field names
package agar

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
)

const (
	// defaultBitDepth is the fallback bit depth when ffprobe does not report one
	// (typical for lossy codecs like MP3/AAC/Opus).
	defaultBitDepth = 16

	// float64Bits is the bit size argument for strconv.ParseFloat.
	float64Bits = 64
)

// ErrNoAudioStream is returned when no audio stream is found in the probe result.
var ErrNoAudioStream = errors.New("no audio stream found")

// FFProbeResult contains the parsed JSON output of ffprobe.
type FFProbeResult struct {
	Streams []FFProbeStream `json:"streams"`
	Format  FFProbeFormat   `json:"format"`
}

// FFProbeStream represents a single stream from ffprobe output.
type FFProbeStream struct {
	Index            int    `json:"index"`
	CodecName        string `json:"codec_name"`
	CodecType        string `json:"codec_type"`
	SampleRate       string `json:"sample_rate,omitempty"`
	Channels         int    `json:"channels,omitempty"`
	ChannelLayout    string `json:"channel_layout,omitempty"`
	BitsPerRawSample string `json:"bits_per_raw_sample,omitempty"`
	BitsPerSample    int    `json:"bits_per_sample,omitempty"`
	Duration         string `json:"duration,omitempty"`
	BitRate          string `json:"bit_rate,omitempty"`
	SampleFmt        string `json:"sample_fmt,omitempty"`
	NbFrames         string `json:"nb_frames,omitempty"`
	DurationTS       int64  `json:"duration_ts,omitempty"`
	TimeBase         string `json:"time_base,omitempty"`
}

// FFProbeFormat represents container-level metadata from ffprobe.
type FFProbeFormat struct {
	Filename   string `json:"filename"`
	NbStreams  int    `json:"nb_streams"`
	FormatName string `json:"format_name"`
	Duration   string `json:"duration,omitempty"`
	Size       string `json:"size,omitempty"`
	BitRate    string `json:"bit_rate,omitempty"`
	ProbeScore int    `json:"probe_score"`
}

// FFProbe runs ffprobe on the given file and returns parsed JSON metadata.
// It probes both streams and format information.
func FFProbe(path string) (*FFProbeResult, error) {
	ffprobePath, err := LookFor(ffprobeBinary)
	if err != nil {
		return nil, err
	}

	//nolint:gosec // path is intentionally user-provided
	cmd := exec.CommandContext(context.Background(), ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	)

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe %s: %w: %s", path, err, stderr.String())
	}

	var result FFProbeResult
	if err = json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("ffprobe JSON parse: %w", err)
	}

	return &result, nil
}

// AudioStream returns the first audio stream, or ErrNoAudioStream if none found.
func (r *FFProbeResult) AudioStream() (*FFProbeStream, error) {
	for i := range r.Streams {
		if r.Streams[i].CodecType == "audio" {
			return &r.Streams[i], nil
		}
	}

	return nil, ErrNoAudioStream
}

// BitDepth returns the effective bit depth for the stream.
// It prefers BitsPerRawSample (most reliable for lossless codecs like FLAC/ALAC),
// falls back to BitsPerSample (reliable for WAV/AIFF), then defaults to 16.
func (s *FFProbeStream) BitDepth() int {
	if s.BitsPerRawSample != "" {
		if v, err := strconv.Atoi(s.BitsPerRawSample); err == nil && v > 0 {
			return v
		}
	}

	if s.BitsPerSample > 0 {
		return s.BitsPerSample
	}

	return defaultBitDepth
}

// SampleRateInt returns the sample rate as an integer.
func (s *FFProbeStream) SampleRateInt() int {
	v, _ := strconv.Atoi(s.SampleRate)

	return v
}

// DurationFloat returns the stream duration as a float64 (seconds).
func (s *FFProbeStream) DurationFloat() float64 {
	v, _ := strconv.ParseFloat(s.Duration, float64Bits)

	return v
}
