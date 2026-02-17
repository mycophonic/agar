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
	"context"
	"os"
	"os/exec"
	"strings"

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

	requireSoxNG(helper)

	if _, err := os.Stat(hs.binary); err != nil {
		helper.Log(hs.binary + " not found: run 'make build' or install in PATH")
		helper.FailNow()
	}
}

// requireSoxNG verifies that the installed sox binary is sox_ng (which provides DSD support).
// Standard sox lacks DSF/DFF I/O and the sdm effect needed for DSD test file generation.
func requireSoxNG(helper tig.T) {
	helper.Helper()

	soxPath, err := LookFor(soxBinary)
	if err != nil {
		return // already handled by the binary check above
	}

	//nolint:gosec // soxPath comes from LookFor
	out, err := exec.CommandContext(
		context.Background(), soxPath, "--version",
	).Output()
	if err != nil {
		helper.Skip("sox --version failed: " + err.Error())
	}

	if !strings.Contains(string(out), "SoX_ng") {
		helper.Skip("sox is not sox_ng (missing DSD support); install with: brew install sox_ng")
	}
}

// Setup initializes tigron with minimal customization and returns a base test case.
// The binary is resolved to an absolute path here so that the shared agarSetup
// instance is immutable once concurrent subtests begin.
func Setup(binary string) *test.Case {
	path, err := LookFor(binary)
	if err != nil {
		// LookFor failed at setup time — AmbientRequirements will catch this
		// via os.Stat and fail/skip the test with a helpful message.
		path = binary
	}

	test.Customize(&agarSetup{
		binary: path,
	})

	return &test.Case{
		Env: map[string]string{},
	}
}
