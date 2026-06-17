# Handoff: Windows Go Meeting Transcriber

## Build request for Codex

Build a Windows-first Go CLI app that automates local transcription of meeting recordings copied from a phone via USB. The app should scan an input directory, extract clean WAV audio from video/audio files, run local Whisper transcription through `whisper.cpp`, and produce transcript and meeting-notes files. The user will manually delete the original recording and generated intermediate files after reviewing the output.

The tool must be privacy-first: do not upload meeting audio, video, transcripts, or notes to any cloud service by default. Do not delete source recordings automatically.

---

## User workflow

1. User records a meeting on a phone.
2. User transfers the recording to a Windows machine via USB.
3. User places the file in a local folder, for example:
   - `D:\Meetings\Incoming\2026-06-17-customer-call\`
4. User runs:

```powershell
meeting-transcriber.exe run `
  --input "D:\Meetings\Incoming\2026-06-17-customer-call" `
  --output "D:\Meetings\Processed" `
  --model ".\models\ggml-small.en.bin" `
  --language en `
  --notes-provider heuristic
```

5. App outputs transcript and draft meeting notes.
6. User reviews the notes.
7. User manually cleans up original recording, WAV file, and any generated files they do not want to keep.

---

## Core design decision

Do not implement speech recognition directly in Go for the MVP. Go should orchestrate local command-line tools:

- `ffmpeg.exe` / `ffprobe.exe` for media inspection and audio extraction.
- `whisper-cli.exe` from `whisper.cpp` for local transcription.
- Optional local-only Ollama integration for higher-quality meeting notes, behind an explicit flag.

This keeps the Go code simple, auditable, and reliable on Windows.

---

## MVP requirements

### Commands

Implement these commands:

```powershell
meeting-transcriber.exe doctor
meeting-transcriber.exe run --input <dir-or-file> [flags]
meeting-transcriber.exe version
```

### `doctor`

Checks:

- App version/build info.
- Windows OS/architecture.
- `ffmpeg.exe` availability.
- `ffprobe.exe` availability.
- `whisper-cli.exe` availability.
- Whisper model file exists and is readable, if `--model` is provided.
- Output directory is writable.
- Optional: Ollama availability if `--notes-provider ollama` is requested.

The command should print actionable fixes, for example:

```text
ERROR: ffmpeg.exe not found.
Fix: install FFmpeg and add its bin folder to PATH, or pass --ffmpeg "C:\path\to\ffmpeg.exe".
```

### `run`

Accepts either a single media file or a directory.

Supported input extensions:

```text
.m4a, .mp3, .wav, .aac, .flac, .mp4, .mov, .mkv, .webm, .3gp
```

Default directory scan behavior:

- Non-recursive by default.
- Add `--recursive` for recursive scanning.
- Skip files that already appear to be generated outputs.
- Process files one at a time by default.
- Add `--workers N` later if desired; MVP can keep `N=1`.

For each input file:

1. Create a unique output folder.
2. Use `ffprobe` to capture media metadata where possible.
3. Use `ffmpeg` to extract 16 kHz mono 16-bit PCM WAV.
4. Run `whisper-cli.exe` on the WAV.
5. Produce transcript files.
6. Produce a draft meeting-notes Markdown file.
7. Write a local run log and manifest.

---

## Suggested output layout

For input:

```text
D:\Meetings\Incoming\2026-06-17-customer-call\meeting.mp4
```

Output should be:

```text
D:\Meetings\Processed\meeting_20260617_143022_a1b2c3d4\
  manifest.json
  run.log
  audio.wav
  transcript.txt
  transcript.srt
  transcript.json            # optional, if whisper-cli supports JSON output
  meeting_notes.md
```

Use a sanitized base filename plus timestamp and/or a short hash to avoid collisions.

Do not store temp files in a random system temp directory unless explicitly requested. Keep everything in the output folder so the user can manually clean up.

---

## CLI flags

### Global/common flags

```text
--ffmpeg <path>              Optional explicit path to ffmpeg.exe
--ffprobe <path>             Optional explicit path to ffprobe.exe
--whisper <path>             Optional explicit path to whisper-cli.exe
--model <path>               Required for run, unless set in config
--config <path>              Optional TOML/JSON config path; not required for MVP
--verbose                    More detailed logging
--dry-run                    Show what would happen without processing files
```

### `run` flags

```text
--input <path>               Required. File or directory.
--output <dir>               Optional. Default: <input>\_transcribed if input is a dir, otherwise beside file.
--recursive                  Scan subdirectories.
--language <code>            Default: en
--threads <n>                Optional whisper thread count. Default: runtime.NumCPU()-1, minimum 1.
--keep-wav                   Default true for MVP. Keep extracted WAV for review/debugging.
--clean-intermediate         Optional. Delete WAV after successful transcription. Default false.
--notes-provider <provider>  none | heuristic | ollama. Default heuristic.
--ollama-url <url>           Default: http://127.0.0.1:11434/api/generate
--ollama-model <name>        Example: llama3.2:latest or qwen2.5:7b-instruct
--allow-cloud-model          Default false. Required if model name contains "cloud".
--force                      Overwrite existing output files.
```

