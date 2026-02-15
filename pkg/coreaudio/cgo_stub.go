//go:build !(darwin && cgo)

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

// cgoCodec stub for non-darwin platforms.
type cgoCodec struct{}

// NewCGO returns a Codec stub that always returns ErrUnavailable.
func NewCGO() Codec {
	return &cgoCodec{}
}

// Available reports whether CGO AudioToolbox is available.
func (*cgoCodec) Available() bool {
	return false
}

// Decode is unavailable on non-darwin platforms.
func (*cgoCodec) Decode(_ []byte) ([]byte, Format, error) {
	return nil, Format{}, ErrUnavailable
}

// Encode is unavailable on non-darwin platforms.
func (*cgoCodec) Encode(_ []byte, _ Format) ([]byte, error) {
	return nil, ErrUnavailable
}
