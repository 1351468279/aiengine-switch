package app

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolvePathsUsesAiEngineNamesForFreshInstall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path assertion")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", "")
	t.Setenv("AIENGINE_SETUP_BINARY", "")
	t.Setenv("AIARE_SETUP_HOME", "")
	t.Setenv("AIARE_SETUP_BINARY", "")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.BaseDir != filepath.Join(home, ".aiengine-setup") {
		t.Fatalf("BaseDir = %q", paths.BaseDir)
	}
	if paths.Binary != filepath.Join(home, ".aiengine-setup", "bin", "aiengine-setup") {
		t.Fatalf("Binary = %q", paths.Binary)
	}
}

func TestResolvePathsReusesLegacyState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix path assertion")
	}
	home := t.TempDir()
	legacyBase := filepath.Join(home, ".aiare-setup")
	if err := os.MkdirAll(legacyBase, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyBase, "state.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", "")
	t.Setenv("AIENGINE_SETUP_BINARY", "")
	t.Setenv("AIARE_SETUP_HOME", "")
	t.Setenv("AIARE_SETUP_BINARY", "")

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.BaseDir != legacyBase || paths.Binary != filepath.Join(legacyBase, "bin", "aiare-setup") {
		t.Fatalf("legacy paths were not reused: %#v", paths)
	}
}

func TestResolvePathsPrefersAiEngineOverrides(t *testing.T) {
	home := t.TempDir()
	preferredBase := filepath.Join(home, "preferred")
	preferredBinary := filepath.Join(home, "preferred-bin")
	t.Setenv("HOME", home)
	t.Setenv("AIENGINE_SETUP_HOME", preferredBase)
	t.Setenv("AIENGINE_SETUP_BINARY", preferredBinary)
	t.Setenv("AIARE_SETUP_HOME", filepath.Join(home, "legacy"))
	t.Setenv("AIARE_SETUP_BINARY", filepath.Join(home, "legacy-bin"))

	paths, err := ResolvePaths()
	if err != nil {
		t.Fatal(err)
	}
	if paths.BaseDir != preferredBase || paths.Binary != preferredBinary {
		t.Fatalf("AiEngine overrides did not win: %#v", paths)
	}
}
