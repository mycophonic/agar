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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// ErrBinaryNotFound is returned when a binary cannot be located.
var ErrBinaryNotFound = errors.New("binary not found")

// LookFor resolves a binary name to a full path.
// If the binary is not an absolute path, it first searches for it in the caller's
// project bin/ directory (walking the call stack to find go.mod), then falls back
// to exec.LookPath.
// On Windows, it automatically appends ".exe" when the path has no extension.
func LookFor(binary string) (string, error) {
	if filepath.IsAbs(binary) {
		return checkBinary(binary)
	}

	// Try each caller's project bin/ directory.
	for _, root := range callerProjectRoots() {
		candidate := filepath.Join(root, "bin", binary)
		if path, err := checkBinary(candidate); err == nil {
			return path, nil
		}
	}

	// Fall back to PATH lookup.
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("%w: %s: %w", ErrBinaryNotFound, binary, err)
	}

	return path, nil
}

// lookForOrFail resolves a binary or fails the test.
func lookForOrFail(helper interface {
	Helper()
	Log(args ...any)
	FailNow()
}, binary string,
) string {
	helper.Helper()

	path, err := LookFor(binary)
	if err != nil {
		helper.Log(binary + ": " + err.Error())
		helper.FailNow()
	}

	return path
}

// checkBinary verifies a binary exists at the given path, handling .exe on Windows.
func checkBinary(path string) (string, error) {
	if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
		withExt := path + ".exe"
		if info, err := os.Stat(withExt); err == nil && !info.IsDir() {
			return withExt, nil
		}
	}

	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path, nil
	}

	return "", fmt.Errorf("%w: %s", ErrBinaryNotFound, path)
}

// callerProjectRoots walks the call stack and returns unique project root
// directories (those containing a go.mod file).
func callerProjectRoots() []string {
	var pcs [20]uintptr

	n := runtime.Callers(3, pcs[:]) // skip: Callers, callerProjectRoots, LookFor

	frames := runtime.CallersFrames(pcs[:n])

	seen := map[string]bool{}

	var roots []string

	for {
		frame, more := frames.Next()
		if frame.File != "" {
			if root := findModuleRoot(filepath.Dir(frame.File)); root != "" && !seen[root] {
				seen[root] = true
				roots = append(roots, root)
			}
		}

		if !more {
			break
		}
	}

	return roots
}

// findModuleRoot walks up from dir looking for a go.mod file.
func findModuleRoot(dir string) string {
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}

		dir = parent
	}
}