---

## Dependency discovery

Search order for executables:

1. Explicit CLI flag.
2. Environment variables:
   - `MEETING_TRANSCRIBER_FFMPEG`
   - `MEETING_TRANSCRIBER_FFPROBE`
   - `MEETING_TRANSCRIBER_WHISPER`
3. App-local tools folder:
   - `.\tools\ffmpeg\bin\ffmpeg.exe`
   - `.\tools\ffmpeg\bin\ffprobe.exe`
   - `.\tools\whisper\whisper-cli.exe`
4. PATH lookup.

Do not silently download dependencies in the MVP. The user or Codex can install/place them manually. A later `init` command may be added, but it is not required.

---

## FFmpeg invocation

Use `exec.CommandContext` with argument slices. Do not shell-concatenate commands.

Audio extraction command:

```text
ffmpeg.exe -nostdin -y -hide_banner -loglevel error -i <input> -vn -ac 1 -ar 16000 -c:a pcm_s16le <output.wav>
```

Notes:

- `-vn` drops video.
- `-ac 1` converts to mono.
- `-ar 16000` converts to 16 kHz sample rate.
- `-c:a pcm_s16le` produces 16-bit PCM WAV.
- Use `-y` only when `--force` is set or when writing into a newly created unique output directory. Safer default: unique output folder avoids overwriting.

Optional ffprobe command:

```text
ffprobe.exe -v error -show_format -show_streams -print_format json <input>
```

Store ffprobe output in `manifest.json` or a separate `media_info.json` if useful.

---

## Whisper invocation

Preferred command shape:

```text
whisper-cli.exe -m <model.bin> -f <audio.wav> -l <language> -t <threads> -pp -pc -otxt -osrt -ojf -of <outputBase>
```

Where:

```text
<outputBase> = D:\Meetings\Processed\...\transcript
```

Expected generated files may include:

```text
transcript.txt
transcript.srt
transcript.json
```

Implementation details:

- Run `whisper-cli.exe -h` in `doctor` and log the first few lines to help diagnose version changes.
- If `-ojf` or another output flag is unsupported in the installed whisper.cpp version, fall back to text and SRT only.
- Capture stdout and stderr into `run.log`.
- If `transcript.txt` is not generated but stdout contains transcript text, write stdout to `transcript.txt`.
- Preserve original Unicode text as UTF-8.

---

## Meeting notes generation

### Default provider: `heuristic`

This should be local, deterministic, and offline. It will not be as good as an LLM, but it creates a useful draft.

`meeting_notes.md` should include:

```markdown
# Meeting Notes

## Source
- Recording: <original file name>
- Processed: <timestamp>
- Duration: <duration if available>
- Transcript: transcript.txt
- Generated by: meeting-transcriber <version>

## Review status
Draft generated from machine transcription. Human review required.

## Executive summary
<short heuristic summary or placeholder if insufficient confidence>

## Topics discussed
- <topic bullets inferred from repeated terms/headings>

## Decisions
- <candidate decisions, each marked "candidate" if heuristic>

## Action items
- [ ] <candidate action item> — Owner: Unknown — Due: Unknown

## Open questions
- <candidate question>

## Dates, deadlines, and follow-ups mentioned
- <candidate date/follow-up>

## Full transcript
See `transcript.txt`.
```

Heuristic extraction ideas:

- Split transcript into paragraphs/segments.
- Candidate action-item phrases:
  - "I will"
  - "I'll"
  - "we will"
  - "we need to"
  - "follow up"
  - "action item"
  - "can you"
  - "please"
  - "by Friday", "by Monday", etc.
- Candidate decisions:
  - "we decided"
  - "decision"
  - "we agreed"
  - "the plan is"
  - "we're going with"
- Candidate open questions:
  - lines ending with `?`
  - "question is"
  - "need to find out"
  - "not sure"
- Topic extraction:
  - remove stop words
  - count repeated noun-ish terms
  - group nearby segments by repeated keywords

Be transparent: label heuristic items as candidates when confidence is not high.

### Optional provider: `ollama`

Implement this only if time allows. It must be explicitly selected:

```powershell
meeting-transcriber.exe run --input "D:\Meetings\Incoming\call" --notes-provider ollama --ollama-model "llama3.2:latest"
```

Ollama API request:

```http
POST http://127.0.0.1:11434/api/generate
Content-Type: application/json
```

Request body:

```json
{
  "model": "llama3.2:latest",
  "prompt": "<prompt>",
  "stream": false
}
```

Parse `response` from the returned JSON.

Privacy guardrails:

- Only allow `127.0.0.1`, `localhost`, or explicitly provided loopback URL by default.
- Reject model names containing `cloud` unless `--allow-cloud-model` is set.
- Print a clear warning before using anything other than loopback.
- Never use OpenAI, Azure, Google, Anthropic, or any other hosted API in this app unless a future user explicitly adds that feature.

