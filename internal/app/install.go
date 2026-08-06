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
	tool, err := detectInstallTool(options.tools)
	if err != nil {
		return err
	}
	model, err := modelForInstall(tool, options.model)
	if err != nil {
		return err
	}
	if tool == "claude" {
		if err := requireNoClaudeEnvironmentConflict(); err != nil {
			return err
		}
	}

	fmt.Printf("将配置: %s\n", toolDisplayName(tool))
	fmt.Printf("API 地址: %s\n", RelayV1URL)
	if genericTools[tool] {
		fmt.Printf("模型: %s\n", model)
	}
	fmt.Printf("安装目录: %s\n", paths.BaseDir)
	if tool == desktopTool {
		fmt.Println("请先完全退出 Claude Desktop；配置完成后重新打开。")
	}
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

	credentialPath := paths.credentialForTool(tool)
	currentCredentialPath := credentialPathForState(state, tool, credentialPath)
	token, err := readToken(options.tokenStdin, currentCredentialPath)
	if err != nil {
		return err
	}
	if !options.skipAPICheck {
		if _, err := validateRequiredModels(token, requiredModelsForTool(tool, model)); err != nil {
			return err
		}
		if tool == desktopTool {
			if err := validateDesktopMessages(token); err != nil {
				return err
			}
		}
		fmt.Println("API 密钥和模型权限验证通过。")
	} else {
		fmt.Println("已按要求跳过 API 验证。")
	}

	pending, nextState, err := prepareInstallFiles(paths, state, tool, token, model, version)
	if err != nil {
		return err
	}
	if err := commitFiles(pending); err != nil {
		return fmt.Errorf("安装失败，已尝试恢复本次改动: %w", err)
	}
	for _, secretPath := range secretPathsForTool(paths, tool) {
		if err := secureCredential(secretPath); err != nil {
			rollbackFiles(pending)
			return fmt.Errorf("安装失败，已尝试恢复本次改动: %w", err)
		}
	}
	_ = nextState
	if tool == desktopTool {
		fmt.Println("Claude Desktop 配置完成。现在重新打开应用，并在模型菜单中选择 AiEngine 的 Claude 模型。")
	}
	fmt.Printf("运行 %s doctor 可检查接入状态。\n", paths.Binary)
	return nil
}

func prepareInstallFiles(paths Paths, state *State, tool, token, model, version string) ([]pendingFile, *State, error) {
	now := time.Now().UTC()
	credentialPath := paths.credentialForTool(tool)
	previousCredentialPath := credentialPathForState(state, tool, credentialPath)
	stateSnapshot, err := snapshotFile(paths.State)
	if err != nil {
		return nil, nil, err
	}
	credentialSnapshot, err := snapshotFile(credentialPath)
	if err != nil {
		return nil, nil, err
	}
	pending := make([]pendingFile, 0, 6)
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
	pending = append(pending, pendingFile{path: credentialPath, data: []byte(token + "\n"), mode: 0o600, snapshot: credentialSnapshot})
	if tool == "claude" {
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
		toolState.CredentialPath = credentialPath
		toolState.Model = model
		state.Tools["claude"] = toolState
	}
	if tool == "codex" {
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
		toolState.CredentialPath = credentialPath
		toolState.Model = model
		state.Tools["codex"] = toolState
	}
	if tool == desktopTool {
		desktopPending, toolState, err := prepareClaudeDesktopInstall(paths, state.Tools[desktopTool], token, now)
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, desktopPending...)
		toolState.CredentialPath = credentialPath
		toolState.Model = model
		state.Tools[desktopTool] = toolState
	}
	if tool == "hermes" {
		toolPending, toolState, err := prepareHermesInstall(paths, state.Tools[tool], token, model, now)
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, toolPending...)
		toolState.CredentialPath = credentialPath
		state.Tools[tool] = toolState
	}
	if tool == "opencode" {
		toolPending, toolState, err := prepareOpenCodeInstall(paths, state.Tools[tool], model, now)
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, toolPending...)
		toolState.CredentialPath = credentialPath
		state.Tools[tool] = toolState
	}
	if tool == "aider" {
		toolPending, toolState, err := prepareAiderInstall(paths, state.Tools[tool], token, model, now)
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, toolPending...)
		toolState.CredentialPath = credentialPath
		state.Tools[tool] = toolState
	}
	state.SchemaVersion = stateSchema
	state.InstallerVersion = version
	state.UpdatedAt = now
	state.BinaryPath = paths.Binary
	cleanupPaths := map[string]bool{
		previousCredentialPath: true,
		state.CredentialPath:   true,
	}
	for cleanupPath := range cleanupPaths {
		if cleanupPath == "" || cleanupPath == credentialPath || credentialPathReferenced(state, cleanupPath) {
			continue
		}
		snapshot, err := snapshotFile(cleanupPath)
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, pendingFile{path: cleanupPath, remove: true, snapshot: snapshot})
	}
	if state.CredentialPath != "" && !credentialPathReferenced(state, state.CredentialPath) {
		state.CredentialPath = ""
	}
	stateData, err := marshalState(state)
	if err != nil {
		return nil, nil, err
	}
	pending = append(pending, pendingFile{path: paths.State, data: stateData, mode: 0o600, snapshot: stateSnapshot})
	return pending, state, nil
}

func secretPathsForTool(paths Paths, tool string) []string {
	result := []string{paths.credentialForTool(tool)}
	switch tool {
	case desktopTool:
		result = append(result, paths.DesktopProfile)
	case "hermes":
		result = append(result, paths.HermesEnv)
	case "aider":
		result = append(result, paths.AiderEnv)
	}
	return result
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
