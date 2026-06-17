package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	err := Write(path, Manifest{
		App:           "meeting-transcriber",
		Version:       "0.1.0",
		ProcessedAt:   time.Date(2026, 6, 17, 14, 30, 22, 0, time.UTC),
		SourceFile:    "call.mp4",
		OutputDir:     "out",
		TranscriptTXT: "transcript.txt",
		Status:        "success",
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Manifest
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Status != "success" || decoded.App != "meeting-transcriber" {
		t.Fatalf("unexpected manifest: %#v", decoded)
	}
}
