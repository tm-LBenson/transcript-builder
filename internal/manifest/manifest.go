package manifest

import (
	"encoding/json"
	"os"
	"time"
)

type Manifest struct {
	App                  string          `json:"app"`
	Version              string          `json:"version"`
	ProcessedAt          time.Time       `json:"processed_at"`
	SourceFile           string          `json:"source_file"`
	OutputDir            string          `json:"output_dir"`
	AudioWAV             string          `json:"audio_wav,omitempty"`
	TranscriptTXT        string          `json:"transcript_txt,omitempty"`
	TranscriptSRT        string          `json:"transcript_srt,omitempty"`
	TranscriptJSON       string          `json:"transcript_json,omitempty"`
	NotesMD              string          `json:"notes_md,omitempty"`
	FFmpegPath           string          `json:"ffmpeg_path,omitempty"`
	FFprobePath          string          `json:"ffprobe_path,omitempty"`
	WhisperPath          string          `json:"whisper_path,omitempty"`
	FFmpegVersion        string          `json:"ffmpeg_version,omitempty"`
	WhisperVersionOrHelp string          `json:"whisper_version_or_help,omitempty"`
	MediaInfo            json.RawMessage `json:"media_info,omitempty"`
	Model                string          `json:"model"`
	Language             string          `json:"language"`
	NotesProvider        string          `json:"notes_provider"`
	Status               string          `json:"status"`
	Errors               []string        `json:"errors"`
}

func Write(path string, m Manifest) error {
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}
