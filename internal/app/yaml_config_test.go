package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestYAMLInstallPreservesCommentsAndRestoresValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := "# account config\nmodel:\n  default: customer-model # keep model note\nui:\n  theme: dark\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	installed, _, state, err := prepareYAMLToolInstall(path, "test", map[string]any{
		"model.default":  "installed-model",
		"model.provider": "custom",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(installed), "installed-model # keep model note") {
		t.Fatalf("managed value comment was lost:\n%s", installed)
	}
	if err := os.WriteFile(path, installed, 0o600); err != nil {
		t.Fatal(err)
	}

	restored, _, conflicts, remove, err := prepareYAMLToolUninstall(state, false)
	if err != nil || len(conflicts) != 0 || remove {
		t.Fatalf("prepare uninstall: conflicts=%v remove=%v err=%v", conflicts, remove, err)
	}
	text := string(restored)
	for _, wanted := range []string{"customer-model # keep model note", "theme: dark"} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("YAML value was not restored or unrelated data was lost:\n%s", restored)
		}
	}
	if strings.Contains(text, "provider:") {
		t.Fatalf("installer-created YAML field remained after uninstall:\n%s", restored)
	}
}

func TestYAMLUninstallDetectsManagedFieldConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	installed, _, state, err := prepareYAMLToolInstall(path, "test", map[string]any{
		"model.default": "installed-model",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(installed), "installed-model", "customer-change", 1)
	if err := os.WriteFile(path, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, conflicts, _, err := prepareYAMLToolUninstall(state, false)
	if err != nil || len(conflicts) != 1 || conflicts[0] != "model.default" {
		t.Fatalf("expected managed field conflict, conflicts=%v err=%v", conflicts, err)
	}
}
