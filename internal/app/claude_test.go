package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testPaths(t *testing.T) Paths {
	t.Helper()
	root := t.TempDir()
	return Paths{
		Home:             root,
		BaseDir:          filepath.Join(root, ".aiengine-setup"),
		BinDir:           filepath.Join(root, ".aiengine-setup", "bin"),
		Binary:           filepath.Join(root, ".aiengine-setup", "bin", "aiengine-setup"),
		Credential:       filepath.Join(root, ".aiengine-setup", "credentials", "token"),
		ClaudeCredential: filepath.Join(root, ".aiengine-setup", "credentials", "claude-token"),
		CodexCredential:  filepath.Join(root, ".aiengine-setup", "credentials", "codex-token"),
		State:            filepath.Join(root, ".aiengine-setup", "state.json"),
		BackupDir:        filepath.Join(root, ".aiengine-setup", "backups"),
		ClaudeSettings:   filepath.Join(root, ".claude", "settings.json"),
		CodexConfig:      filepath.Join(root, ".codex", "config.toml"),
		CodexAuth:        filepath.Join(root, ".codex", "auth.json"),
	}
}

func TestClaudeInstallAndUninstallPreserveUnrelatedSettings(t *testing.T) {
	paths := testPaths(t)
	if err := os.MkdirAll(filepath.Dir(paths.ClaudeSettings), 0o700); err != nil {
		t.Fatal(err)
	}
	original := `{
  "model": "old-model",
  "permissions": {"allow": ["Read"]},
  "env": {"ANTHROPIC_MODEL": "old-env-model", "KEEP_ME": "yes"}
}`
	if err := os.WriteFile(paths.ClaudeSettings, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	installed, _, state, err := prepareClaudeInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(installed), "secret-token") {
		t.Fatal("generated Claude config contains a token")
	}
	if err := os.WriteFile(paths.ClaudeSettings, installed, 0o600); err != nil {
		t.Fatal(err)
	}

	var changed map[string]any
	if err := json.Unmarshal(installed, &changed); err != nil {
		t.Fatal(err)
	}
	if changed["apiKeyHelper"] != shellCommand(paths.Binary, "credential", "print", "claude") {
		t.Fatalf("Claude apiKeyHelper does not select the Claude credential: %v", changed["apiKeyHelper"])
	}
	changed["added_after_install"] = true
	changedData, _ := json.Marshal(changed)
	if err := os.WriteFile(paths.ClaudeSettings, changedData, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, _, conflicts, remove, err := prepareClaudeUninstall(paths, state, false)
	if err != nil {
		t.Fatal(err)
	}
	if remove || len(conflicts) != 0 {
		t.Fatalf("unexpected uninstall result: remove=%v conflicts=%v", remove, conflicts)
	}
	var result map[string]any
	if err := json.Unmarshal(restored, &result); err != nil {
		t.Fatal(err)
	}
	if result["model"] != "old-model" || result["added_after_install"] != true {
		t.Fatalf("original or later settings were not preserved: %#v", result)
	}
	env := result["env"].(map[string]any)
	if env["ANTHROPIC_MODEL"] != "old-env-model" || env["KEEP_ME"] != "yes" {
		t.Fatalf("original environment was not restored: %#v", env)
	}
	if _, exists := result["apiKeyHelper"]; exists {
		t.Fatal("managed apiKeyHelper was not removed")
	}
}

func TestClaudeUninstallDetectsManagedFieldConflict(t *testing.T) {
	paths := testPaths(t)
	installed, _, state, err := prepareClaudeInstall(paths, nil)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(installed, &config); err != nil {
		t.Fatal(err)
	}
	config["model"] = "user-changed-model"
	data, _ := json.Marshal(config)
	if err := os.MkdirAll(filepath.Dir(paths.ClaudeSettings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.ClaudeSettings, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, conflicts, _, err := prepareClaudeUninstall(paths, state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || conflicts[0] != "model" {
		t.Fatalf("expected model conflict, got %v", conflicts)
	}
}
