package app

import (
	"strings"
	"testing"
)

func TestModelForInstallDefaultsGenericClients(t *testing.T) {
	for _, tool := range []string{"hermes", "opencode", "aider"} {
		model, err := modelForInstall(tool, "")
		if err != nil || model != GeneralModel {
			t.Fatalf("%s default model = %q, err=%v", tool, model, err)
		}
	}
}

func TestModelForInstallRejectsInvalidOrFixedClientOverride(t *testing.T) {
	if _, err := modelForInstall("codex", "custom-model"); err == nil {
		t.Fatal("Codex accepted a model override")
	}
	for _, model := range []string{"two models", "line\nbreak", strings.Repeat("x", 257)} {
		if _, err := modelForInstall("hermes", model); err == nil {
			t.Fatalf("accepted invalid model ID %q", model)
		}
	}
}
