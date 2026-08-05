package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	Home             string
	BaseDir          string
	BinDir           string
	Binary           string
	Credential       string
	ClaudeCredential string
	CodexCredential  string
	State            string
	BackupDir        string
	ClaudeSettings   string
	CodexConfig      string
	CodexAuth        string
}

func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return Paths{}, fmt.Errorf("无法确定用户主目录")
	}

	base := filepath.Join(home, ".aiare-setup")
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		if local == "" {
			local = filepath.Join(home, "AppData", "Local")
		}
		base = filepath.Join(local, "AIARE", "CLISetup")
	}
	if configured := os.Getenv("AIARE_SETUP_HOME"); configured != "" {
		base = configured
	}

	claudeDir := filepath.Join(home, ".claude")
	if configured := os.Getenv("CLAUDE_CONFIG_DIR"); configured != "" {
		claudeDir = configured
	}
	codexDir := filepath.Join(home, ".codex")
	if configured := os.Getenv("CODEX_HOME"); configured != "" {
		codexDir = configured
	}

	binaryName := "aiare-setup"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binary := filepath.Join(base, "bin", binaryName)
	if configured := os.Getenv("AIARE_SETUP_BINARY"); configured != "" {
		binary = configured
	}
	return Paths{
		Home:             home,
		BaseDir:          base,
		BinDir:           filepath.Join(base, "bin"),
		Binary:           binary,
		Credential:       filepath.Join(base, "credentials", "token"),
		ClaudeCredential: filepath.Join(base, "credentials", "claude-token"),
		CodexCredential:  filepath.Join(base, "credentials", "codex-token"),
		State:            filepath.Join(base, "state.json"),
		BackupDir:        filepath.Join(base, "backups"),
		ClaudeSettings:   filepath.Join(claudeDir, "settings.json"),
		CodexConfig:      filepath.Join(codexDir, "config.toml"),
		CodexAuth:        filepath.Join(codexDir, "auth.json"),
	}, nil
}

func (paths Paths) credentialForTool(tool string) string {
	switch tool {
	case "claude":
		return paths.ClaudeCredential
	case "codex":
		return paths.CodexCredential
	default:
		return ""
	}
}
