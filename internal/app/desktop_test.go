package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func desktopTestPaths(t *testing.T) Paths {
	t.Helper()
	root := t.TempDir()
	threeP := filepath.Join(root, "Claude-3p")
	library := filepath.Join(threeP, "configLibrary")
	return Paths{
		BaseDir:             filepath.Join(root, "setup"),
		BackupDir:           filepath.Join(root, "setup", "backups"),
		DesktopCredential:   filepath.Join(root, "setup", "credentials", "claude-desktop-token"),
		DesktopNormalConfig: filepath.Join(root, "Claude", "claude_desktop_config.json"),
		DesktopThreePConfig: filepath.Join(threeP, "claude_desktop_config.json"),
		DesktopProfile:      filepath.Join(library, desktopProfileID+".json"),
		DesktopMeta:         filepath.Join(library, "_meta.json"),
	}
}

func TestClaudeDesktopInstallRotateAndUninstall(t *testing.T) {
	paths := desktopTestPaths(t)
	originalNormal := []byte("{\n  \"deploymentMode\": \"1p\",\n  \"theme\": \"dark\"\n}\n")
	originalProfile := []byte("{\n  \"existing\": true\n}\n")
	originalMeta := []byte("{\n  \"appliedId\": \"official\",\n  \"entries\": [{\"id\": \"official\", \"name\": \"Official\"}]\n}\n")
	for path, data := range map[string][]byte{
		paths.DesktopNormalConfig: originalNormal,
		paths.DesktopProfile:      originalProfile,
		paths.DesktopMeta:         originalMeta,
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	pending, state, err := prepareClaudeDesktopInstallFiles(paths, nil, "desktop-secret-one", time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := commitFiles(pending); err != nil {
		t.Fatal(err)
	}
	assertDesktopProfile(t, paths.DesktopProfile, "desktop-secret-one")
	stateData, err := marshalJSON(state)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(stateData), "desktop-secret-one") {
		t.Fatal("Claude Desktop secret leaked into installer state")
	}

	pending, state, err = prepareClaudeDesktopInstallFiles(paths, state, "desktop-secret-two", time.Unix(2, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := commitFiles(pending); err != nil {
		t.Fatal(err)
	}
	assertDesktopProfile(t, paths.DesktopProfile, "desktop-secret-two")
	meta, _, err := readJSONObject(paths.DesktopMeta)
	if err != nil {
		t.Fatal(err)
	}
	meta["entries"] = append(meta["entries"].([]any), map[string]any{"id": "added-later", "name": "Added later"})
	if err := writeJSON(paths.DesktopMeta, meta, 0o600); err != nil {
		t.Fatal(err)
	}

	normal, _, err := readJSONObject(paths.DesktopNormalConfig)
	if err != nil {
		t.Fatal(err)
	}
	normal["theme"] = "light"
	if err := writeJSON(paths.DesktopNormalConfig, normal, 0o600); err != nil {
		t.Fatal(err)
	}

	uninstall, conflicts, err := prepareClaudeDesktopUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("unexpected conflicts: %v", conflicts)
	}
	if err := commitFiles(uninstall); err != nil {
		t.Fatal(err)
	}
	normal, _, err = readJSONObject(paths.DesktopNormalConfig)
	if err != nil {
		t.Fatal(err)
	}
	if normal["deploymentMode"] != "1p" || normal["theme"] != "light" {
		t.Fatalf("normal config was not restored safely: %#v", normal)
	}
	if _, err := os.Stat(paths.DesktopThreePConfig); !os.IsNotExist(err) {
		t.Fatalf("new 3P config was not removed: %v", err)
	}
	assertFileEquals(t, paths.DesktopProfile, originalProfile)
	meta, _, err = readJSONObject(paths.DesktopMeta)
	if err != nil {
		t.Fatal(err)
	}
	if meta["appliedId"] != "official" {
		t.Fatalf("original applied profile was not restored: %#v", meta)
	}
	if strings.Contains(string(mustReadFile(t, paths.DesktopMeta)), desktopProfileID) {
		t.Fatal("AiEngine profile remained in Desktop metadata")
	}
	if !strings.Contains(string(mustReadFile(t, paths.DesktopMeta)), "added-later") {
		t.Fatal("unmanaged Desktop profile added after installation was removed")
	}
}

func TestClaudeDesktopUninstallDetectsProfileConflict(t *testing.T) {
	paths := desktopTestPaths(t)
	pending, state, err := prepareClaudeDesktopInstallFiles(paths, nil, "desktop-secret", time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := commitFiles(pending); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.DesktopProfile, []byte("{\"changed\":true}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, conflicts, err := prepareClaudeDesktopUninstall(state, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || !strings.Contains(conflicts[0], desktopProfileID) {
		t.Fatalf("expected profile conflict, got %v", conflicts)
	}
}

func assertDesktopProfile(t *testing.T, path, token string) {
	t.Helper()
	config, _, err := readJSONObject(path)
	if err != nil {
		t.Fatal(err)
	}
	if config["inferenceProvider"] != "gateway" || config["inferenceGatewayAuthScheme"] != "bearer" || config["inferenceGatewayBaseUrl"] != RelayRootURL || config["inferenceGatewayApiKey"] != token {
		t.Fatalf("invalid Desktop profile: %#v", config)
	}
	models, ok := config["inferenceModels"].([]any)
	if !ok || len(models) != 3 {
		t.Fatalf("invalid Desktop model list: %#v", config["inferenceModels"])
	}
}

func assertFileEquals(t *testing.T, path string, expected []byte) {
	t.Helper()
	actual := mustReadFile(t, path)
	var actualJSON any
	var expectedJSON any
	if err := json.Unmarshal(actual, &actualJSON); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(expected, &expectedJSON); err != nil {
		t.Fatal(err)
	}
	actualData, _ := json.Marshal(actualJSON)
	expectedData, _ := json.Marshal(expectedJSON)
	if string(actualData) != string(expectedData) {
		t.Fatalf("file %s differs: %s", path, actual)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
