package app

import (
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
