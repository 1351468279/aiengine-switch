package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDotEnvInstallRerunAndRestore(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	original := []byte("# keep this comment\nKEEP_ME=yes\nexport OPENAI_API_KEY='original-secret'\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}

	installed, snapshot, state, err := prepareDotEnvInstall(path, openAIAPIKey, "new-secret", nil)
	if err != nil {
		t.Fatal(err)
	}
	state.BackupPath = filepath.Join(t.TempDir(), "original.env")
	if err := os.WriteFile(state.BackupPath, snapshot.data, 0o600); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(mustMarshalJSON(t, state)), "new-secret") {
		t.Fatal("secret leaked into managed file state")
	}
	if err := os.WriteFile(path, installed, 0o600); err != nil {
		t.Fatal(err)
	}

	rerun, _, rerunState, err := prepareDotEnvInstall(path, openAIAPIKey, "rotated-secret", &state)
	if err != nil {
		t.Fatal(err)
	}
	if !rerunState.SecretFields[openAIAPIKey].OriginalExists {
		t.Fatal("rerun forgot that the original key existed")
	}
	if err := os.WriteFile(path, rerun, 0o600); err != nil {
		t.Fatal(err)
	}

	pending, conflicts, err := prepareDotEnvUninstall(rerunState, false)
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("prepare uninstall: conflicts=%v err=%v", conflicts, err)
	}
	if err := commitFiles([]pendingFile{pending}); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(restored) != string(original) {
		t.Fatalf("dotenv was not restored:\n%s", restored)
	}
}

func TestDotEnvUninstallDetectsChangedSecret(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	data, _, state, err := prepareDotEnvInstall(path, openAIAPIKey, "installed", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := setDotEnvValue(data, openAIAPIKey, "customer-change")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	_, conflicts, err := prepareDotEnvUninstall(state, false)
	if err != nil || len(conflicts) != 1 || conflicts[0] != openAIAPIKey {
		t.Fatalf("expected secret conflict, conflicts=%v err=%v", conflicts, err)
	}
}

func mustMarshalJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := marshalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
