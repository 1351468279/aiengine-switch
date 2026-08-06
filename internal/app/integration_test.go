//go:build !windows

package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallRerunDoctorAndUninstall(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeTool(t, binDir, "claude", "Claude Code test")
	writeFakeTool(t, binDir, "codex", "codex-cli test")

	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", filepath.Join(home, ".aiengine-setup"))
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, ".claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, name := range claudeConflictVariables {
		t.Setenv(name, "")
	}

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.ClaudeSettings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	originalClaude := `{"model":"customer-model","theme":"dark","env":{"KEEP_ME":"yes"}}`
	originalCodex := "# customer comment\nmodel = \"official-model\"\napproval_policy = \"on-request\"\n"
	originalAuth := `{"tokens":{"access_token":"official-login"}}`
	if err := os.WriteFile(paths.ClaudeSettings, []byte(originalClaude), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, []byte(originalCodex), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexAuth, []byte(originalAuth), 0o600); err != nil {
		t.Fatal(err)
	}

	codexOptions := commonOptions{tools: "codex", yes: true, tokenStdin: true, skipAPICheck: true}
	if err := withTestStdin(t, "codex-test-token\n", func() error {
		return runInstall(codexOptions, "test-codex")
	}); err != nil {
		t.Fatal(err)
	}
	assertCredential(t, paths.CodexCredential, "codex-test-token")
	assertMissing(t, paths.ClaudeCredential)
	assertSecretsNotLeaked(t, paths, "codex-test-token")

	claudeOptions := commonOptions{tools: "claude", yes: true, tokenStdin: true, skipAPICheck: true}
	if err := withTestStdin(t, "claude-test-token\n", func() error {
		return runInstall(claudeOptions, "test-claude")
	}); err != nil {
		t.Fatal(err)
	}
	assertCredential(t, paths.CodexCredential, "codex-test-token")
	assertCredential(t, paths.ClaudeCredential, "claude-test-token")
	assertSecretsNotLeaked(t, paths, "codex-test-token", "claude-test-token")

	if err := withTestStdin(t, "rotated-codex-token\n", func() error {
		return runInstall(codexOptions, "test-rerun")
	}); err != nil {
		t.Fatal(err)
	}
	assertCredential(t, paths.CodexCredential, "rotated-codex-token")
	assertCredential(t, paths.ClaudeCredential, "claude-test-token")
	assertSecretsNotLeaked(t, paths, "rotated-codex-token", "claude-test-token")
	state, err := loadState(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	originalModel, exists, err := state.Tools["claude"].Fields["model"].Original.decoded()
	if err != nil || !exists || originalModel != "customer-model" {
		t.Fatalf("rerun replaced the original Claude value: value=%v exists=%v err=%v", originalModel, exists, err)
	}

	if err := runDoctor(commonOptions{skipAPICheck: true}, "test-rerun"); err != nil {
		t.Fatal(err)
	}
	if err := runUninstall(commonOptions{tools: "codex"}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, paths.CodexCredential)
	assertCredential(t, paths.ClaudeCredential, "claude-test-token")
	state, err = loadState(paths.State)
	if err != nil || state.Tools["codex"] != nil || state.Tools["claude"] == nil {
		t.Fatalf("partial uninstall produced invalid state: state=%#v err=%v", state, err)
	}
	codexData, err := os.ReadFile(paths.CodexConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(codexData), "# customer comment") || !strings.Contains(string(codexData), `model = "official-model"`) {
		t.Fatalf("Codex settings were not restored:\n%s", codexData)
	}

	if err := runUninstall(commonOptions{tools: "all"}); err != nil {
		t.Fatal(err)
	}

	claudeData, err := os.ReadFile(paths.ClaudeSettings)
	if err != nil {
		t.Fatal(err)
	}
	var claude map[string]any
	if err := json.Unmarshal(claudeData, &claude); err != nil {
		t.Fatal(err)
	}
	if claude["model"] != "customer-model" || claude["theme"] != "dark" {
		t.Fatalf("Claude settings were not restored: %#v", claude)
	}
	if _, exists := claude["apiKeyHelper"]; exists {
		t.Fatal("Claude apiKeyHelper remained after uninstall")
	}
	authData, err := os.ReadFile(paths.CodexAuth)
	if err != nil || string(authData) != originalAuth {
		t.Fatalf("Codex auth.json changed: data=%s err=%v", authData, err)
	}
	for _, removed := range []string{paths.Binary, paths.Credential, paths.ClaudeCredential, paths.CodexCredential, paths.State} {
		assertMissing(t, removed)
	}
	entries, err := os.ReadDir(paths.BackupDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("initial backups were not retained: entries=%d err=%v", len(entries), err)
	}
}

