package notes

import (
	"strings"
	"testing"
	"time"
)

func TestHeuristicIncludesCandidateSections(t *testing.T) {
	transcript := strings.Join([]string{
		"We discussed budget and budget timing for the launch.",
		"We decided the plan is to keep the current launch scope.",
		"I will follow up by Friday with the budget owner.",
		"Can we confirm the final launch date?",
		"We are not sure who approves the final launch date.",
	}, "\n")
	got := Heuristic(transcript, MeetingMeta{
		SourceFile:     `D:\Meetings\call.mp4`,
		ProcessedAt:    time.Date(2026, 6, 17, 14, 30, 22, 0, time.UTC),
		Duration:       "15:00",
		TranscriptFile: "transcript.txt",
		Version:        "0.1.0",
	})
	for _, want := range []string{
		"# Meeting Notes",
		"budget",
		"Candidate: We decided",
		"Candidate: I will follow up",
		"Candidate: Can we confirm",
		"Owner: Unknown",
		"Human review required",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("notes missing %q:\n%s", want, got)
		}
	}
}
