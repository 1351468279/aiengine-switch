package app

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type pendingFile struct {
	path     string
	data     []byte
	mode     os.FileMode
	remove   bool
	snapshot fileSnapshot
}

func runInstall(options commonOptions, version string) error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	state, err := loadState(paths.State)
	if err != nil {
		return err
	}
	if state == nil {
		state = newState(version, paths)
	}
	tools, err := detectTools(options.tools, state.Tools, false)
	if err != nil {
		return err
	}
	if containsTool(tools, "claude") {
		if err := requireNoClaudeEnvironmentConflict(); err != nil {
			return err
		}
	}

	fmt.Printf("将配置: %s\n", strings.Join(tools, ", "))
	fmt.Printf("API 地址: %s\n", RelayV1URL)
	fmt.Printf("安装目录: %s\n", paths.BaseDir)
	if options.dryRun {
		fmt.Println("演练完成：未读取密钥，也未修改任何文件。")
		return nil
	}
	if !options.yes {
		confirmed, err := confirmInstall()
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("用户取消安装")
		}
	}

	token, err := readToken(options.tokenStdin, paths.Credential)
	if err != nil {
		return err
	}
	if !options.skipAPICheck {
		if _, err := validateModels(token, tools); err != nil {
			return err
		}
		fmt.Println("API 密钥和模型权限验证通过。")
	} else {
		fmt.Println("已按要求跳过 API 验证。")
	}

	pending, nextState, err := prepareInstallFiles(paths, state, tools, token, version)
	if err != nil {
		return err
	}
	if err := commitFiles(pending); err != nil {
		return fmt.Errorf("安装失败，已尝试恢复本次改动: %w", err)
	}
	if err := secureCredential(paths.Credential); err != nil {
		rollbackFiles(pending)
		return fmt.Errorf("安装失败，已尝试恢复本次改动: %w", err)
	}
	_ = nextState
	fmt.Printf("配置完成。运行 %s doctor 可检查接入状态。\n", paths.Binary)
	return nil
}

func prepareInstallFiles(paths Paths, state *State, tools []string, token, version string) ([]pendingFile, *State, error) {
	now := time.Now().UTC()
	stateSnapshot, err := snapshotFile(paths.State)
	if err != nil {
		return nil, nil, err
	}
	credentialSnapshot, err := snapshotFile(paths.Credential)
	if err != nil {
		return nil, nil, err
	}
	pending := make([]pendingFile, 0, len(tools)+3)
	if !executableMatches(paths.Binary) {
		binarySnapshot, err := snapshotFile(paths.Binary)
		if err != nil {
			return nil, nil, err
		}
		executable, err := os.Executable()
		if err != nil {
			return nil, nil, fmt.Errorf("定位安装器自身: %w", err)
		}
		binaryData, err := os.ReadFile(executable)
		if err != nil {
			return nil, nil, fmt.Errorf("读取安装器自身: %w", err)
		}
		pending = append(pending, pendingFile{path: paths.Binary, data: binaryData, mode: 0o755, snapshot: binarySnapshot})
	}
	pending = append(pending, pendingFile{path: paths.Credential, data: []byte(token + "\n"), mode: 0o600, snapshot: credentialSnapshot})
	if containsTool(tools, "claude") {
		data, snapshot, toolState, err := prepareClaudeInstall(paths, state.Tools["claude"])
		if err != nil {
			return nil, nil, err
		}
		if toolState.BackupPath == "" {
			backup, err := makeBackup(snapshot, paths.BackupDir, "claude-settings.json", now)
			if err != nil {
				return nil, nil, fmt.Errorf("备份 Claude 配置: %w", err)
			}
			toolState.BackupPath = backup
		}
		mode := os.FileMode(0o600)
		if snapshot.existed {
			mode = snapshot.mode
		}
		pending = append(pending, pendingFile{path: paths.ClaudeSettings, data: data, mode: mode, snapshot: snapshot})
		state.Tools["claude"] = toolState
	}
	if containsTool(tools, "codex") {
		data, snapshot, toolState, err := prepareCodexInstall(paths, state.Tools["codex"])
		if err != nil {
			return nil, nil, err
		}
		if toolState.BackupPath == "" {
			backup, err := makeBackup(snapshot, paths.BackupDir, "codex-config.toml", now)
			if err != nil {
				return nil, nil, fmt.Errorf("备份 Codex 配置: %w", err)
			}
			toolState.BackupPath = backup
		}
		mode := os.FileMode(0o600)
		if snapshot.existed {
			mode = snapshot.mode
		}
		pending = append(pending, pendingFile{path: paths.CodexConfig, data: data, mode: mode, snapshot: snapshot})
		state.Tools["codex"] = toolState
	}
	state.SchemaVersion = stateSchema
	state.InstallerVersion = version
	state.UpdatedAt = now
	state.BinaryPath = paths.Binary
	state.CredentialPath = paths.Credential
	stateData, err := marshalState(state)
	if err != nil {
		return nil, nil, err
	}
	pending = append(pending, pendingFile{path: paths.State, data: stateData, mode: 0o600, snapshot: stateSnapshot})
	return pending, state, nil
}

func marshalState(state *State) ([]byte, error) {
	data, err := marshalJSON(state)
	if err != nil {
		return nil, fmt.Errorf("生成安装状态: %w", err)
	}
	return data, nil
}

func commitFiles(files []pendingFile) error {
	for index, file := range files {
		var err error
		if file.remove {
			err = os.Remove(file.path)
			if errors.Is(err, os.ErrNotExist) {
				err = nil
			}
		} else {
			err = atomicWriteFile(file.path, file.data, file.mode)
		}
		if err != nil {
			rollbackFiles(files[:index])
			return fmt.Errorf("写入 %s: %w", file.path, err)
		}
	}
	return nil
}

func rollbackFiles(files []pendingFile) {
	for index := len(files) - 1; index >= 0; index-- {
		_ = files[index].snapshot.restore()
	}
}

func confirmInstall() (bool, error) {
	terminal, err := openTerminalInput()
	if err != nil {
		return false, fmt.Errorf("无法读取确认；非交互安装请添加 --yes")
	}
	defer terminal.Close()
	fmt.Print("继续吗？[Y/n] ")
	line, err := bufio.NewReader(terminal).ReadString('\n')
	if err != nil && !errors.Is(err, os.ErrClosed) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "" || answer == "y" || answer == "yes", nil
}

func executableMatches(path string) bool {
	current, err := os.Executable()
	if err != nil {
		return false
	}
	a, errA := filepath.Abs(current)
	b, errB := filepath.Abs(path)
	return errA == nil && errB == nil && strings.EqualFold(a, b)
}