func TestSchemaOneSharedCredentialMigratesOneToolAtATime(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.ClaudeSettings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.CodexConfig), 0o700); err != nil {
		t.Fatal(err)
	}
	claudeData, _, claudeState, err := prepareClaudeInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	codexData, _, codexState, err := prepareCodexInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ClaudeSettings, claudeData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.CodexConfig, codexData, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(paths.Credential), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Credential, []byte("legacy-shared-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacyState := &State{
		SchemaVersion:    1,
		InstallerVersion: "1.0.2",
		BinaryPath:       paths.Binary,
		CredentialPath:   paths.Credential,
		Tools: map[string]*ToolState{
			"claude": claudeState,
			"codex":  codexState,
		},
	}
	if err := writeJSON(paths.State, legacyState, 0o600); err != nil {
		t.Fatal(err)
	}

	loaded, err := loadState(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tools["claude"].CredentialPath != paths.Credential || loaded.Tools["codex"].CredentialPath != paths.Credential {
		t.Fatalf("schema 1 credential was not mapped to both tools: %#v", loaded)
	}
	pending, _, err := prepareInstallFiles(paths, loaded, "codex", "legacy-shared-token", CodexModel, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := commitFiles(pending); err != nil {
		t.Fatal(err)
	}
	assertCredential(t, paths.CodexCredential, "legacy-shared-token")
	assertCredential(t, paths.Credential, "legacy-shared-token")
	loaded, err = loadState(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SchemaVersion != stateSchema || loaded.Tools["codex"].CredentialPath != paths.CodexCredential || loaded.Tools["claude"].CredentialPath != paths.Credential {
		t.Fatalf("first migration produced invalid state: %#v", loaded)
	}

	pending, _, err = prepareInstallFiles(paths, loaded, "claude", "legacy-shared-token", ClaudeModel, "1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if err := commitFiles(pending); err != nil {
		t.Fatal(err)
	}
	assertCredential(t, paths.ClaudeCredential, "legacy-shared-token")
	assertMissing(t, paths.Credential)
	loaded, err = loadState(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CredentialPath != "" || loaded.Tools["claude"].CredentialPath != paths.ClaudeCredential {
		t.Fatalf("legacy credential was not fully retired: %#v", loaded)
	}
}

func TestInstallValidationFailureDoesNotWriteFiles(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeTool(t, binDir, "codex", "codex-cli test")
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", filepath.Join(home, ".aiengine-setup"))
	t.Setenv("CODEX_HOME", filepath.Join(home, ".codex"))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"claude-sonnet-5"}]}`))
	}))
	defer server.Close()
	previousEndpoint := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previousEndpoint })

	err := withTestStdin(t, "codex-invalid-token\n", func() error {
		return runInstall(commonOptions{tools: "codex", yes: true, tokenStdin: true}, "test")
	})
	if err == nil || !strings.Contains(err.Error(), CodexModel) {
		t.Fatalf("expected missing Codex model error, got %v", err)
	}
	paths, resolveErr := ResolvePaths()
	if resolveErr != nil {
		t.Fatal(resolveErr)
	}
	assertMissing(t, paths.BaseDir)
	assertMissing(t, paths.CodexConfig)
}

func TestGenericClientsInstallDoctorAndUninstall(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tool := range []string{"hermes", "opencode", "aider"} {
		writeFakeTool(t, binDir, tool, tool+" test")
	}
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", filepath.Join(home, ".aiengine-setup"))
	t.Setenv("HERMES_HOME", filepath.Join(home, ".hermes"))
	t.Setenv("OPENCODE_CONFIG", filepath.Join(home, ".config", "opencode", "opencode.json"))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{paths.HermesConfig, paths.HermesEnv, paths.OpenCodeConfig, paths.AiderConfig} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	originalHermes := "model:\n  default: customer-hermes\nui:\n  theme: dark\n"
	originalHermesEnv := "# customer env\nKEEP_ME=yes\nOPENAI_API_KEY='original-hermes-secret'\n"
	originalOpenCode := `{"theme":"customer","provider":{"aiengine":{"name":"Customer provider"}}}`
	originalAider := "dark-mode: true\n"
	for path, data := range map[string]string{
		paths.HermesConfig:   originalHermes,
		paths.HermesEnv:      originalHermesEnv,
		paths.OpenCodeConfig: originalOpenCode,
		paths.AiderConfig:    originalAider,
	} {
		if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	installCases := []struct {
		tool  string
		model string
		token string
	}{
		{tool: "hermes", model: "gpt-5.6-sol", token: "hermes-new-secret"},
		{tool: "opencode", model: "claude-sonnet-5", token: "opencode-new-secret"},
		{tool: "aider", model: "gpt-5.6-sol", token: "aider-new-secret"},
	}
	for _, test := range installCases {
		options := commonOptions{tools: test.tool, model: test.model, yes: true, tokenStdin: true, skipAPICheck: true}
		if err := withTestStdin(t, test.token+"\n", func() error {
			return runInstall(options, "generic-test")
		}); err != nil {
			t.Fatalf("install %s: %v", test.tool, err)
		}
	}

	stateData, err := os.ReadFile(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range installCases {
		if strings.Contains(string(stateData), test.token) {
			t.Fatalf("%s token leaked into state", test.tool)
		}
	}
	state, err := loadState(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range installCases {
		if state.Tools[test.tool] == nil || state.Tools[test.tool].Model != test.model {
			t.Fatalf("%s model was not stored: %#v", test.tool, state.Tools[test.tool])
		}
	}
	assertCredential(t, paths.HermesCredential, "hermes-new-secret")
	assertCredential(t, paths.OpenCodeCredential, "opencode-new-secret")
	assertCredential(t, paths.AiderCredential, "aider-new-secret")
	assertPrivateFileContains(t, paths.HermesEnv, `OPENAI_API_KEY="hermes-new-secret"`)
	assertPrivateFileContains(t, paths.AiderEnv, `OPENAI_API_KEY="aider-new-secret"`)

	if err := runDoctor(commonOptions{skipAPICheck: true}, "generic-test"); err != nil {
		t.Fatal(err)
	}
	if err := runUninstall(commonOptions{tools: "all"}); err != nil {
		t.Fatal(err)
	}
	for path, wanted := range map[string]string{
		paths.HermesConfig:   "customer-hermes",
		paths.HermesEnv:      "original-hermes-secret",
		paths.OpenCodeConfig: "Customer provider",
		paths.AiderConfig:    "dark-mode: true",
	} {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), wanted) {
			t.Fatalf("%s was not restored: data=%s err=%v", path, data, err)
		}
	}
	assertMissing(t, paths.AiderEnv)
}

func TestGenericInstallValidatesOnlySelectedModel(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "fake-bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFakeTool(t, binDir, "aider", "aider test")
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", filepath.Join(home, ".aiengine-setup"))
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"id":"customer-model"}]}`))
	}))
	defer server.Close()
	previousEndpoint := modelEndpoint
	modelEndpoint = server.URL
	t.Cleanup(func() { modelEndpoint = previousEndpoint })

	err := withTestStdin(t, "aider-token\n", func() error {
		return runInstall(commonOptions{tools: "aider", model: "customer-model", yes: true, tokenStdin: true}, "test")
	})
	if err != nil {
		t.Fatalf("generic tool should only validate its selected model: %v", err)
	}
}