Recommended Ollama prompt:

```text
You are creating private meeting notes from an automatic transcript.
Do not invent details. If a detail is unclear, write "Unclear".
Return concise Markdown with these sections:

# Meeting Notes
## Executive Summary
## Key Topics
## Decisions
## Action Items
Use checkboxes. Include owner and due date only when stated.
## Open Questions
## Risks / Blockers
## Follow-ups

Transcript:
<<<
{transcript}
>>>
```

For long transcripts, chunk by approximate character count, summarize chunks, then run a final merge prompt. Preserve a link/reference to the full transcript.

---

## Go implementation notes

Use the standard library unless a small dependency clearly improves the CLI.

Recommended package layout:

```text
meeting-transcriber/
  go.mod
  README.md
  cmd/meeting-transcriber/main.go
  internal/app/run.go
  internal/config/config.go
  internal/discovery/tools.go
  internal/media/scan.go
  internal/media/ffprobe.go
  internal/audio/ffmpeg.go
  internal/transcribe/whisper.go
  internal/notes/heuristic.go
  internal/notes/ollama.go
  internal/manifest/manifest.go
  internal/logging/logging.go
```

Important coding practices:

- Use `context.Context` and timeouts for external commands.
- Use `exec.CommandContext`; never construct shell commands with unsanitized paths.
- Support spaces and Unicode in Windows file paths.
- Normalize output to UTF-8.
- Fail one file without crashing the whole batch unless `--fail-fast` is set.
- Keep run logs local.
- Do not log full transcript text unless `--verbose-transcript-log` is explicitly set.
- Include clear exit codes:
  - `0`: success
  - `1`: processing failure
  - `2`: invalid user input/config
  - `3`: missing dependency

---

## Manifest format

Write `manifest.json` per processed input.

Example:

```json
{
  "app": "meeting-transcriber",
  "version": "0.1.0",
  "processed_at": "2026-06-17T14:30:22-04:00",
  "source_file": "D:\\Meetings\\Incoming\\2026-06-17-customer-call\\meeting.mp4",
  "output_dir": "D:\\Meetings\\Processed\\meeting_20260617_143022_a1b2c3d4",
  "audio_wav": "audio.wav",
  "transcript_txt": "transcript.txt",
  "transcript_srt": "transcript.srt",
  "notes_md": "meeting_notes.md",
  "ffmpeg_version": "<captured>",
  "whisper_version_or_help": "<captured>",
  "model": ".\\models\\ggml-small.en.bin",
  "language": "en",
  "notes_provider": "heuristic",
  "status": "success",
  "errors": []
}
```

---

## Acceptance criteria

1. `go build ./cmd/meeting-transcriber` succeeds on Windows.
2. `meeting-transcriber.exe doctor` clearly reports missing dependencies and model issues.
3. Given an `.mp4` copied from a phone, `run` creates a WAV, transcript, notes, manifest, and log.
4. Source recording is never deleted by default.
5. No network calls occur unless `--notes-provider ollama` is explicitly selected.
6. Ollama provider defaults to loopback only.
7. Paths with spaces work.
8. Batch directory processing continues after one bad file and reports failures at the end.
9. Output files are human-readable and easy to manually clean up.

---

## Test plan

Unit tests:

- Media extension filtering.
- Output path sanitization and collision avoidance.
- Dependency discovery search order.
- Command argument construction for FFmpeg and Whisper.
- Heuristic notes extraction.
- Manifest serialization.

Integration tests:

- Use fake executables or small batch files to validate command invocation without needing real FFmpeg/Whisper.
- Optional real integration test behind an environment variable, for example:

```powershell
$env:MTX_RUN_REAL_INTEGRATION="1"
go test ./... -run Integration
```

Manual smoke test:

```powershell
go build -o .\meeting-transcriber.exe .\cmd\meeting-transcriber
.\meeting-transcriber.exe doctor --model .\models\ggml-small.en.bin
.\meeting-transcriber.exe run --input "D:\Meetings\Incoming\test" --output "D:\Meetings\Processed" --model .\models\ggml-small.en.bin --language en
```

---

## README content Codex should include

The README should explain:

- This is a local Windows transcription helper.
- It requires FFmpeg and whisper.cpp.
- It does not delete recordings.
- It does not upload files by default.
- Basic setup steps.
- Example commands.
- Model recommendations:
  - `base.en`: faster, lower accuracy.
  - `small.en`: good default for English meetings.
  - `medium.en`: better accuracy, slower.
  - quantized large/turbo models can be explored later.
- Transcript and notes require human review.
- Cleanup remains manual.

---

## Nice-to-have later, not MVP

- Windows GUI.
- Drag-and-drop folder support.
- Speaker diarization.
- Direct phone import watcher.
- Secure-delete helper, with clear warning that secure deletion on SSDs/cloud-synced folders is not guaranteed.
- Config file wizard.
- Optional packaging script that bundles the Go EXE with local tool paths.
