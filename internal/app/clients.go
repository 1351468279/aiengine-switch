package app

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const openAIAPIKey = "OPENAI_API_KEY"

func prepareHermesInstall(paths Paths, previous *ToolState, token, model string, now time.Time) ([]pendingFile, *ToolState, error) {
	wanted := map[string]any{
		"model.default":  model,
		"model.provider": "custom",
		"model.base_url": RelayV1URL,
	}
	data, snapshot, state, err := prepareYAMLToolInstall(paths.HermesConfig, "Hermes", wanted, previous)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureToolBackup(paths, state, snapshot, "hermes-config.yaml", now, false); err != nil {
		return nil, nil, err
	}
	envPrevious, _ := previousManagedFile(previous, paths.HermesEnv)
	envData, envSnapshot, envState, err := prepareDotEnvInstall(paths.HermesEnv, openAIAPIKey, token, managedFilePointer(envPrevious))
	if err != nil {
		return nil, nil, err
	}
	if err := ensureManagedBackup(paths, &envState, envSnapshot, "hermes.env", now, true); err != nil {
		return nil, nil, err
	}
	state.Model = model
	state.Files = []ManagedFileState{envState}
	return []pendingFile{
		configPending(state.ConfigPath, data, snapshot),
		{path: envState.Path, data: envData, mode: 0o600, snapshot: envSnapshot},
	}, state, nil
}

func prepareOpenCodeInstall(paths Paths, previous *ToolState, model string, now time.Time) ([]pendingFile, *ToolState, error) {
	provider := map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "AiEngine",
		"options": map[string]any{
			"baseURL": RelayV1URL,
			"apiKey":  "{file:" + filepath.ToSlash(paths.OpenCodeCredential) + "}",
		},
		"models": map[string]any{
			model: map[string]any{"name": "AiEngine " + model},
		},
	}
	wanted := map[string]any{
		"$schema":           "https://opencode.ai/config.json",
		"model":             "aiengine/" + model,
		"provider.aiengine": provider,
	}
	data, snapshot, state, err := prepareJSONToolInstall(paths.OpenCodeConfig, "OpenCode", wanted, previous)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureToolBackup(paths, state, snapshot, "opencode.json", now, false); err != nil {
		return nil, nil, err
	}
	state.Model = model
	return []pendingFile{configPending(state.ConfigPath, data, snapshot)}, state, nil
}

func prepareAiderInstall(paths Paths, previous *ToolState, token, model string, now time.Time) ([]pendingFile, *ToolState, error) {
	wanted := map[string]any{
		"model":           "openai/" + model,
		"openai-api-base": RelayV1URL,
		"env-file":        paths.AiderEnv,
	}
	data, snapshot, state, err := prepareYAMLToolInstall(paths.AiderConfig, "Aider", wanted, previous)
	if err != nil {
		return nil, nil, err
	}
	if err := ensureToolBackup(paths, state, snapshot, "aider.conf.yml", now, false); err != nil {
		return nil, nil, err
	}
	envPrevious, _ := previousManagedFile(previous, paths.AiderEnv)
	envData, envSnapshot, envState, err := prepareDotEnvInstall(paths.AiderEnv, openAIAPIKey, token, managedFilePointer(envPrevious))
	if err != nil {
		return nil, nil, err
	}
	if err := ensureManagedBackup(paths, &envState, envSnapshot, "aider.env", now, true); err != nil {
		return nil, nil, err
	}
	state.Model = model
	state.Files = []ManagedFileState{envState}
	return []pendingFile{
		configPending(state.ConfigPath, data, snapshot),
		{path: envState.Path, data: envData, mode: 0o600, snapshot: envSnapshot},
	}, state, nil
}

func managedFilePointer(file ManagedFileState) *ManagedFileState {
	if file.Path == "" {
		return nil
	}
	return &file
}

func ensureToolBackup(paths Paths, state *ToolState, snapshot fileSnapshot, name string, now time.Time, secret bool) error {
	if state.BackupPath != "" {
		return nil
	}
	backup, err := makeBackup(snapshot, paths.BackupDir, name, now)
	if err != nil {
		return fmt.Errorf("备份 %s: %w", state.ConfigPath, err)
	}
	state.BackupPath = backup
	if secret && backup != "" {
		if err := secureCredential(backup); err != nil {
			return fmt.Errorf("保护配置备份 %s: %w", backup, err)
		}
	}
	return nil
}

func ensureManagedBackup(paths Paths, state *ManagedFileState, snapshot fileSnapshot, name string, now time.Time, secret bool) error {
	if state.BackupPath != "" {
		return nil
	}
	backup, err := makeBackup(snapshot, paths.BackupDir, name, now)
	if err != nil {
		return fmt.Errorf("备份 %s: %w", state.Path, err)
	}
	state.BackupPath = backup
	if secret && backup != "" {
		if err := secureCredential(backup); err != nil {
			return fmt.Errorf("保护环境文件备份 %s: %w", backup, err)
		}
	}
	return nil
}

func configPending(path string, data []byte, snapshot fileSnapshot) pendingFile {
	mode := os.FileMode(0o600)
	if snapshot.existed {
		mode = snapshot.mode
	}
	return pendingFile{path: path, data: data, mode: mode, snapshot: snapshot}
}

func prepareGenericToolUninstall(tool string, state *ToolState, force bool) ([]pendingFile, []string, error) {
	var data []byte
	var snapshot fileSnapshot
	var conflicts []string
	var remove bool
	var err error
	switch tool {
	case "opencode":
		data, snapshot, conflicts, remove, err = prepareJSONToolUninstall(state, force)
	case "hermes", "aider":
		data, snapshot, conflicts, remove, err = prepareYAMLToolUninstall(state, force)
	default:
		return nil, nil, fmt.Errorf("未知通用客户端 %s", tool)
	}
	if err != nil {
		return nil, nil, err
	}
	pending := []pendingFile{{path: snapshot.path, data: data, mode: configPending(snapshot.path, data, snapshot).mode, remove: remove, snapshot: snapshot}}
	for _, file := range state.Files {
		filePending, fileConflicts, err := prepareDotEnvUninstall(file, force)
		if err != nil {
			return nil, nil, err
		}
		for _, conflict := range fileConflicts {
			conflicts = append(conflicts, filepath.Base(file.Path)+":"+conflict)
		}
		pending = append(pending, filePending)
	}
	return pending, conflicts, nil
}

func checkGenericToolDoctor(report *doctorReport, tool string, state *ToolState) {
	switch tool {
	case "opencode":
		checkJSONToolDoctor(report, "OpenCode", state)
	case "hermes":
		checkYAMLToolDoctor(report, "Hermes", state)
	case "aider":
		checkYAMLToolDoctor(report, "Aider", state)
	}
	for _, file := range state.Files {
		checkDotEnvDoctor(report, toolDisplayName(tool), file)
	}
}
