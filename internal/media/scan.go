package media

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var supportedExtensions = map[string]bool{
	".m4a":  true,
	".mp3":  true,
	".wav":  true,
	".aac":  true,
	".flac": true,
	".mp4":  true,
	".mov":  true,
	".mkv":  true,
	".webm": true,
	".3gp":  true,
}

var generatedNames = map[string]bool{
	"audio.wav":        true,
	"transcript.txt":   true,
	"transcript.srt":   true,
	"transcript.json":  true,
	"meeting_notes.md": true,
	"manifest.json":    true,
	"run.log":          true,
}

func IsSupported(path string) bool {
	return supportedExtensions[strings.ToLower(filepath.Ext(path))]
}

func SupportedExtensions() []string {
	exts := make([]string, 0, len(supportedExtensions))
	for ext := range supportedExtensions {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	return exts
}

func Scan(input string, recursive bool) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		if !IsSupported(input) {
			return nil, fmt.Errorf("unsupported input extension %q", filepath.Ext(input))
		}
		if appearsGenerated(input) {
			return nil, fmt.Errorf("input appears to be a generated output: %s", input)
		}
		return []string{input}, nil
	}

	var files []string
	if recursive {
		err = filepath.WalkDir(input, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if path != input && generatedDirName(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if IsSupported(path) && !appearsGenerated(path) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(input)
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(input, entry.Name())
			if IsSupported(path) && !appearsGenerated(path) {
				files = append(files, path)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

func appearsGenerated(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	if generatedNames[name] {
		return true
	}
	dir := strings.ToLower(filepath.Base(filepath.Dir(path)))
	return generatedDirName(dir)
}

func generatedDirName(name string) bool {
	return name == "_transcribed" || strings.HasSuffix(name, "_transcribed")
}

var safeNamePattern = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func UniqueOutputDir(root, source string, now time.Time) (string, error) {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	safe := safeNamePattern.ReplaceAllString(base, "_")
	safe = strings.Trim(safe, "._-")
	if safe == "" {
		safe = "recording"
	}
	if len(safe) > 80 {
		safe = safe[:80]
	}

	absSource, err := filepath.Abs(source)
	if err != nil {
		absSource = source
	}
	sum := sha256.Sum256([]byte(absSource))
	shortHash := hex.EncodeToString(sum[:])[:8]
	stamp := now.Format("20060102_150405")
	name := fmt.Sprintf("%s_%s_%s", safe, stamp, shortHash)
	candidate := filepath.Join(root, name)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate, nil
	}
	for i := 2; i < 1000; i++ {
		next := fmt.Sprintf("%s-%d", candidate, i)
		if _, err := os.Stat(next); os.IsNotExist(err) {
			return next, nil
		}
	}
	return "", fmt.Errorf("could not find a free output directory below %s", root)
}

func DefaultOutputDir(input string) (string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return filepath.Join(input, "_transcribed"), nil
	}
	return filepath.Join(filepath.Dir(input), "_transcribed"), nil
}
