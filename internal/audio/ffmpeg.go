package audio

import (
	"context"
	"os/exec"
	"time"
)

type ExtractOptions struct {
	FFmpeg  string
	Input   string
	Output  string
	Timeout time.Duration
}

func FFmpegArgs(input, output string) []string {
	return []string{
		"-nostdin",
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", input,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-c:a", "pcm_s16le",
		output,
	}
}

func ExtractWAV(ctx context.Context, opts ExtractOptions) ([]byte, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 6 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opts.FFmpeg, FFmpegArgs(opts.Input, opts.Output)...)
	return cmd.CombinedOutput()
}
