package media

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsSupported(t *testing.T) {
	for _, path := range []string{"call.MP4", "voice.m4a", "meeting.3gp", "audio.wav"} {
		if !IsSupported(path) {
			t.Fatalf("%s should be supported", path)
		}
	}
	if IsSupported("notes.md") {
		t.Fatal("notes.md should not be supported")
	}
}

func TestScanNonRecursiveSkipsGeneratedAndSubdirectories(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "meeting.mp4"), "video")
	mustWrite(t, filepath.Join(root, "transcript.txt"), "generated")
	mustWrite(t, filepath.Join(root, "audio.wav"), "generated")
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "nested", "nested.mp4"), "video")

	got, err := Scan(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "meeting.mp4" {
		t.Fatalf("unexpected scan result: %#v", got)
	}
}

func TestScanRecursiveSkipsGeneratedOutputDirs(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "meeting.mp4"), "video")
	if err := os.Mkdir(filepath.Join(root, "_transcribed"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "_transcribed", "old.mp4"), "video")
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(root, "nested", "nested.m4a"), "audio")

	got, err := Scan(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 files, got %#v", got)
	}
	for _, path := range got {
		if strings.Contains(path, "_transcribed") {
			t.Fatalf("generated output dir was not skipped: %#v", got)
		}
	}
}

func TestUniqueOutputDirSanitizesAndAvoidsCollision(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "Customer Call 2026!.mp4")
	now := time.Date(2026, 6, 17, 14, 30, 22, 0, time.UTC)

	first, err := UniqueOutputDir(root, source, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(filepath.Base(first), "Customer_Call_2026") {
		t.Fatalf("expected sanitized base name, got %s", first)
	}
	if err := os.Mkdir(first, 0o755); err != nil {
		t.Fatal(err)
	}
	second, err := UniqueOutputDir(root, source, now)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(second, "-2") {
		t.Fatalf("expected collision suffix, got %s", second)
	}
}

func TestDurationFromProbe(t *testing.T) {
	got := DurationFromProbe([]byte(`{"format":{"duration":"3661.49"}}`))
	if got != "1:01:01" {
		t.Fatalf("DurationFromProbe = %q", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