func TestClaudeUninstallTreatsDeletedConfigAsAlreadyRemoved(t *testing.T) {
	paths := testPaths(t)
	_, _, state, err := prepareClaudeInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, snapshot, conflicts, remove, err := prepareClaudeUninstall(paths, state, false)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.existed || !remove || len(conflicts) != 0 {
		t.Fatalf("unexpected result for deleted config: existed=%v remove=%v conflicts=%v", snapshot.existed, remove, conflicts)
	}
}

func writeFakeTool(t *testing.T, dir, name, version string) {
	t.Helper()
	path := filepath.Join(dir, name)
	script := "#!/bin/sh\nprintf '%s\\n' '" + version + "'\n"
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
}

func withTestStdin(t *testing.T, contents string, run func() error) error {
	t.Helper()
	input, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		return err
	}
	defer input.Close()
	if _, err := input.WriteString(contents); err != nil {
		return err
	}
	if _, err := input.Seek(0, 0); err != nil {
		return err
	}
	previous := os.Stdin
	os.Stdin = input
	defer func() { os.Stdin = previous }()
	return run()
}

func assertCredential(t *testing.T, path, token string) {
	t.Helper()
	credential, err := os.ReadFile(path)
	if err != nil || strings.TrimSpace(string(credential)) != token {
		t.Fatalf("credential %s was not stored correctly: err=%v", path, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("credential %s permissions are not private: info=%v err=%v", path, info, err)
	}
}

func assertPrivateFileContains(t *testing.T, path, wanted string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), wanted) {
		t.Fatalf("private file %s does not contain expected data: err=%v", path, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("private file %s permissions are not private: info=%v err=%v", path, info, err)
	}
}

func assertSecretsNotLeaked(t *testing.T, paths Paths, tokens ...string) {
	t.Helper()
	for _, path := range []string{paths.ClaudeSettings, paths.CodexConfig, paths.State, paths.CodexAuth} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, token := range tokens {
			if strings.Contains(string(data), token) {
				t.Fatalf("secret leaked into %s", path)
			}
		}
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("managed file still exists: %s (err=%v)", path, err)
	}
}
