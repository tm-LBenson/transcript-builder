package media

import (
	"context"
	"os/exec"
	"time"
)

func FFprobeArgs(input string) []string {
	return []string{"-v", "error", "-show_format", "-show_streams", "-print_format", "json", input}
}

func Probe(ctx context.Context, ffprobe, input string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffprobe, FFprobeArgs(input)...)
	return cmd.CombinedOutput()
}
