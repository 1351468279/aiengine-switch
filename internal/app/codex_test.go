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
	for _, wanted := range []string{"# keep this comment", "[model_providers.other]", "[model_providers.aiengine.auth]", `args = ["credential", "print", "codex"]`, CodexModel} {
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
	if strings.Contains(text, "model_providers.aiengine") || strings.Contains(text, "model_reasoning_effort") {
		t.Fatalf("managed settings remain after uninstall:\n%s", text)
	}
}

func TestCodexRestoresExistingAiEngineProvider(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	originalBlock := `[model_providers.aiengine]
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
	modified := strings.Replace(string(installed), "AiEngine NewAPI", "User changed", 1)
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
	if len(conflicts) != 1 || conflicts[0] != "model_providers.aiengine" {
		t.Fatalf("expected provider conflict, got %v", conflicts)
	}
}

func TestCodexMigratesLegacyManagedProvider(t *testing.T) {
	paths := testPaths(t)
	installed, _, legacyState, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	legacyConfig := strings.ReplaceAll(string(installed), codexProviderID, legacyProviderID)
	legacyConfig = strings.ReplaceAll(legacyConfig, "AiEngine NewAPI", "AIARE NewAPI")
	legacyState.InstalledBlock = strings.ReplaceAll(legacyState.InstalledBlock, codexProviderID, legacyProviderID)
	legacyState.InstalledBlock = strings.ReplaceAll(legacyState.InstalledBlock, "AiEngine NewAPI", "AIARE NewAPI")
	legacyState.ProviderID = ""
	legacyProvider, err := storedValue(legacyProviderID, true)
	if err != nil {
		t.Fatal(err)
	}
	providerField := legacyState.Fields["model_provider"]
	providerField.Installed = legacyProvider
	legacyState.Fields["model_provider"] = providerField
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(legacyConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	migrated, _, migratedState, err := prepareCodexInstall(paths, legacyState)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(migrated), "model_providers."+legacyProviderID) || !strings.Contains(string(migrated), "model_providers."+codexProviderID) {
		t.Fatalf("legacy provider was not migrated:\n%s", migrated)
	}
	if err := os.WriteFile(paths.CodexConfig, migrated, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, conflicts, remove, err := prepareCodexUninstall(migratedState, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 || !remove {
		t.Fatalf("migrated provider did not uninstall cleanly: conflicts=%v remove=%v", conflicts, remove)
	}
}

func TestCodexReusesExistingOpenAIProviderID(t *testing.T) {
	paths := testPaths(t)
	original := `# keep the user's Codex settings
model_provider = "OpenAI"
model = "official-model"
`
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	installedText := string(installed)
	if state.ProviderID != "OpenAI" {
		t.Fatalf("provider ID = %q, want OpenAI", state.ProviderID)
	}
	for _, wanted := range []string{`model_provider = "OpenAI"`, "[model_providers.OpenAI]", "[model_providers.OpenAI.auth]"} {
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
	if string(restored) != original {
		t.Fatalf("Codex config was not restored exactly:\n%s", restored)
	}
}

func TestCodexRestoresExistingCustomProviderID(t *testing.T) {
	paths := testPaths(t)
	original := `model_provider = "customer_proxy"

[model_providers.customer_proxy]
name = "Customer provider"
base_url = "https://customer.example/v1"
wire_api = "responses"
`
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state.ProviderID != "customer_proxy" {
		t.Fatalf("provider ID = %q, want customer_proxy", state.ProviderID)
	}
	if !strings.Contains(string(installed), `base_url = "`+RelayV1URL+`"`) {
		t.Fatalf("customer provider was not configured for AiEngine:\n%s", installed)
	}
	if err := os.WriteFile(paths.CodexConfig, installed, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, _, conflicts, _, err := prepareCodexUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 || string(restored) != original {
		t.Fatalf("custom provider was not restored: conflicts=%v\n%s", conflicts, restored)
	}
}

func TestCodexReusesProviderFoundInSessionHistory(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(paths.CodexSessions, 0o700); err != nil {
		t.Fatal(err)
	}
	history := `{"type":"session_meta","payload":{"model_provider":"OpenAI"}}
not-json
{"type":"response_item","payload":{"type":"message"}}
`
	if err := os.WriteFile(filepath.Join(paths.CodexSessions, "2026-08-08.jsonl"), []byte(history), 0o600); err != nil {
		t.Fatal(err)
	}

	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state.ProviderID != "OpenAI" || !strings.Contains(string(installed), `model_provider = "OpenAI"`) {
		t.Fatalf("session provider was not reused: provider=%q\n%s", state.ProviderID, installed)
	}
}

func TestCodexMigrationUsesSessionProviderAfterOldAiEngineInstall(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.CodexSessions, 0o700); err != nil {
		t.Fatal(err)
	}
	original := `model_provider = "aiengine"
model = "official-model"

[model_providers.aiengine]
name = "AiEngine NewAPI"
base_url = "https://old.example/v1"
wire_api = "responses"
`
	if err := os.WriteFile(paths.CodexConfig, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.CodexSessions, "old.jsonl"), []byte(`{"type":"session_meta","payload":{"model_provider":"OpenAI"}}
`), 0o600); err != nil {
		t.Fatal(err)
	}
	previous := &ToolState{
		ConfigPath:      paths.CodexConfig,
		ConfigExisted:   true,
		Fields:          make(map[string]FieldState),
		InstalledBlock:  "[model_providers.aiengine]\nname = \"AiEngine NewAPI\"\nbase_url = \"https://old.example/v1\"\nwire_api = \"responses\"\n",
		OriginalBlockOK: false,
	}

	installed, _, state, err := prepareCodexInstall(paths, previous)
	if err != nil {
		t.Fatal(err)
	}
	if state.ProviderID != "OpenAI" || !strings.Contains(string(installed), "[model_providers.OpenAI]") || strings.Contains(string(installed), "https://old.example/v1") {
		t.Fatalf("old AiEngine install did not migrate to session provider:\n%s", installed)
	}
}

func TestCodexDoesNotOverrideBuiltinOpenAIProvider(t *testing.T) {
	paths := testPaths(t)
	config := `model_provider = "openai"
model = "official-model"
`
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	installed, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state.ProviderID != codexProviderID || !strings.Contains(string(installed), `model_provider = "`+codexProviderID+`"`) {
		t.Fatalf("builtin provider was overridden: provider=%q\n%s", state.ProviderID, installed)
	}
}

func TestCodexRejectsUnsafeProviderID(t *testing.T) {
	paths := testPaths(t)
	config := `model_provider = "customer.proxy"
`
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, state, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if state.ProviderID != codexProviderID {
		t.Fatalf("unsafe provider ID was reused: %q", state.ProviderID)
	}
}
