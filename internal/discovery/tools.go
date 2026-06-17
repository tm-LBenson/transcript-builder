package discovery

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type ExplicitPaths struct {
	FFmpeg  string
	FFprobe string
	Whisper string
}

type ToolPaths struct {
	FFmpeg  string
	FFprobe string
	Whisper string
}

type Result struct {
	Tools  ToolPaths
	Errors []error
}

func Discover(explicit ExplicitPaths) Result {
	appDir := executableDir()
	cwd, _ := os.Getwd()

	var result Result
	result.Tools.FFmpeg = discoverOne("ffmpeg.exe", explicit.FFmpeg, "MEETING_TRANSCRIBER_FFMPEG", []string{
		filepath.Join(appDir, "tools", "ffmpeg", "bin", "ffmpeg.exe"),
		filepath.Join(cwd, "tools", "ffmpeg", "bin", "ffmpeg.exe"),
	})
	result.Tools.FFprobe = discoverOne("ffprobe.exe", explicit.FFprobe, "MEETING_TRANSCRIBER_FFPROBE", []string{
		filepath.Join(appDir, "tools", "ffmpeg", "bin", "ffprobe.exe"),
		filepath.Join(cwd, "tools", "ffmpeg", "bin", "ffprobe.exe"),
	})
	result.Tools.Whisper = discoverOne("whisper-cli.exe", explicit.Whisper, "MEETING_TRANSCRIBER_WHISPER", []string{
		filepath.Join(appDir, "tools", "whisper", "whisper-cli.exe"),
		filepath.Join(cwd, "tools", "whisper", "whisper-cli.exe"),
	})

	if result.Tools.FFmpeg == "" {
		result.Errors = append(result.Errors, MissingToolError{Name: "ffmpeg.exe", Flag: "--ffmpeg", Env: "MEETING_TRANSCRIBER_FFMPEG"})
	}
	if result.Tools.FFprobe == "" {
		result.Errors = append(result.Errors, MissingToolError{Name: "ffprobe.exe", Flag: "--ffprobe", Env: "MEETING_TRANSCRIBER_FFPROBE"})
	}
	if result.Tools.Whisper == "" {
		result.Errors = append(result.Errors, MissingToolError{Name: "whisper-cli.exe", Flag: "--whisper", Env: "MEETING_TRANSCRIBER_WHISPER"})
	}
	return result
}

func discoverOne(name, explicit, envName string, appLocal []string) string {
	if explicit != "" {
		if executableExists(explicit) {
			return explicit
		}
		return ""
	}
	if value := os.Getenv(envName); value != "" {
		if executableExists(value) {
			return value
		}
		return ""
	}
	for _, candidate := range dedupe(appLocal) {
		if executableExists(candidate) {
			return candidate
		}
	}
	if found, err := exec.LookPath(name); err == nil {
		return found
	}
	return ""
}

func executableExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return true
}

func executableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	return filepath.Dir(exe)
}

func dedupe(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

type MissingToolError struct {
	Name string
	Flag string
	Env  string
}

func (e MissingToolError) Error() string {
	return fmt.Sprintf("%s not found", e.Name)
}

func MissingToolFix(err error) string {
	var missing MissingToolError
	if !errors.As(err, &missing) {
		return ""
	}
	return fmt.Sprintf("Fix: install %s and add it to PATH, or pass %s \"C:\\path\\to\\%s\". You can also set %s.", missing.Name, missing.Flag, missing.Name, missing.Env)
}
