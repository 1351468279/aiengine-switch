package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateModelsUsesBearerTokenAndChecksModels(t *testing.T) {
	const token = "test-secret-token"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"claude-sonnet-5"},{"id":"claude-opus-5"},{"id":"claude-haiku-4-5-20251001"},{"id":"gpt-5.6-sol"}]}`))
	}))
	defer server.Close()
	previous := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previous })
	if _, err := validateModels(token, []string{"claude", "codex"}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateModelsNeverIncludesTokenInError(t *testing.T) {
	const token = "must-not-leak"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()
	previous := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previous })
	_, err := validateModels(token, []string{"codex"})
	if err == nil || strings.Contains(err.Error(), token) {
		t.Fatalf("expected redacted authorization error, got %v", err)
	}
}

func TestValidateModelsOnlyChecksSelectedTool(t *testing.T) {
	const token = "codex-only-token"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"gpt-5.6-sol"}]}`))
	}))
	defer server.Close()
	previous := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previous })

	if _, err := validateModels(token, []string{"codex"}); err != nil {
		t.Fatalf("Codex-only key was rejected: %v", err)
	}
	if _, err := validateModels(token, []string{"claude"}); err == nil {
		t.Fatal("Claude validation unexpectedly accepted a Codex-only key")
	}
}

func TestValidateModelsTreatsClaudeDesktopAsClaude(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"claude-sonnet-5"},{"id":"claude-opus-5"},{"id":"claude-haiku-4-5-20251001"}]}`))
	}))
	defer server.Close()
	previous := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previous })
	if _, err := validateModels("desktop-token", []string{desktopTool}); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDesktopMessagesRequiresStreamingAnthropicResponse(t *testing.T) {
	const token = "desktop-stream-token"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer "+token || request.Header.Get("Anthropic-Version") == "" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || body["model"] != ClaudeModel || body["stream"] != true {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = writer.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\"}\n\n"))
	}))
	defer server.Close()
	previous := messagesEndpoint
	messagesEndpoint = server.URL
	t.Cleanup(func() { messagesEndpoint = previous })
	if err := validateDesktopMessages(token); err != nil {
		t.Fatal(err)
	}
}
