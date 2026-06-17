package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/tm-LBenson/transcript-builder/internal/discovery"
	"github.com/tm-LBenson/transcript-builder/internal/notes"
	"github.com/tm-LBenson/transcript-builder/internal/transcribe"
)

func runDoctor(ctx context.Context, stdout, stderr io.Writer, build BuildInfo, opts doctorOptions) int {
	fmt.Fprintf(stdout, "meeting-transcriber %s\n", build.String())
	fmt.Fprintf(stdout, "OS: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)

	status := exitOK
	result := discovery.Discover(discovery.ExplicitPaths{
		FFmpeg:  opts.FFmpeg,
		FFprobe: opts.FFprobe,
		Whisper: opts.Whisper,
	})

	if result.Tools.FFmpeg != "" {
		fmt.Fprintf(stdout, "OK: ffmpeg.exe: %s\n", result.Tools.FFmpeg)
		if version := firstLines(commandOutput(ctx, result.Tools.FFmpeg, "-version"), 1); version != "" {
			fmt.Fprintf(stdout, "    %s\n", version)
		}
	}
	if result.Tools.FFprobe != "" {
		fmt.Fprintf(stdout, "OK: ffprobe.exe: %s\n", result.Tools.FFprobe)
	}
	if result.Tools.Whisper != "" {
		fmt.Fprintf(stdout, "OK: whisper-cli.exe: %s\n", result.Tools.Whisper)
		if help, err := transcribe.Help(ctx, result.Tools.Whisper); err == nil {
			if lines := firstLines(help, 4); lines != "" {
				fmt.Fprintf(stdout, "    whisper help:\n%s\n", indent(lines, "      "))
			}
		}
	}

	for _, err := range result.Errors {
		status = exitMissingDependency
		fmt.Fprintf(stderr, "ERROR: %s.\n", err)
		if fix := discovery.MissingToolFix(err); fix != "" {
			fmt.Fprintln(stderr, fix)
		}
	}

	if opts.Model != "" {
		if err := readableFile(opts.Model); err != nil {
			if status == exitOK {
				status = exitInvalidInput
			}
			fmt.Fprintf(stderr, "ERROR: model file is not readable: %s\n", err)
			fmt.Fprintln(stderr, "Fix: pass --model \"C:\\path\\to\\ggml-small.en.bin\" or place the model where your config points.")
		} else {
			fmt.Fprintf(stdout, "OK: model file: %s\n", opts.Model)
		}
	} else {
		fmt.Fprintln(stdout, "INFO: no --model provided; model readability was not checked.")
	}

	if opts.OutputDir != "" {
		if err := writableDir(opts.OutputDir); err != nil {
			if status == exitOK {
				status = exitInvalidInput
			}
			fmt.Fprintf(stderr, "ERROR: output directory is not writable: %s\n", err)
			fmt.Fprintln(stderr, "Fix: choose a writable --output directory or create it first.")
		} else {
			fmt.Fprintf(stdout, "OK: output directory is writable: %s\n", opts.OutputDir)
		}
	}

	if opts.NotesProvider == "ollama" {
		if err := notes.ValidateOllamaOptions(notes.OllamaOptions{
			URL:             opts.OllamaURL,
			Model:           opts.OllamaModel,
			AllowCloudModel: opts.AllowCloudModel,
		}); err != nil {
			if status == exitOK {
				status = exitInvalidInput
			}
			fmt.Fprintf(stderr, "ERROR: Ollama settings failed privacy validation: %s\n", err)
		} else if err := pingOllama(ctx, opts.OllamaURL); err != nil {
			if status == exitOK {
				status = exitMissingDependency
			}
			fmt.Fprintf(stderr, "ERROR: Ollama is not reachable at %s: %s\n", opts.OllamaURL, err)
			fmt.Fprintln(stderr, "Fix: start Ollama locally or use --notes-provider heuristic.")
		} else {
			fmt.Fprintf(stdout, "OK: Ollama reachable at %s\n", opts.OllamaURL)
		}
	}

	if status == exitOK {
		fmt.Fprintln(stdout, "\nDoctor check passed.")
	}
	return status
}

func readableFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	return file.Close()
}

func writableDir(path string) error {
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	probe, err := os.CreateTemp(path, ".meeting-transcriber-write-test-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if closeErr := probe.Close(); closeErr != nil {
		return closeErr
	}
	return os.Remove(name)
}

func commandOutput(ctx context.Context, path string, args ...string) []byte {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(ctx, path, args...).CombinedOutput()
	return out
}

func firstLines(raw []byte, n int) string {
	if len(raw) == 0 || n <= 0 {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	return strings.Join(lines, "\n")
}

func pingOllama(ctx context.Context, endpoint string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	parsed.Path = "/api/tags"
	parsed.RawQuery = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return nil
}
