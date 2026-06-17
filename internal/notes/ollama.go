package notes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type OllamaOptions struct {
	URL             string
	Model           string
	AllowCloudModel bool
}

type OllamaClient struct {
	HTTPClient *http.Client
}

func (c OllamaClient) Generate(ctx context.Context, opts OllamaOptions, transcript string) (string, error) {
	if err := ValidateOllamaOptions(opts); err != nil {
		return "", err
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 10 * time.Minute}
	}
	chunks := chunkText(transcript, 24000)
	if len(chunks) == 0 {
		return "", errors.New("transcript is empty")
	}
	if len(chunks) == 1 {
		return c.generate(ctx, opts.URL, opts.Model, finalPrompt(chunks[0]))
	}

	summaries := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		prompt := fmt.Sprintf("Summarize transcript chunk %d of %d for private meeting notes. Do not invent details. Return concise bullets for topics, decisions, action items, questions, and risks.\n\nTranscript chunk:\n<<<\n%s\n>>>", i+1, len(chunks), chunk)
		summary, err := c.generate(ctx, opts.URL, opts.Model, prompt)
		if err != nil {
			return "", err
		}
		summaries = append(summaries, summary)
	}
	return c.generate(ctx, opts.URL, opts.Model, finalPrompt(strings.Join(summaries, "\n\n")))
}

func ValidateOllamaOptions(opts OllamaOptions) error {
	if strings.TrimSpace(opts.Model) == "" {
		return errors.New("ollama model is required")
	}
	if strings.Contains(strings.ToLower(opts.Model), "cloud") && !opts.AllowCloudModel {
		return errors.New("refusing Ollama model name containing \"cloud\" without --allow-cloud-model")
	}
	parsed, err := url.Parse(opts.URL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported Ollama URL scheme %q", parsed.Scheme)
	}
	host := parsed.Hostname()
	if !isLoopbackHost(host) {
		return fmt.Errorf("refusing non-loopback Ollama URL %q; use 127.0.0.1 or localhost", opts.URL)
	}
	return nil
}

func (c OllamaClient) generate(ctx context.Context, endpoint, model, prompt string) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024))
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var decoded struct {
		Response string `json:"response"`
		Error    string `json:"error"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return "", err
	}
	if decoded.Error != "" {
		return "", errors.New(decoded.Error)
	}
	if strings.TrimSpace(decoded.Response) == "" {
		return "", errors.New("ollama returned an empty response")
	}
	return strings.TrimSpace(decoded.Response) + "\n", nil
}

func finalPrompt(transcript string) string {
	return `You are creating private meeting notes from an automatic transcript.
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
` + transcript + `
>>>`
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func chunkText(text string, maxChars int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if maxChars <= 0 || len(text) <= maxChars {
		return []string{text}
	}
	var chunks []string
	for len(text) > maxChars {
		cut := strings.LastIndexAny(text[:maxChars], "\n.?!")
		if cut < maxChars/2 {
			cut = maxChars
		}
		chunks = append(chunks, strings.TrimSpace(text[:cut]))
		text = strings.TrimSpace(text[cut:])
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}
