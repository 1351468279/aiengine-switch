package app

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func runUninstall(options commonOptions) error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	state, err := loadState(paths.State)
	if err != nil {
		return err
	}
	if state == nil {
		return fmt.Errorf("没有找到 AiEngine 安装状态")
	}
	tools, err := detectUninstallTools(options.tools, state.Tools)
	if err != nil {
		return err
	}
	pending := make([]pendingFile, 0, len(tools)+3)
	for _, tool := range tools {
		var data []byte
		var snapshot fileSnapshot
		var conflicts []string
		var remove bool
		switch tool {
		case "claude":
			data, snapshot, conflicts, remove, err = prepareClaudeUninstall(paths, state.Tools[tool], options.force)
		case "codex":
			data, snapshot, conflicts, remove, err = prepareCodexUninstall(state.Tools[tool], options.force)
		}
		if err != nil {
			return err
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("%s 配置在安装后被修改，未覆盖这些字段: %s；确认要恢复安装前状态时可使用 --force", tool, strings.Join(conflicts, ", "))
		}
		mode := os.FileMode(0o600)
		if snapshot.existed {
			mode = snapshot.mode
		}
		pending = append(pending, pendingFile{path: snapshot.path, data: data, mode: mode, remove: remove, snapshot: snapshot})
	}
	if options.dryRun {
		fmt.Printf("将卸载: %s\n", strings.Join(tools, ", "))
		fmt.Println("演练完成：未修改任何文件。")
		return nil
	}
	credentialPaths := make(map[string]bool)
	for _, tool := range tools {
		credentialPaths[credentialPathForState(state, tool, paths.credentialForTool(tool))] = true
		delete(state.Tools, tool)
	}
	if state.CredentialPath != "" && !credentialPathReferenced(state, state.CredentialPath) {
		credentialPaths[state.CredentialPath] = true
		state.CredentialPath = ""
	}
	for credentialPath := range credentialPaths {
		if credentialPath == "" || credentialPathReferenced(state, credentialPath) {
			continue
		}
		credentialSnapshot, err := snapshotFile(credentialPath)
		if err != nil {
			return err
		}
		pending = append(pending, pendingFile{path: credentialPath, remove: true, snapshot: credentialSnapshot})
	}
	state.SchemaVersion = stateSchema
	state.UpdatedAt = time.Now().UTC()
	stateSnapshot, err := snapshotFile(paths.State)
	if err != nil {
		return err
	}
	removeEverything := len(state.Tools) == 0
	if removeEverything {
		pending = append(pending, pendingFile{path: paths.State, remove: true, snapshot: stateSnapshot})
	} else {
		data, err := marshalState(state)
		if err != nil {
			return err
		}
		pending = append(pending, pendingFile{path: paths.State, data: data, mode: 0o600, snapshot: stateSnapshot})
	}
	if err := commitFiles(pending); err != nil {
		return fmt.Errorf("卸载失败，已尝试恢复本次改动: %w", err)
	}
	if removeEverything {
		if err := removeInstalledBinary(paths.Binary); err != nil {
			return fmt.Errorf("配置已恢复，但删除安装器失败: %w", err)
		}
	}
	fmt.Printf("已卸载: %s\n", strings.Join(tools, ", "))
	if removeEverything {
		fmt.Printf("配置备份仍保留在 %s\n", paths.BackupDir)
	}
	return nil
}
