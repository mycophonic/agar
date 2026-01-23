package agar

import (
	"context"
	"os/exec"

	"github.com/containerd/nerdctl/mod/tigron/test"
)

// generate runs an ffmpeg command to create an audio file in the test's temp directory.
func generate(helpers test.Helpers, outputPath string, args []string) string {
	helpers.T().Helper()

	fullArgs := append([]string{"-y"}, args...)
	fullArgs = append(fullArgs, outputPath)

	helpers.Custom(ffmpegBinary, fullArgs...).Run(&test.Expected{})

	return outputPath
}

// generateWithPipe runs two ffmpeg commands piped together.
func generateWithPipe(helpers test.Helpers, outputPath string, firstArgs, secondArgs []string) string {
	helpers.T().Helper()

	ctx := context.Background()

	first := exec.CommandContext(ctx, ffmpegBinary, append([]string{"-y"}, firstArgs...)...)

	second := exec.CommandContext(ctx, ffmpegBinary, append([]string{"-y", "-i", "-"}, secondArgs...)...)
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
