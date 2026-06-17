package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tm-LBenson/transcript-builder/internal/audio"
	"github.com/tm-LBenson/transcript-builder/internal/discovery"
	"github.com/tm-LBenson/transcript-builder/internal/logging"
	"github.com/tm-LBenson/transcript-builder/internal/manifest"
	"github.com/tm-LBenson/transcript-builder/internal/media"
	"github.com/tm-LBenson/transcript-builder/internal/notes"
	"github.com/tm-LBenson/transcript-builder/internal/transcribe"
)

func runTranscription(ctx context.Context, stdout, stderr io.Writer, build BuildInfo, opts runOptions) int {
	if err := notesProviderPrivacyCheck(opts); err != nil {
		fmt.Fprintf(stderr, "ERROR: %s\n", err)
		return exitInvalidInput
	}

	result := discovery.Discover(discovery.ExplicitPaths{
		FFmpeg:  opts.FFmpeg,
		FFprobe: opts.FFprobe,
		Whisper: opts.Whisper,
	})
	if len(result.Errors) > 0 {
		for _, err := range result.Errors {
			fmt.Fprintf(stderr, "ERROR: %s.\n", err)
			if fix := discovery.MissingToolFix(err); fix != "" {
				fmt.Fprintln(stderr, fix)
			}
		}
		return exitMissingDependency
	}
	if err := readableFile(opts.Model); err != nil {
		fmt.Fprintf(stderr, "ERROR: model file is not readable: %s\n", err)
		return exitInvalidInput
	}

	outputRoot := opts.Output
	if outputRoot == "" {
		var err error
		outputRoot, err = media.DefaultOutputDir(opts.Input)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: cannot determine output directory: %s\n", err)
			return exitInvalidInput
		}
	}

	inputs, err := media.Scan(opts.Input, opts.Recursive)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot scan input: %s\n", err)
		return exitInvalidInput
	}
	if len(inputs) == 0 {
		fmt.Fprintln(stderr, "ERROR: no supported media files found.")
		fmt.Fprintf(stderr, "Supported extensions: %s\n", strings.Join(media.SupportedExtensions(), ", "))
		return exitInvalidInput
	}

	fmt.Fprintf(stdout, "Found %d media file(s).\n", len(inputs))
	fmt.Fprintf(stdout, "Output root: %s\n", outputRoot)
	if opts.DryRun {
		for _, input := range inputs {
			fmt.Fprintf(stdout, "DRY RUN: would process %s\n", input)
		}
		return exitOK
	}
	if err := os.MkdirAll(outputRoot, 0o755); err != nil {
		fmt.Fprintf(stderr, "ERROR: cannot create output directory: %s\n", err)
		return exitInvalidInput
	}

	var failures []string
	for i, input := range inputs {
		fmt.Fprintf(stdout, "[%d/%d] Processing %s\n", i+1, len(inputs), input)
		if err := processOne(ctx, stdout, input, outputRoot, build, opts, result.Tools); err != nil {
			failures = append(failures, fmt.Sprintf("%s: %s", input, err))
			fmt.Fprintf(stderr, "ERROR: %s: %s\n", input, err)
			if opts.FailFast {
				break
			}
			continue
		}
	}

	if len(failures) > 0 {
		fmt.Fprintf(stderr, "\n%d file(s) failed:\n", len(failures))
		for _, failure := range failures {
			fmt.Fprintf(stderr, "- %s\n", failure)
		}
		return exitProcessingFailure
	}
	fmt.Fprintln(stdout, "All files processed successfully.")
	return exitOK
}

