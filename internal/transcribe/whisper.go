package transcribe

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

type Options struct {
	Whisper    string
	Model      string
	Audio      string
	Language   string
	Threads    int
	OutputBase string
	Timeout    time.Duration
}

type Result struct {
	Stdout       []byte
	Stderr       []byte
	RetriedNoJPF bool
}

func WhisperArgs(model, audio, language string, threads int, outputBase string, jsonFull bool) []string {
	args := []string{
		"-m", model,
		"-f", audio,
		"-l", language,
		"-t", intString(threads),
		"-pp",
		"-pc",
		"-otxt",
		"-osrt",
	}
	if jsonFull {
		args = append(args, "-ojf")
	}
	args = append(args, "-of", outputBase)
	return args
}

func Help(ctx context.Context, whisper string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, whisper, "-h")
	return cmd.CombinedOutput()
}

func Run(ctx context.Context, opts Options) (Result, error) {
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 12 * time.Hour
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	args := WhisperArgs(opts.Model, opts.Audio, opts.Language, opts.Threads, opts.OutputBase, true)
	stdout, stderr, err := runCommand(ctx, opts.Whisper, args)
	if err == nil || !looksLikeUnsupportedOJf(stdout, stderr) {
		return Result{Stdout: stdout, Stderr: stderr}, err
	}

	retryArgs := WhisperArgs(opts.Model, opts.Audio, opts.Language, opts.Threads, opts.OutputBase, false)
	retryStdout, retryStderr, retryErr := runCommand(ctx, opts.Whisper, retryArgs)
	return Result{Stdout: retryStdout, Stderr: retryStderr, RetriedNoJPF: true}, retryErr
}

func runCommand(ctx context.Context, path string, args []string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return []byte(stdout.String()), []byte(stderr.String()), err
}

func looksLikeUnsupportedOJf(stdout, stderr []byte) bool {
	text := strings.ToLower(string(stdout) + "\n" + string(stderr))
	return strings.Contains(text, "ojf") &&
		(strings.Contains(text, "unknown") ||
			strings.Contains(text, "invalid") ||
			strings.Contains(text, "unrecognized") ||
			strings.Contains(text, "unsupported"))
}

func intString(n int) string {
	if n < 1 {
		n = 1
	}
	return strconvItoa(n)
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
