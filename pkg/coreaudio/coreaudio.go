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

// Package coreaudio provides ALAC encoding and decoding via macOS AudioToolbox.
// Two implementations are available:
//   - CGO: In-process via AudioToolbox (darwin only, requires CGO)
//   - Binary: Shell-out to alac-coreaudio binary (requires binary to be built)
package coreaudio

import "errors"

// Format describes audio format metadata.
type Format struct {
	SampleRate int
	BitDepth   int
	Channels   int
}

// Codec provides ALAC encoding and decoding.
type Codec interface {
	// Encode encodes raw interleaved signed LE PCM to ALAC M4A.
	// Returns the encoded M4A data.
	Encode(pcm []byte, format Format) ([]byte, error)

	// Decode decodes ALAC (or other AudioToolbox-supported format) to raw PCM.
	// Returns raw interleaved signed LE PCM and the detected format.
	Decode(data []byte) (pcm []byte, format Format, err error)

	// Available reports whether this codec implementation is usable.
	Available() bool
}

// ErrUnavailable is returned when the codec implementation is not available.
var ErrUnavailable = errors.New("coreaudio: implementation not available")

// ErrEncodeFailed is returned when encoding fails.
var ErrEncodeFailed = errors.New("coreaudio: encode failed")

// ErrDecodeFailed is returned when decoding fails.
var ErrDecodeFailed = errors.New("coreaudio: decode failed")
