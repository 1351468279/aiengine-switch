//go:build !windows

package app

import (
	"encoding/json"
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
	t.Setenv("AIARE_SETUP_HOME", filepath.Join(home, ".aiare-setup"))
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

	options := commonOptions{tools: "all", yes: true, tokenStdin: true, skipAPICheck: true}
	if err := withTestStdin(t, "first-test-token\n", func() error {
		return runInstall(options, "test")
	}); err != nil {
		t.Fatal(err)
	}
	assertSecretOnlyInCredential(t, paths, "first-test-token")

	if err := withTestStdin(t, "second-test-token\n", func() error {
		return runInstall(options, "test-rerun")
	}); err != nil {
		t.Fatal(err)
	}
	assertSecretOnlyInCredential(t, paths, "second-test-token")
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
	codexData, err := os.ReadFile(paths.CodexConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(codexData), "# customer comment") || !strings.Contains(string(codexData), `model = "official-model"`) {
		t.Fatalf("Codex settings were not restored:\n%s", codexData)
	}
	authData, err := os.ReadFile(paths.CodexAuth)
	if err != nil || string(authData) != originalAuth {
		t.Fatalf("Codex auth.json changed: data=%s err=%v", authData, err)
	}
	for _, removed := range []string{paths.Binary, paths.Credential, paths.State} {
		if _, err := os.Stat(removed); !os.IsNotExist(err) {
			t.Fatalf("managed file still exists after uninstall: %s", removed)
		}
	}
	entries, err := os.ReadDir(paths.BackupDir)
	if err != nil || len(entries) == 0 {
		t.Fatalf("initial backups were not retained: entries=%d err=%v", len(entries), err)
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

func assertSecretOnlyInCredential(t *testing.T, paths Paths, token string) {
	t.Helper()
	credential, err := os.ReadFile(paths.Credential)
	if err != nil || strings.TrimSpace(string(credential)) != token {
		t.Fatalf("credential was not stored correctly: err=%v", err)
	}
	for _, path := range []string{paths.ClaudeSettings, paths.CodexConfig, paths.State, paths.CodexAuth} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), token) {
			t.Fatalf("secret leaked into %s", path)
		}
	}
}
