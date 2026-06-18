package app

import "time"

const (
	exitOK                = 0
	exitProcessingFailure = 1
	exitInvalidInput      = 2
	exitMissingDependency = 3

	defaultLanguage      = "en"
	defaultNotesProvider = "none"
	defaultOllamaURL     = "http://127.0.0.1:11434/api/generate"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

func (b BuildInfo) String() string {
	if b.Commit == "" || b.Commit == "dev" {
		return b.Version
	}
	return b.Version + " (" + b.Commit + ")"
}

type commonOptions struct {
	FFmpeg     string
	FFprobe    string
	Whisper    string
	Model      string
	ConfigPath string
	Verbose    bool
	DryRun     bool
}

type doctorOptions struct {
	commonOptions
	OutputDir       string
	NotesProvider   string
	OllamaURL       string
	OllamaModel     string
	AllowCloudModel bool
}

type runOptions struct {
	commonOptions
	Input                string
	Output               string
	Recursive            bool
	Language             string
	Threads              int
	KeepWAV              bool
	CleanIntermediate    bool
	NotesProvider        string
	OllamaURL            string
	OllamaModel          string
	AllowCloudModel      bool
	Force                bool
	FailFast             bool
	VerboseTranscriptLog bool
	CommandTimeout       time.Duration
}
