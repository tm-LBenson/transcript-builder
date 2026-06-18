# meeting-transcriber

A Windows-first Go CLI for local meeting transcription. It scans a media file or folder, extracts clean WAV audio with FFmpeg, runs local Whisper transcription through `whisper.cpp`, and writes transcript files, a manifest, and a local run log.

The app is privacy-first:

- It does not upload recordings, transcripts, or notes by default.
- It does not delete source recordings.
- It keeps generated files together in one output folder so cleanup stays manual and visible.
- Network access is used only when `--notes-provider ollama` is explicitly selected, and the Ollama URL must be loopback.

## Requirements

- Windows on amd64 or arm64.
- Go 1.24 or newer to build from source.
- `ffmpeg.exe` and `ffprobe.exe`.
- `whisper-cli.exe` from [whisper.cpp](https://github.com/ggerganov/whisper.cpp).
- A local whisper.cpp model file, for example `ggml-small.en.bin`.

Dependency lookup order:

1. Explicit CLI flags: `--ffmpeg`, `--ffprobe`, `--whisper`.
2. Environment variables:
   - `MEETING_TRANSCRIBER_FFMPEG`
   - `MEETING_TRANSCRIBER_FFPROBE`
   - `MEETING_TRANSCRIBER_WHISPER`
3. App-local folders:
   - `.\tools\ffmpeg\bin\ffmpeg.exe`
   - `.\tools\ffmpeg\bin\ffprobe.exe`
   - `.\tools\whisper\whisper-cli.exe`
4. `PATH`.

## Build

```powershell
go test ./...
go build -o .\meeting-transcriber.exe .\cmd\meeting-transcriber
```

Optional release-style build metadata:

```powershell
go build -trimpath -ldflags "-s -w -X main.version=0.1.0 -X main.commit=$(git rev-parse --short HEAD) -X main.date=$(Get-Date -Format o)" -o .\meeting-transcriber.exe .\cmd\meeting-transcriber
```

## Check Setup

```powershell
.\meeting-transcriber.exe doctor `
  --model .\models\ggml-small.en.bin
```

If a dependency is missing, `doctor` prints a fix, for example:

```text
ERROR: ffmpeg.exe not found.
Fix: install ffmpeg.exe and add it to PATH, or pass --ffmpeg "C:\path\to\ffmpeg.exe". You can also set MEETING_TRANSCRIBER_FFMPEG.
```

## Transcribe

PowerShell backticks are easy to break if any trailing spaces sneak in. The least fussy option is one line:

```powershell
.\meeting-transcriber.exe run --input "D:\Meetings\Incoming\2026-06-17-customer-call\meeting.mp4" --output "D:\Meetings\Processed" --model ".\models\ggml-small.en.bin" --language en
```

Or use the wrapper script, which defaults to `D:\Meetings\Processed`, `.\models\ggml-small.en.bin`, and transcript-only output:

```powershell
.\transcribe.ps1 "D:\Meetings\Incoming\2026-06-17-customer-call\meeting.mp4"
```

Supported inputs:

```text
.m4a, .mp3, .wav, .aac, .flac, .mp4, .mov, .mkv, .webm, .3gp
```

Directory scans are non-recursive by default. Add `--recursive` to scan subfolders.

## Outputs

For an input such as:

```text
D:\Meetings\Incoming\2026-06-17-customer-call\meeting.mp4
```

The app creates a unique folder like:

```text
D:\Meetings\Processed\meeting_20260617_143022_a1b2c3d4\
  manifest.json
  run.log
  audio.wav
  transcript.txt
  transcript.srt
  transcript.json
  meeting_notes.md       # minimal placeholder unless notes are explicitly enabled
```

`audio.wav` is kept by default for review and troubleshooting. Use `--clean-intermediate` to remove it after a successful transcription. The original recording is never deleted.

## Notes Providers

`none` is the default. It writes a minimal `meeting_notes.md` saying notes generation was disabled, leaving `transcript.txt` ready for review or manual export.

`heuristic` is local, deterministic, offline, and labels decisions, action items, questions, and dates as candidates where appropriate. It is intentionally rough.

`ollama` is optional and must be selected explicitly:

```powershell
.\meeting-transcriber.exe run `
  --input "D:\Meetings\Incoming\call.mp4" `
  --output "D:\Meetings\Processed" `
  --model ".\models\ggml-small.en.bin" `
  --notes-provider ollama `
  --ollama-model "llama3.2:latest"
```

Ollama guardrails:

- Default URL: `http://127.0.0.1:11434/api/generate`.
- Only loopback hosts such as `127.0.0.1`, `::1`, or `localhost` are accepted.
- Model names containing `cloud` are rejected unless `--allow-cloud-model` is set.
- No hosted model APIs are called by this app.

## Model Guidance

- `base.en`: faster, lower accuracy.
- `small.en`: a good default for English meetings.
- `medium.en`: better accuracy, slower.
- Quantized large or turbo models can be explored later.

All transcripts and notes require human review before sharing or using as an official record.

## Useful Flags

```text
--input <path>               Required for run. File or directory.
--output <dir>               Output root. Defaults to _transcribed near the input.
--model <path>               Required for run.
--language <code>            Default: en.
--threads <n>                Default: runtime.NumCPU()-1, minimum 1.
--recursive                  Scan subdirectories.
--dry-run                    Print planned work without processing files.
--notes-provider <provider>  none | heuristic | ollama. Default: none.
--clean-intermediate         Delete audio.wav after successful transcription.
--fail-fast                  Stop a batch after the first failed file.
--verbose-transcript-log     Allow transcript text in run.log.
```

Exit codes:

```text
0  success
1  processing failure
2  invalid user input or config
3  missing dependency
```

## Tests

```powershell
go test ./...
```

The test suite includes command argument tests, dependency discovery tests, media scanning tests, heuristic notes tests, manifest serialization tests, and a fake-tools integration test that exercises the full run path without real FFmpeg or Whisper.
