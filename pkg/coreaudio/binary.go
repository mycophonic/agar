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

package coreaudio

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mycophonic/primordium/filesystem"
)

// binaryCodec implements Codec by shelling out to alac-coreaudio binary.
type binaryCodec struct {
	path string
}

// NewBinary returns a Codec that uses the alac-coreaudio binary at the given path.
// If path is empty, it searches for "alac-coreaudio" in PATH.
func NewBinary(path string) Codec {
	if path == "" {
		// Search in PATH.
		if p, err := exec.LookPath("alac-coreaudio"); err == nil {
			path = p
		}
	}

	return &binaryCodec{path: path}
}

// Available reports whether the binary is available.
func (b *binaryCodec) Available() bool {
	if b.path == "" {
		return false
	}

	_, err := os.Stat(b.path)

	return err == nil
}

// Decode decodes audio using the alac-coreaudio binary.
// The binary outputs format info to stderr: "sample_rate=N bit_depth=N channels=N frames=N".
func (b *binaryCodec) Decode(data []byte) ([]byte, Format, error) {
	if !b.Available() {
		return nil, Format{}, ErrUnavailable
	}

	if len(data) == 0 {
		return nil, Format{}, fmt.Errorf("%w: empty input", ErrDecodeFailed)
	}

	// Write input to temp file (binary reads from file).
	tmpDir, err := os.MkdirTemp("", "coreaudio-decode-")
	if err != nil {
		return nil, Format{}, fmt.Errorf("%w: %w", ErrDecodeFailed, err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.m4a")
	if err := os.WriteFile(inputPath, data, filesystem.FilePermissionsPrivate); err != nil {
		return nil, Format{}, fmt.Errorf("%w: %w", ErrDecodeFailed, err)
	}

	// Run binary: alac-coreaudio decode <input> -
	var stdout, stderr bytes.Buffer

	//nolint:gosec // G204: binary path is caller-controlled, inputPath is temp file we created.
	cmd := exec.CommandContext(context.Background(), b.path, "decode", inputPath, "-")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, Format{}, fmt.Errorf("%w: %w: %s", ErrDecodeFailed, err, stderr.String())
	}

	// Parse format from stderr.
	format := parseFormatFromStderr(stderr.String())

	return stdout.Bytes(), format, nil
}

// Encode encodes raw PCM to ALAC M4A using the alac-coreaudio binary.
func (b *binaryCodec) Encode(pcm []byte, format Format) ([]byte, error) {
	if !b.Available() {
		return nil, ErrUnavailable
	}

	if len(pcm) == 0 {
		return nil, fmt.Errorf("%w: empty input", ErrEncodeFailed)
	}

	if format.SampleRate <= 0 || format.BitDepth <= 0 || format.Channels <= 0 {
		return nil, fmt.Errorf("%w: invalid format", ErrEncodeFailed)
	}

	// Write PCM to temp file.
	tmpDir, err := os.MkdirTemp("", "coreaudio-encode-")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodeFailed, err)
	}
	defer os.RemoveAll(tmpDir)

	inputPath := filepath.Join(tmpDir, "input.raw")
	if err := os.WriteFile(inputPath, pcm, filesystem.FilePermissionsPrivate); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodeFailed, err)
	}

	outputPath := filepath.Join(tmpDir, "output.m4a")

	// Run binary: alac-coreaudio encode --sample-rate N --bit-depth N --channels N <input> <output>
	var stderr bytes.Buffer

	//nolint:gosec // G204: binary path is caller-controlled, paths are temp files we created.
	cmd := exec.CommandContext(context.Background(), b.path,
		"encode",
		"--sample-rate", strconv.Itoa(format.SampleRate),
		"--bit-depth", strconv.Itoa(format.BitDepth),
		"--channels", strconv.Itoa(format.Channels),
		inputPath, outputPath,
	)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%w: %w: %s", ErrEncodeFailed, err, stderr.String())
	}

	// Read encoded output.
	m4a, err := os.ReadFile(outputPath) //nolint:gosec // G304: outputPath is temp file we created.
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEncodeFailed, err)
	}

	return m4a, nil
}

// parseFormatFromStderr parses format info from the binary's stderr output.
// Expected format: "sample_rate=N bit_depth=N channels=N frames=N".
func parseFormatFromStderr(stderr string) Format {
	var format Format

	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		for part := range strings.FieldsSeq(line) {
			if after, ok := strings.CutPrefix(part, "sample_rate="); ok {
				format.SampleRate, _ = strconv.Atoi(after)
			} else if after, ok := strings.CutPrefix(part, "bit_depth="); ok {
				format.BitDepth, _ = strconv.Atoi(after)
			} else if after, ok := strings.CutPrefix(part, "channels="); ok {
				format.Channels, _ = strconv.Atoi(after)
			}
		}
	}

	return format
}
