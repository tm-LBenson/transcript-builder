package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := MainWithIO(t.Context(), []string{"meeting-transcriber", "version"}, &stdout, &stderr, BuildInfo{Version: "9.9.9"})
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "meeting-transcriber 9.9.9") {
		t.Fatalf("unexpected stdout: %s", stdout.String())
	}
}

func TestParseRunRequiresInputAndModel(t *testing.T) {
	if _, err := parseRun(nil); err == nil {
		t.Fatal("expected missing input error")
	}
	if _, err := parseRun([]string{"--input", "call.mp4"}); err == nil {
		t.Fatal("expected missing model error")
	}
}

func TestRunHelpExitsOK(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := MainWithIO(t.Context(), []string{"meeting-transcriber", "run", "--help"}, &stdout, &stderr, BuildInfo{Version: "0.1.0"})
	if code != exitOK {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "run flags:") {
		t.Fatalf("expected run help, got %s", stdout.String())
	}
}

func TestConfigFlagErrorsClearly(t *testing.T) {
	_, err := parseRun([]string{"--input", "call.mp4", "--model", "model.bin", "--config", "config.json"})
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("expected clear config error, got %v", err)
	}
}

func TestRunDryRunDoesNotRequireToolsOrReadableModel(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "meeting.mp4")
	if err := os.WriteFile(input, []byte("fake"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := MainWithIO(t.Context(), []string{
		"meeting-transcriber",
		"run",
		"--input", input,
		"--model", filepath.Join(dir, "missing-model.bin"),
		"--ffmpeg", "Z:\\missing\\ffmpeg.exe",
		"--ffprobe", "Z:\\missing\\ffprobe.exe",
		"--whisper", "Z:\\missing\\whisper-cli.exe",
		"--dry-run",
	}, &stdout, &stderr, BuildInfo{Version: "0.1.0"})
	if code != exitOK {
		t.Fatalf("exit code = %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "DRY RUN: would process") {
		t.Fatalf("expected dry-run output, got %s", stdout.String())
	}
}

func TestDoctorMissingExplicitTools(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := MainWithIO(t.Context(), []string{
		"meeting-transcriber",
		"doctor",
		"--ffmpeg", "Z:\\missing\\ffmpeg.exe",
		"--ffprobe", "Z:\\missing\\ffprobe.exe",
		"--whisper", "Z:\\missing\\whisper-cli.exe",
	}, &stdout, &stderr, BuildInfo{Version: "0.1.0"})
	if code != exitMissingDependency {
		t.Fatalf("exit code = %d, stdout = %s, stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "ffmpeg.exe not found") {
		t.Fatalf("missing actionable error: %s", stderr.String())
	}
}
