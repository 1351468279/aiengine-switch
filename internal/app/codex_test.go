package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCodexInstallAndUninstallPreserveCommentsAndProviders(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	original := `# keep this comment
model = "official-model"
approval_policy = "on-request"

[model_providers.other]
name = "Other"
base_url = "https://example.com/v1"
wire_api = "responses"
`
	if err := os.WriteFile(paths.CodexConfig, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	installedText := string(installed)
	for _, wanted := range []string{"# keep this comment", "[model_providers.other]", "[model_providers.aiare.auth]", `args = ["credential", "print", "codex"]`, CodexModel} {
		if !strings.Contains(installedText, wanted) {
			t.Fatalf("generated config does not contain %q:\n%s", wanted, installedText)
		}
	}
	if err := os.WriteFile(paths.CodexConfig, installed, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, _, conflicts, remove, err := prepareCodexUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if remove || len(conflicts) != 0 {
		t.Fatalf("unexpected uninstall result: remove=%v conflicts=%v", remove, conflicts)
	}
	text := string(restored)
	for _, wanted := range []string{"# keep this comment", `model = "official-model"`, "[model_providers.other]"} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("restored config does not contain %q:\n%s", wanted, text)
		}
	}
	if strings.Contains(text, "model_providers.aiare") || strings.Contains(text, "model_reasoning_effort") {
		t.Fatalf("managed settings remain after uninstall:\n%s", text)
	}
}

func TestCodexRestoresExistingAIAREProvider(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	originalBlock := `[model_providers.aiare]
name = "User provider"
base_url = "https://user.example/v1"
wire_api = "responses"
`
	if err := os.WriteFile(paths.CodexConfig, []byte(originalBlock), 0o600); err != nil {
		t.Fatal(err)
	}
	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, installed, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, _, conflicts, _, err := prepareCodexUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 || !strings.Contains(string(restored), "https://user.example/v1") {
		t.Fatalf("existing provider was not restored: conflicts=%v\n%s", conflicts, restored)
	}
}

func TestCodexUninstallDetectsProviderConflict(t *testing.T) {
	paths := testPaths(t)
	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	modified := strings.Replace(string(installed), "AIARE NewAPI", "User changed", 1)
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(modified), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, conflicts, _, err := prepareCodexUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0] != "model_providers.aiare" {
		t.Fatalf("expected provider conflict, got %v", conflicts)
	}
}
