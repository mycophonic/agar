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
	"os/exec"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

// generate runs an ffmpeg command to create an audio file in the test's temp directory.
func generate(helpers test.Helpers, outputPath string, args []string) string {
	helpers.T().Helper()

	ffmpeg := lookForOrFail(helpers.T(), ffmpegBinary)

	fullArgs := append([]string{"-y"}, args...)
	fullArgs = append(fullArgs, outputPath)

	helpers.Custom(ffmpeg, fullArgs...).Run(&test.Expected{})

	return outputPath
}

// generateWithPipe runs two ffmpeg commands piped together.
func generateWithPipe(helpers test.Helpers, outputPath string, firstArgs, secondArgs []string) string {
	helpers.T().Helper()

	ffmpeg := lookForOrFail(helpers.T(), ffmpegBinary)
	ctx := context.Background()

	//nolint:gosec // this is a test helper
	first := exec.CommandContext(ctx, ffmpeg, append([]string{"-y"}, firstArgs...)...)

	//nolint:gosec // this is a test helper
	second := exec.CommandContext(ctx, ffmpeg, append([]string{"-y", "-i", "-"}, secondArgs...)...)
	second.Args = append(second.Args, outputPath)

	pipe, err := first.StdoutPipe()
	if err != nil {
		helpers.T().Log(err.Error())
		helpers.T().Fail()
	}

	second.Stdin = pipe

	if err := first.Start(); err != nil {
		helpers.T().Log(err.Error())
		helpers.T().Fail()
	}

	if output, err := second.CombinedOutput(); err != nil {
		helpers.T().Log(err.Error(), string(output))
		helpers.T().Fail()
	}

	if err := first.Wait(); err != nil {
		helpers.T().Log(err.Error())
		helpers.T().Fail()
	}

	return outputPath
}
