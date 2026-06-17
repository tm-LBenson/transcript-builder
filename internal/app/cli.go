package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
)

func Main(args []string, build BuildInfo) int {
	return MainWithIO(context.Background(), args, os.Stdout, os.Stderr, build)
}

func MainWithIO(ctx context.Context, args []string, stdout, stderr io.Writer, build BuildInfo) int {
	if len(args) < 2 {
		printUsage(stderr)
		return exitInvalidInput
	}

	switch args[1] {
	case "doctor":
		opts, err := parseDoctor(args[2:])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return exitInvalidInput
		}
		return runDoctor(ctx, stdout, stderr, build, opts)
	case "run":
		opts, err := parseRun(args[2:])
		if err != nil {
			fmt.Fprintln(stderr, err)
			return exitInvalidInput
		}
		return runTranscription(ctx, stdout, stderr, build, opts)
	case "version":
		fmt.Fprintf(stdout, "meeting-transcriber %s\n", build.String())
		if build.Date != "" && build.Date != "unknown" {
			fmt.Fprintf(stdout, "built: %s\n", build.Date)
		}
		return exitOK
	case "-h", "--help", "help":
		printUsage(stdout)
		return exitOK
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[1])
		printUsage(stderr)
		return exitInvalidInput
	}
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  meeting-transcriber doctor [flags]")
	fmt.Fprintln(w, "  meeting-transcriber run --input <file-or-dir> --model <model.bin> [flags]")
	fmt.Fprintln(w, "  meeting-transcriber version")
}

func addCommonFlags(fs *flag.FlagSet, opts *commonOptions) {
	fs.StringVar(&opts.FFmpeg, "ffmpeg", "", "path to ffmpeg.exe")
	fs.StringVar(&opts.FFprobe, "ffprobe", "", "path to ffprobe.exe")
	fs.StringVar(&opts.Whisper, "whisper", "", "path to whisper-cli.exe")
	fs.StringVar(&opts.Model, "model", "", "path to whisper.cpp model file")
	fs.StringVar(&opts.ConfigPath, "config", "", "optional config path")
	fs.BoolVar(&opts.Verbose, "verbose", false, "enable verbose output")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "show what would happen without processing files")
}

func parseDoctor(args []string) (doctorOptions, error) {
	var opts doctorOptions
	opts.NotesProvider = defaultNotesProvider
	opts.OllamaURL = defaultOllamaURL

	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addCommonFlags(fs, &opts.commonOptions)
	fs.StringVar(&opts.OutputDir, "output", "", "output directory to check for writability")
	fs.StringVar(&opts.NotesProvider, "notes-provider", opts.NotesProvider, "notes provider: none, heuristic, ollama")
	fs.StringVar(&opts.OllamaURL, "ollama-url", opts.OllamaURL, "Ollama generate API URL")
	fs.StringVar(&opts.OllamaModel, "ollama-model", "", "Ollama model name")
	fs.BoolVar(&opts.AllowCloudModel, "allow-cloud-model", false, "allow model names containing cloud")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if err := validateNotesProvider(opts.NotesProvider); err != nil {
		return opts, err
	}
	return opts, nil
}

func parseRun(args []string) (runOptions, error) {
	var opts runOptions
	opts.Language = defaultLanguage
	opts.Threads = max(runtime.NumCPU()-1, 1)
	opts.KeepWAV = true
	opts.NotesProvider = defaultNotesProvider
	opts.OllamaURL = defaultOllamaURL

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addCommonFlags(fs, &opts.commonOptions)
	fs.StringVar(&opts.Input, "input", "", "input media file or directory")
	fs.StringVar(&opts.Output, "output", "", "output directory")
	fs.BoolVar(&opts.Recursive, "recursive", false, "scan input directories recursively")
	fs.StringVar(&opts.Language, "language", opts.Language, "Whisper language code")
	fs.IntVar(&opts.Threads, "threads", opts.Threads, "Whisper thread count")
	fs.BoolVar(&opts.KeepWAV, "keep-wav", opts.KeepWAV, "keep extracted WAV")
	fs.BoolVar(&opts.CleanIntermediate, "clean-intermediate", false, "delete WAV after successful transcription")
	fs.StringVar(&opts.NotesProvider, "notes-provider", opts.NotesProvider, "notes provider: none, heuristic, ollama")
	fs.StringVar(&opts.OllamaURL, "ollama-url", opts.OllamaURL, "Ollama generate API URL")
	fs.StringVar(&opts.OllamaModel, "ollama-model", "", "Ollama model name")
	fs.BoolVar(&opts.AllowCloudModel, "allow-cloud-model", false, "allow model names containing cloud")
	fs.BoolVar(&opts.Force, "force", false, "allow overwrite when a generated path already exists")
	fs.BoolVar(&opts.FailFast, "fail-fast", false, "stop batch processing after the first failure")
	fs.BoolVar(&opts.VerboseTranscriptLog, "verbose-transcript-log", false, "allow transcript text in run.log")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if opts.Input == "" {
		return opts, errors.New("missing required --input")
	}
	if opts.Model == "" {
		return opts, errors.New("missing required --model")
	}
	if opts.Threads < 1 {
		return opts, errors.New("--threads must be at least 1")
	}
	if opts.CleanIntermediate {
		opts.KeepWAV = false
	}
	if err := validateNotesProvider(opts.NotesProvider); err != nil {
		return opts, err
	}
	if opts.NotesProvider == "ollama" && opts.OllamaModel == "" {
		return opts, errors.New("--ollama-model is required when --notes-provider ollama")
	}
	return opts, nil
}

func validateNotesProvider(provider string) error {
	switch provider {
	case "none", "heuristic", "ollama":
		return nil
	default:
		return fmt.Errorf("unsupported --notes-provider %q; use none, heuristic, or ollama", provider)
	}
}
