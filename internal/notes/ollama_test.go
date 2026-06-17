package notes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateOllamaOptionsRejectsCloudModelByDefault(t *testing.T) {
	err := ValidateOllamaOptions(OllamaOptions{
		URL:   "http://127.0.0.1:11434/api/generate",
		Model: "some-cloud-model",
	})
	if err == nil {
		t.Fatal("expected cloud model rejection")
	}
}

func TestValidateOllamaOptionsRejectsNonLoopback(t *testing.T) {
	err := ValidateOllamaOptions(OllamaOptions{
		URL:   "http://example.com/api/generate",
		Model: "llama3.2:latest",
	})
	if err == nil {
		t.Fatal("expected non-loopback URL rejection")
	}
}

func TestOllamaGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["stream"] != false {
			t.Fatalf("stream should be false, got %#v", payload["stream"])
		}
		if !strings.Contains(payload["prompt"].(string), "Transcript") {
			t.Fatalf("prompt did not include transcript wrapper: %s", payload["prompt"])
		}
		_, _ = w.Write([]byte(`{"response":"# Meeting Notes\n\n## Executive Summary\nDone."}`))
	}))
	defer server.Close()

	got, err := OllamaClient{HTTPClient: server.Client()}.Generate(t.Context(), OllamaOptions{
		URL:   server.URL,
		Model: "llama3.2:latest",
	}, "hello transcript")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "# Meeting Notes") {
		t.Fatalf("unexpected response: %s", got)
	}
}