func processOne(ctx context.Context, stdout io.Writer, input, outputRoot string, build BuildInfo, opts runOptions, tools discovery.ToolPaths) error {
	processedAt := time.Now()
	outputDir, err := media.UniqueOutputDir(outputRoot, input, processedAt)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}

	runLogPath := filepath.Join(outputDir, "run.log")
	logger, err := logging.New(runLogPath)
	if err != nil {
		return err
	}
	defer logger.Close()

	manifestPath := filepath.Join(outputDir, "manifest.json")
	audioPath := filepath.Join(outputDir, "audio.wav")
	transcriptBase := filepath.Join(outputDir, "transcript")
	transcriptTXT := transcriptBase + ".txt"
	transcriptSRT := transcriptBase + ".srt"
	transcriptJSON := transcriptBase + ".json"
	notesPath := filepath.Join(outputDir, "meeting_notes.md")

	m := manifest.Manifest{
		App:            "meeting-transcriber",
		Version:        build.String(),
		ProcessedAt:    processedAt,
		SourceFile:     input,
		OutputDir:      outputDir,
		AudioWAV:       "audio.wav",
		TranscriptTXT:  "transcript.txt",
		TranscriptSRT:  "transcript.srt",
		TranscriptJSON: "transcript.json",
		NotesMD:        "meeting_notes.md",
		FFmpegPath:     tools.FFmpeg,
		FFprobePath:    tools.FFprobe,
		WhisperPath:    tools.Whisper,
		Model:          opts.Model,
		Language:       opts.Language,
		NotesProvider:  opts.NotesProvider,
		Status:         "running",
	}
	m.FFmpegVersion = firstLines(commandOutput(ctx, tools.FFmpeg, "-version"), 1)
	if help, err := transcribe.Help(ctx, tools.Whisper); err == nil {
		m.WhisperVersionOrHelp = firstLines(help, 4)
	}
	_ = manifest.Write(manifestPath, m)
	defer func() {
		if m.Status == "running" {
			m.Status = "failed"
			_ = manifest.Write(manifestPath, m)
		}
	}()

	logger.Section("start")
	logger.Printf("source: %s", input)
	logger.Printf("output: %s", outputDir)
	fmt.Fprintf(stdout, "  Output: %s\n", outputDir)

	logger.Section("ffprobe")
	probeRaw, probeErr := media.Probe(ctx, tools.FFprobe, input)
	logger.Output("ffprobe output", probeRaw, true)
	if probeErr != nil {
		logger.Printf("ffprobe error: %s", probeErr)
		m.Errors = append(m.Errors, "ffprobe: "+probeErr.Error())
	} else if json.Valid(probeRaw) {
		m.MediaInfo = append([]byte(nil), probeRaw...)
	}
	duration := media.DurationFromProbe(probeRaw)
	if err := manifest.Write(manifestPath, m); err != nil {
		return err
	}

	logger.Section("ffmpeg")
	ffmpegOutput, err := audio.ExtractWAV(ctx, audio.ExtractOptions{
		FFmpeg: tools.FFmpeg,
		Input:  input,
		Output: audioPath,
	})
	logger.Output("ffmpeg output", ffmpegOutput, true)
	if err != nil {
		m.Status = "failed"
		m.Errors = append(m.Errors, "ffmpeg: "+err.Error())
		_ = manifest.Write(manifestPath, m)
		return fmt.Errorf("extract WAV: %w", err)
	}

	logger.Section("whisper")
	whisperResult, err := transcribe.Run(ctx, transcribe.Options{
		Whisper:    tools.Whisper,
		Model:      opts.Model,
		Audio:      audioPath,
		Language:   opts.Language,
		Threads:    opts.Threads,
		OutputBase: transcriptBase,
	})
	if whisperResult.RetriedNoJPF {
		logger.Printf("whisper retry: -ojf was not supported; retried without JSON output")
	}
	logger.Output("whisper stdout", whisperResult.Stdout, opts.VerboseTranscriptLog)
	logger.Output("whisper stderr", whisperResult.Stderr, true)
	if err != nil {
		m.Status = "failed"
		m.Errors = append(m.Errors, "whisper: "+err.Error())
		_ = manifest.Write(manifestPath, m)
		return fmt.Errorf("transcribe: %w", err)
	}
	if err := ensureTranscriptFile(transcriptTXT, whisperResult.Stdout); err != nil {
		m.Status = "failed"
		m.Errors = append(m.Errors, "transcript: "+err.Error())
		_ = manifest.Write(manifestPath, m)
		return err
	}

	logger.Section("notes")
	transcriptText, err := os.ReadFile(transcriptTXT)
	if err != nil {
		m.Status = "failed"
		m.Errors = append(m.Errors, "read transcript: "+err.Error())
		_ = manifest.Write(manifestPath, m)
		return err
	}
	if err := writeNotes(ctx, notesPath, string(transcriptText), notes.MeetingMeta{
		SourceFile:     input,
		ProcessedAt:    processedAt,
		Duration:       duration,
		TranscriptFile: "transcript.txt",
		Version:        build.String(),
	}, opts); err != nil {
		m.Status = "failed"
		m.Errors = append(m.Errors, "notes: "+err.Error())
		_ = manifest.Write(manifestPath, m)
		return err
	}
	logger.Printf("notes written: %s", notesPath)

	if opts.CleanIntermediate || !opts.KeepWAV {
		logger.Section("cleanup")
		if err := os.Remove(audioPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			m.Errors = append(m.Errors, "cleanup audio.wav: "+err.Error())
			logger.Printf("cleanup error: %s", err)
		} else {
			m.AudioWAV = ""
			logger.Printf("removed audio.wav after successful transcription")
		}
	}

	if _, err := os.Stat(transcriptSRT); err != nil {
		m.TranscriptSRT = ""
	}
	if _, err := os.Stat(transcriptJSON); err != nil {
		m.TranscriptJSON = ""
	}
	m.Status = "success"
	if err := manifest.Write(manifestPath, m); err != nil {
		return err
	}
	logger.Section("complete")
	logger.Printf("status: success")
	return nil
}

func notesProviderPrivacyCheck(opts runOptions) error {
	if opts.NotesProvider != "ollama" {
		return nil
	}
	return notes.ValidateOllamaOptions(notes.OllamaOptions{
		URL:             opts.OllamaURL,
		Model:           opts.OllamaModel,
		AllowCloudModel: opts.AllowCloudModel,
	})
}

func writeNotes(ctx context.Context, path, transcript string, meta notes.MeetingMeta, opts runOptions) error {
	var body string
	switch opts.NotesProvider {
	case "none":
		body = "# Meeting Notes\n\nNotes generation was disabled with `--notes-provider none`.\n\n## Full transcript\nSee `transcript.txt`.\n"
	case "heuristic":
		body = notes.Heuristic(transcript, meta)
	case "ollama":
		generated, err := notes.OllamaClient{}.Generate(ctx, notes.OllamaOptions{
			URL:             opts.OllamaURL,
			Model:           opts.OllamaModel,
			AllowCloudModel: opts.AllowCloudModel,
		}, transcript)
		if err != nil {
			return err
		}
		body = generated
	default:
		return fmt.Errorf("unsupported notes provider %q", opts.NotesProvider)
	}
	return os.WriteFile(path, []byte(body), 0o600)
}

func ensureTranscriptFile(path string, stdout []byte) error {
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	text := strings.TrimSpace(string(stdout))
	if text == "" {
		return fmt.Errorf("whisper did not create %s and stdout was empty", filepath.Base(path))
	}
	return os.WriteFile(path, []byte(text+"\n"), 0o600)
}
