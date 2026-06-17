package transcribe

import (
	"reflect"
	"testing"
)

func TestWhisperArgsWithJSON(t *testing.T) {
	got := WhisperArgs("model.bin", "audio.wav", "en", 4, "transcript", true)
	want := []string{
		"-m", "model.bin",
		"-f", "audio.wav",
		"-l", "en",
		"-t", "4",
		"-pp",
		"-pc",
		"-otxt",
		"-osrt",
		"-ojf",
		"-of", "transcript",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WhisperArgs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestWhisperArgsWithoutJSON(t *testing.T) {
	got := WhisperArgs("model.bin", "audio.wav", "en", 0, "transcript", false)
	want := []string{
		"-m", "model.bin",
		"-f", "audio.wav",
		"-l", "en",
		"-t", "1",
		"-pp",
		"-pc",
		"-otxt",
		"-osrt",
		"-of", "transcript",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("WhisperArgs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestLooksLikeUnsupportedOJf(t *testing.T) {
	if !looksLikeUnsupportedOJf(nil, []byte("error: unknown option -ojf")) {
		t.Fatal("expected unsupported -ojf to be detected")
	}
	if looksLikeUnsupportedOJf(nil, []byte("model file not found")) {
		t.Fatal("unrelated errors should not be treated as -ojf support issues")
	}
}
