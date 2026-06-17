package app

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunWithFakeTools(t *testing.T) {
	toolsDir := buildFakeTools(t)
	inputDir := t.TempDir()
	outputDir := t.TempDir()
	input := filepath.Join(inputDir, "customer call.mp4")
	model := filepath.Join(inputDir, "ggml-small.en.bin")
	mustWriteFile(t, input, "fake media")
	mustWriteFile(t, model, "fake model")

	var stdout, stderr bytes.Buffer
	code := MainWithIO(t.Context(), []string{
		"meeting-transcriber",
		"run",
		"--input", input,
		"--output", outputDir,
		"--model", model,
		"--ffmpeg", filepath.Join(toolsDir, exeName("ffmpeg")),
		"--ffprobe", filepath.Join(toolsDir, exeName("ffprobe")),
		"--whisper", filepath.Join(toolsDir, exeName("whisper-cli")),
		"--language", "en",
	}, &stdout, &stderr, BuildInfo{Version: "test"})
	if code != exitOK {
		t.Fatalf("exit code = %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one output folder, got %d", len(entries))
	}
	out := filepath.Join(outputDir, entries[0].Name())
	for _, name := range []string{"manifest.json", "run.log", "audio.wav", "transcript.txt", "transcript.srt", "transcript.json", "meeting_notes.md"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}
	rawNotes, err := os.ReadFile(filepath.Join(out, "meeting_notes.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rawNotes), "Candidate: We decided") {
		t.Fatalf("heuristic notes did not include fake transcript decision:\n%s", rawNotes)
	}
}

func buildFakeTools(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "fake_tool.go")
	if err := os.WriteFile(source, []byte(fakeToolSource), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, exeName("fake-tool"))
	cmd := exec.Command("go", "build", "-o", binary, source)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fake tool: %v\n%s", err, out)
	}
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ffmpeg", "ffprobe", "whisper-cli"} {
		if err := os.WriteFile(filepath.Join(dir, exeName(name)), raw, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func mustWriteFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

const fakeToolSource = `package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	name := strings.ToLower(filepath.Base(os.Args[0]))
	switch {
	case strings.Contains(name, "ffprobe"):
		fmt.Print(` + "`" + `{"format":{"duration":"123.4"},"streams":[{"codec_type":"audio"}]}` + "`" + `)
	case strings.Contains(name, "ffmpeg"):
		if len(os.Args) > 1 && os.Args[1] == "-version" {
			fmt.Println("ffmpeg fake")
			return
		}
		out := os.Args[len(os.Args)-1]
		if err := os.WriteFile(out, []byte("wav"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case strings.Contains(name, "whisper-cli"):
		for _, arg := range os.Args[1:] {
			if arg == "-h" {
				fmt.Println("whisper fake help")
				return
			}
		}
		var base string
		for i, arg := range os.Args {
			if arg == "-of" && i+1 < len(os.Args) {
				base = os.Args[i+1]
			}
		}
		if base == "" {
			fmt.Fprintln(os.Stderr, "missing -of")
			os.Exit(1)
		}
		transcript := "We decided the plan is to keep scope. I will follow up by Friday with the budget owner."
		_ = os.WriteFile(base+".txt", []byte(transcript+"\n"), 0600)
		_ = os.WriteFile(base+".srt", []byte("1\n00:00:00,000 --> 00:00:02,000\n"+transcript+"\n"), 0600)
		_ = os.WriteFile(base+".json", []byte(` + "`" + `{"transcription":[{"text":"ok"}]}` + "`" + `), 0600)
	default:
		fmt.Fprintln(os.Stderr, "unknown fake tool", name)
		os.Exit(1)
	}
}
`
