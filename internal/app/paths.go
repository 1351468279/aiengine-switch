package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Paths struct {
	Home                string
	BaseDir             string
	BinDir              string
	Binary              string
	Credential          string
	ClaudeCredential    string
	DesktopCredential   string
	CodexCredential     string
	HermesCredential    string
	OpenCodeCredential  string
	AiderCredential     string
	State               string
	BackupDir           string
	ClaudeSettings      string
	DesktopNormalConfig string
	DesktopThreePConfig string
	DesktopProfile      string
	DesktopMeta         string
	CodexHome           string
	CodexConfig         string
	CodexSessions       string
	CodexAuth           string
	HermesConfig        string
	HermesEnv           string
	OpenCodeConfig      string
	AiderConfig         string
	AiderEnv            string
}

func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Paths{}, fmt.Errorf("无法确定用户主目录")
	}

	base := filepath.Join(home, ".aiengine-setup")
	legacyBase := filepath.Join(home, ".aiare-setup")
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		base = filepath.Join(local, "AiEngine", "CLISetup")
		legacyBase = filepath.Join(local, "AIARE", "CLISetup")
	}
	legacyLayout := false
	if configured := os.Getenv("AIENGINE_SETUP_HOME"); configured != "" {
		base = configured
	} else if configured := os.Getenv("AIARE_SETUP_HOME"); configured != "" {
		// Keep installations made before the AiEngine rename operable.
		base = configured
		legacyLayout = true
	} else if _, err := os.Stat(filepath.Join(legacyBase, "state.json")); err == nil {
		base = legacyBase
		legacyLayout = true
	}

	claudeDir := filepath.Join(home, ".claude")
	if configured := os.Getenv("CLAUDE_CONFIG_DIR"); configured != "" {
		claudeDir = configured
	}
	codexDir := filepath.Join(home, ".codex")
	if configured := os.Getenv("CODEX_HOME"); configured != "" {
		codexDir = configured
	}
	hermesDir := filepath.Join(home, ".hermes")
	if configured := os.Getenv("HERMES_HOME"); configured != "" {
		hermesDir = configured
	}
	openCodeConfig := ""
	if configured := os.Getenv("OPENCODE_CONFIG"); configured != "" {
		openCodeConfig = configured
	} else {
		configRoot := filepath.Join(home, ".config")
		if configured := os.Getenv("XDG_CONFIG_HOME"); configured != "" {
			configRoot = configured
		}
		openCodeConfig = filepath.Join(configRoot, "opencode", "opencode.json")
	}
	desktopNormalDir := ""
	desktopThreePDir := ""
	switch runtime.GOOS {
	case "darwin":
		applicationSupport := filepath.Join(home, "Library", "Application Support")
		desktopNormalDir = filepath.Join(applicationSupport, "Claude")
		desktopThreePDir = filepath.Join(applicationSupport, "Claude-3p")
	case "windows":
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		desktopNormalDir = windowsClaudeDir(local, false)
		desktopThreePDir = windowsClaudeDir(local, true)
	}
	desktopNormalConfig := ""
	desktopThreePConfig := ""
	desktopProfile := ""
	desktopMeta := ""
	if desktopNormalDir != "" && desktopThreePDir != "" {
		desktopLibrary := filepath.Join(desktopThreePDir, "configLibrary")
		desktopNormalConfig = filepath.Join(desktopNormalDir, "claude_desktop_config.json")
		desktopThreePConfig = filepath.Join(desktopThreePDir, "claude_desktop_config.json")
		desktopProfile = filepath.Join(desktopLibrary, desktopProfileID+".json")
		desktopMeta = filepath.Join(desktopLibrary, "_meta.json")
	}

	binaryName := "aiengine-setup"
	if legacyLayout {
		binaryName = "aiare-setup"
	}
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(base, "bin", binaryName)
	if configured := os.Getenv("AIENGINE_SETUP_BINARY"); configured != "" {
		binary = configured
	} else if configured := os.Getenv("AIARE_SETUP_BINARY"); configured != "" {
		// Deprecated compatibility override for existing installations.
		binary = configured
	}
	return Paths{
		Home:                home,
		BaseDir:             base,
		BinDir:              filepath.Join(base, "bin"),
		Binary:              binary,
		Credential:          filepath.Join(base, "credentials", "token"),
		ClaudeCredential:    filepath.Join(base, "credentials", "claude-token"),
		DesktopCredential:   filepath.Join(base, "credentials", "claude-desktop-token"),
		CodexCredential:     filepath.Join(base, "credentials", "codex-token"),
		HermesCredential:    filepath.Join(base, "credentials", "hermes-token"),
		OpenCodeCredential:  filepath.Join(base, "credentials", "opencode-token"),
		AiderCredential:     filepath.Join(base, "credentials", "aider-token"),
		State:               filepath.Join(base, "state.json"),
		BackupDir:           filepath.Join(base, "backups"),
		ClaudeSettings:      filepath.Join(claudeDir, "settings.json"),
		DesktopNormalConfig: desktopNormalConfig,
		DesktopThreePConfig: desktopThreePConfig,
		DesktopProfile:      desktopProfile,
		DesktopMeta:         desktopMeta,
		CodexHome:           codexDir,
		CodexConfig:         filepath.Join(codexDir, "config.toml"),
		CodexSessions:       filepath.Join(codexDir, "sessions"),
		CodexAuth:           filepath.Join(codexDir, "auth.json"),
		HermesConfig:        filepath.Join(hermesDir, "config.yaml"),
		HermesEnv:           filepath.Join(hermesDir, ".env"),
		OpenCodeConfig:      openCodeConfig,
		AiderConfig:         filepath.Join(home, ".aider.conf.yml"),
		AiderEnv:            filepath.Join(base, "clients", "aider.env"),
	}, nil
}

func windowsClaudeDir(local string, threeP bool) string {
	exactName := "Claude"
	if threeP {
		exactName = "Claude-3p"
	}
	exact := filepath.Join(local, exactName)
	if info, err := os.Stat(exact); err == nil && info.IsDir() {
		return exact
	}
	entries, err := os.ReadDir(local)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			isThreeP := strings.Contains(name, "-3p")
			if entry.IsDir() && strings.HasPrefix(name, "Claude") && isThreeP == threeP {
				return filepath.Join(local, name)
			}
		}
	}
	return exact
}

func (paths Paths) credentialForTool(tool string) string {
	switch tool {
	case "claude":
		return paths.ClaudeCredential
	case desktopTool:
		return paths.DesktopCredential
	case "codex":
		return paths.CodexCredential
	case "hermes":
		return paths.HermesCredential
	case "opencode":
		return paths.OpenCodeCredential
	case "aider":
		return paths.AiderCredential
	default:
		return ""
	}
}
