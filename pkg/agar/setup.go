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
	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"
)

// agarSetup implements test.Testable for hypha CLI testing.
type agarSetup struct {
	binary string
}

// CustomCommand returns a command configured with the hypha binary.
func (hs *agarSetup) CustomCommand(_ *test.Case, _ tig.T) test.CustomizableCommand {
	cmd := test.NewGenericCommand()
	cmd.WithBinary(hs.binary)

	gen := *(cmd.(*test.GenericCommand))
	gen.WithWhitelist([]string{
		"PATH",
		"HOME",
		"XDG_*",
		// Windows
		"SYSTEMROOT",
		"SYSTEMDRIVE",
		"COMSPEC",
		"TEMP",
		"TMP",
		"USERPROFILE",
		"PATHEXT",
	})

	return &gen
}

// AmbientRequirements checks environment prerequisites.
func (hs *agarSetup) AmbientRequirements(_ *test.Case, helper tig.T) {
	for _, bin := range []string{ffprobeBinary, ffmpegBinary, metaflacBinary, soxBinary, atomicParsleyBinary} {
		if _, err := LookFor(bin); err != nil {
			helper.Skip(bin + " not found")
		}
	}

	path, err := LookFor(hs.binary)
	if err != nil {
		helper.Log(hs.binary + " not found: run 'make build' or install in PATH")
		helper.FailNow()
	}

	hs.binary = path
}

// Setup initializes tigron with minimal customization and returns a base test case.
func Setup(binary string) *test.Case {
	test.Customize(&agarSetup{
		binary: binary,
	})

	return &test.Case{
		Env: map[string]string{},
	}
}
