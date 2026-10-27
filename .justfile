# This file is the project's own.
# Add recipes leveraging provided `do` ready-made recipes, or create your own.
# The import must be kept: it mounts every shared limen task under `just do ...`.
import '.limen/just/main.just'

# The FIRST recipe defined here becomes `just`'s default.
# The shared default covers the language-agnostic linters (limen, just, aqua,
# yaml, shell, dockerfile, commits); the Go linters (golangci-lint, govulncheck,
# go mod tidy, go-licenses) live in their own submodule and are added explicitly,
# and lint-cgo adds the cgo build on macOS.
[doc('Lint everything (shared linters + Go linters)')]
lint: do::lint::default do::lint::go::default lint-cgo

# Shared fixers plus the Go fixers (golangci-lint --fix, go mod tidy).
[doc('Auto-fix what can be fixed (shared fixers + Go fixers)')]
fix: do::fix::default do::fix::go::default

# Unit, race and cover. Benchmarks are a performance measure, not a test:
# `just do perf go bench`.
[doc('Run the Go test suite: unit, race, cover')]
test: do::test::go::unit do::test::go::race do::test::go::cover

# The shared security scans (govulncheck, for a Go module); the security
# workflow runs this recipe.
[doc('Run the security scans')]
security: do::security::default

# The shared lanes lint with cgo off, on every platform, so they never see
# pkg/coreaudio/cgo_darwin.go (darwin && cgo), the in-process AudioToolbox
# decoder; the stub stands in for it everywhere. On macOS, this runs the
# golangci-lint lane again with cgo on, natively, which is the only build that
# compiles that file. On other systems there is nothing more to see: the file
# never builds there.
[doc('Lint the cgo build (macOS only: the AudioToolbox decoder in pkg/coreaudio)')]
lint-cgo:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ "$(uname -s)" != Darwin ]; then
        echo "lint-cgo: the cgo build exists on macOS only (skipped, not failed)."
        exit 0
    fi
    CGO_ENABLED=1 "{{ just_executable() }}" do lint go code
