package audio

import (
	"reflect"
	"testing"
)

func TestFFmpegArgs(t *testing.T) {
	got := FFmpegArgs(`D:\Meetings\call one.mp4`, `D:\Out\audio.wav`)
	want := []string{
		"-nostdin",
		"-y",
		"-hide_banner",
		"-loglevel", "error",
		"-i", `D:\Meetings\call one.mp4`,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-c:a", "pcm_s16le",
		`D:\Out\audio.wav`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("FFmpegArgs mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
}
