// Package testutil provides testing utilities for hypha integration tests using tigron.
package agar

import (
	"os"
	"os/exec"

	"github.com/containerd/nerdctl/mod/tigron/test"
	"github.com/containerd/nerdctl/mod/tigron/tig"
)

// agarSetup implements test.Testable for hypha CLI testing.
type agarSetup struct {
	binary string
}

// CustomCommand returns a command configured with the hypha binary.
func (hs *agarSetup) CustomCommand(_ *test.Case, testing tig.T) test.CustomizableCommand {
	cmd := test.NewGenericCommand()
	cmd.WithBinary(hs.binary)

	gen := *(cmd.(*test.GenericCommand))
	gen.WithWhitelist([]string{
		"PATH",
		"HOME",
		"XDG_*",
	})

	return &gen
}

// AmbientRequirements checks environment prerequisites.
func (hs *agarSetup) AmbientRequirements(_ *test.Case, testing tig.T) {
	if _, err := exec.LookPath(ffprobeBinary); err != nil {
		testing.Skip("ffprobe not found in PATH")
	}

	if _, err := exec.LookPath(ffmpegBinary); err != nil {
		testing.Skip("ffmpeg not found in PATH")
	}

	if _, err := exec.LookPath(metaflacBinary); err != nil {
		testing.Skip("ffmpeg not found in PATH")
	}

	if _, err := exec.LookPath(soxBinary); err != nil {
		testing.Skip("sox not found in PATH")
	}

	if _, err := os.Stat(hs.binary); err != nil {
		// Binary not found at given path, try PATH lookup
		if path, err := exec.LookPath(hs.binary); err == nil {
			hs.binary = path
		} else {
			testing.Log("binary %s not found: run 'make build' or install your binary in PATH", hs.binary)
			testing.FailNow()
		}
	}
	// else: binary exists at the given path, use it as-is
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
