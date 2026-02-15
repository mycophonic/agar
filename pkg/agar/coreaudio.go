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
	"errors"
	"testing"
	"time"

	"github.com/mycophonic/agar/pkg/coreaudio"
)

// BenchDecodeCodec benchmarks decoding using the given coreaudio.Codec.
// It skips the test if the codec is not available.
func BenchDecodeCodec(
	t *testing.T,
	codec coreaudio.Codec,
	name string,
	format BenchFormat,
	opts BenchOptions,
	encoded []byte,
) BenchResult {
	t.Helper()

	if !codec.Available() {
		t.Skipf("%s codec not available", name)
	}

	// Verify decode works before benchmarking.
	if _, _, err := codec.Decode(encoded); err != nil {
		if errors.Is(err, coreaudio.ErrUnavailable) {
			t.Skipf("%s not available on this platform", name)
		}

		t.Fatalf("%s: %v", name, err)
	}

	opts = opts.WithDefaults()
	durations := make([]time.Duration, opts.Iterations)

	for iter := range opts.Iterations {
		start := time.Now()

		_, _, err := codec.Decode(encoded)
		if err != nil {
			t.Fatalf("%s iter %d: %v", name, iter, err)
		}

		durations[iter] = time.Since(start)
	}

	return ComputeResult(format, name, "decode", durations, len(encoded))
}

// BenchDecodeCoreAudio benchmarks CoreAudio decoding via CGO (in-process).
// It skips the test if CoreAudio is not available on the current platform.
func BenchDecodeCoreAudio(t *testing.T, format BenchFormat, opts BenchOptions, encoded []byte) BenchResult {
	t.Helper()

	return BenchDecodeCodec(t, coreaudio.NewCGO(), "coreaudio", format, opts, encoded)
}
