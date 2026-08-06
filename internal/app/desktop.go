package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"
)

type desktopPreparedFile struct {
	data     []byte
	snapshot fileSnapshot
	state    ManagedFileState
}

func claudeDesktopSupported() bool {
	return runtime.GOOS == "darwin" || runtime.GOOS == "windows"
}

func requireClaudeDesktopSupported() error {
	if claudeDesktopSupported() {
		return nil
	}
	return fmt.Errorf("Claude Desktop 3P 接入仅支持 Windows 和 macOS；Linux/WSL 请使用 --tools claude 配置 Claude Code")
}

func claudeDesktopDetected(paths Paths) bool {
	if !claudeDesktopSupported() {
		return false
	}
	for _, path := range []string{filepath.Dir(paths.DesktopNormalConfig), filepath.Dir(paths.DesktopThreePConfig)} {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return true
		}
	}
	if runtime.GOOS == "darwin" {
		for _, path := range []string{"/Applications/Claude.app", filepath.Join(paths.Home, "Applications", "Claude.app")} {
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
	}
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		for _, path := range []string{
			filepath.Join(local, "Programs", "Claude", "Claude.exe"),
			filepath.Join(local, "AnthropicClaude", "Claude.exe"),
		} {
			if _, err := os.Stat(path); err == nil {
				return true
			}
		}
	}
	return false
}

func desktopProfile(token string) map[string]any {
	return map[string]any{
		"coworkEgressAllowedHosts":     []string{"*"},
		"disableDeploymentModeChooser": true,
		"inferenceGatewayApiKey":       token,
		"inferenceGatewayAuthScheme":   "bearer",
		"inferenceGatewayBaseUrl":      RelayRootURL,
		"inferenceModels":              []string{ClaudeModel, ClaudeOpusModel, ClaudeHaikuModel},
		"inferenceProvider":            "gateway",
	}
}

func desktopMetaValues(config map[string]any) (map[string]any, error) {
	entries := make([]any, 0)
	if current, exists := config["entries"]; exists {
		currentEntries, ok := current.([]any)
		if !ok {
			return nil, fmt.Errorf("Claude Desktop 配置库 entries 不是数组")
		}
		for _, entry := range currentEntries {
			object, ok := entry.(map[string]any)
			if ok && object["id"] == desktopProfileID {
				continue
			}
			entries = append(entries, entry)
		}
	}
	entries = append(entries, map[string]any{"id": desktopProfileID, "name": desktopProfileName})
	return map[string]any{"entries": entries, "appliedId": desktopProfileID}, nil
}

func previousManagedFile(previous *ToolState, path string) (ManagedFileState, bool) {
	if previous == nil {
		return ManagedFileState{}, false
	}
	for _, file := range previous.Files {
		if file.Path == path {
			return file, true
		}
	}
	return ManagedFileState{}, false
}

func prepareDesktopJSONFields(path string, wanted map[string]any, previous *ToolState) (desktopPreparedFile, error) {
	config, snapshot, err := readJSONObject(path)
	if err != nil {
		return desktopPreparedFile{}, err
	}
	fileState := ManagedFileState{Path: path, ConfigExisted: snapshot.existed, Fields: make(map[string]FieldState)}
	previousFile, hadPrevious := previousManagedFile(previous, path)
	if hadPrevious {
		fileState.ConfigExisted = previousFile.ConfigExisted
		fileState.BackupPath = previousFile.BackupPath
	}
	for field, value := range wanted {
		current, exists := jsonPathGet(config, field)
		original, err := storedValue(current, exists)
		if err != nil {
			return desktopPreparedFile{}, err
		}
		if old, ok := previousFile.Fields[field]; hadPrevious && ok {
			original = old.Original
		}
		installed, err := storedValue(value, true)
		if err != nil {
			return desktopPreparedFile{}, err
		}
		fileState.Fields[field] = FieldState{Original: original, Installed: installed}
		if err := jsonPathSet(config, field, value); err != nil {
			return desktopPreparedFile{}, err
		}
	}
	data, err := marshalJSON(config)
	if err != nil {
		return desktopPreparedFile{}, err
	}
	return desktopPreparedFile{data: data, snapshot: snapshot, state: fileState}, nil
}

func prepareDesktopProfile(path, token string, previous *ToolState) (desktopPreparedFile, error) {
	snapshot, err := snapshotFile(path)
	if err != nil {
		return desktopPreparedFile{}, err
	}
	data, err := marshalJSON(desktopProfile(token))
	if err != nil {
		return desktopPreparedFile{}, err
	}
	fileState := ManagedFileState{Path: path, ConfigExisted: snapshot.existed, InstalledSHA256: hashBytes(data)}
	if old, ok := previousManagedFile(previous, path); ok {
		fileState.ConfigExisted = old.ConfigExisted
		fileState.BackupPath = old.BackupPath
	}
	return desktopPreparedFile{data: data, snapshot: snapshot, state: fileState}, nil
}

func prepareClaudeDesktopInstall(paths Paths, previous *ToolState, token string, now time.Time) ([]pendingFile, *ToolState, error) {
	if err := requireClaudeDesktopSupported(); err != nil {
		return nil, nil, err
	}
	return prepareClaudeDesktopInstallFiles(paths, previous, token, now)
}

func prepareClaudeDesktopInstallFiles(paths Paths, previous *ToolState, token string, now time.Time) ([]pendingFile, *ToolState, error) {
	if previous != nil && previous.ConfigPath != paths.DesktopProfile {
		return nil, nil, fmt.Errorf("Claude Desktop 配置目录已从 %s 改为 %s，请先在原环境卸载", previous.ConfigPath, paths.DesktopProfile)
	}
	normal, err := prepareDesktopJSONFields(paths.DesktopNormalConfig, map[string]any{"deploymentMode": "3p"}, previous)
	if err != nil {
		return nil, nil, err
	}
	threeP, err := prepareDesktopJSONFields(paths.DesktopThreePConfig, map[string]any{"deploymentMode": "3p"}, previous)
	if err != nil {
		return nil, nil, err
	}
	profile, err := prepareDesktopProfile(paths.DesktopProfile, token, previous)
	if err != nil {
		return nil, nil, err
	}
	metaConfig, _, err := readJSONObject(paths.DesktopMeta)
	if err != nil {
		return nil, nil, err
	}
	metaValues, err := desktopMetaValues(metaConfig)
	if err != nil {
		return nil, nil, err
	}
	meta, err := prepareDesktopJSONFields(paths.DesktopMeta, metaValues, previous)
	if err != nil {
		return nil, nil, err
	}

	prepared := []desktopPreparedFile{normal, threeP, profile, meta}
	backupNames := []string{"claude-desktop-normal.json", "claude-desktop-3p.json", "claude-desktop-profile.json", "claude-desktop-meta.json"}
	files := make([]ManagedFileState, 0, len(prepared))
	pending := make([]pendingFile, 0, len(prepared))
	for index := range prepared {
		item := &prepared[index]
		if item.state.BackupPath == "" {
			backup, err := makeBackup(item.snapshot, paths.BackupDir, backupNames[index], now)
			if err != nil {
				return nil, nil, fmt.Errorf("备份 Claude Desktop 配置: %w", err)
			}
			item.state.BackupPath = backup
			if index == 2 && backup != "" {
				if err := secureCredential(backup); err != nil {
					return nil, nil, fmt.Errorf("保护 Claude Desktop profile 备份: %w", err)
				}
			}
		}
		mode := os.FileMode(0o600)
		if item.snapshot.existed {
			mode = item.snapshot.mode
		}
		pending = append(pending, pendingFile{path: item.state.Path, data: item.data, mode: mode, snapshot: item.snapshot})
		files = append(files, item.state)
	}
	return pending, &ToolState{
		ConfigPath:     paths.DesktopProfile,
		CredentialPath: paths.DesktopCredential,
		ConfigExisted:  profile.state.ConfigExisted,
		BackupPath:     profile.state.BackupPath,
		Fields:         make(map[string]FieldState),
		Files:          files,
	}, nil
}

func prepareDesktopFieldUninstall(file ManagedFileState, force bool) (pendingFile, []string, error) {
	config, snapshot, err := readJSONObject(file.Path)
	if err != nil {
		return pendingFile{}, nil, err
	}
	if !snapshot.existed {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	var conflicts []string
	for field, record := range file.Fields {
		current, exists := jsonPathGet(config, field)
		if !force && !equalStored(current, exists, record.Installed) {
			conflicts = append(conflicts, filepath.Base(file.Path)+":"+field)
			continue
		}
		original, existed, err := record.Original.decoded()
		if err != nil {
			return pendingFile{}, nil, err
		}
		if existed {
			if err := jsonPathSet(config, field, original); err != nil {
				return pendingFile{}, nil, err
			}
		} else {
			jsonPathDelete(config, field)
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return pendingFile{}, conflicts, nil
	}
	if !file.ConfigExisted && len(config) == 0 {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	data, err := marshalJSON(config)
	if err != nil {
		return pendingFile{}, nil, err
	}
	return pendingFile{path: file.Path, data: data, mode: snapshot.mode, snapshot: snapshot}, nil, nil
}

func desktopEntryWithID(entries []any, id string) (any, int, bool) {
	for index, entry := range entries {
		object, ok := entry.(map[string]any)
		if ok && object["id"] == id {
			return entry, index, true
		}
	}
	return nil, -1, false
}

func originalDesktopEntries(record FieldState) ([]any, error) {
	value, existed, err := record.Original.decoded()
	if err != nil || !existed {
		return nil, err
	}
	entries, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("Claude Desktop 原配置库 entries 不是数组")
	}
	return entries, nil
}

func prepareDesktopMetaUninstall(file ManagedFileState, force bool) (pendingFile, []string, error) {
	config, snapshot, err := readJSONObject(file.Path)
	if err != nil {
		return pendingFile{}, nil, err
	}
	if !snapshot.existed {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}

	entriesValue, exists := config["entries"]
	entries, ok := entriesValue.([]any)
	if !exists || !ok {
		if !force {
			return pendingFile{}, []string{filepath.Base(file.Path) + ":entries"}, nil
		}
		entries = []any{}
	}
	expectedEntry := map[string]any{"id": desktopProfileID, "name": desktopProfileName}
	currentEntry, currentIndex, currentExists := desktopEntryWithID(entries, desktopProfileID)
	if currentExists && !force {
		current, err := storedValue(currentEntry, true)
		if err != nil {
			return pendingFile{}, nil, err
		}
		expected, err := storedValue(expectedEntry, true)
		if err != nil {
			return pendingFile{}, nil, err
		}
		if !equalStoredValue(current, expected) {
			return pendingFile{}, []string{filepath.Base(file.Path) + ":entries[AiEngine]"}, nil
		}
	}

	originalEntries, err := originalDesktopEntries(file.Fields["entries"])
	if err != nil {
		return pendingFile{}, nil, err
	}
	originalEntry, _, originalExists := desktopEntryWithID(originalEntries, desktopProfileID)
	if currentExists {
		if originalExists {
			entries[currentIndex] = originalEntry
		} else {
			entries = append(entries[:currentIndex], entries[currentIndex+1:]...)
		}
	} else if originalExists {
		entries = append(entries, originalEntry)
	}
	config["entries"] = entries

	appliedRecord := file.Fields["appliedId"]
	currentApplied, appliedExists := config["appliedId"]
	if force || (appliedExists && currentApplied == desktopProfileID) {
		original, originalExists, err := appliedRecord.Original.decoded()
		if err != nil {
			return pendingFile{}, nil, err
		}
		if originalExists {
			config["appliedId"] = original
		} else {
			delete(config, "appliedId")
		}
	}

	if !file.ConfigExisted && len(config) == 0 {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	data, err := marshalJSON(config)
	if err != nil {
		return pendingFile{}, nil, err
	}
	return pendingFile{path: file.Path, data: data, mode: snapshot.mode, snapshot: snapshot}, nil, nil
}

func equalStoredValue(left, right StoredValue) bool {
	leftValue, leftExists, leftErr := left.decoded()
	_, rightExists, rightErr := right.decoded()
	return leftErr == nil && rightErr == nil && equalStored(leftValue, leftExists, StoredValue{Exists: rightExists, Value: right.Value})
}

func prepareDesktopProfileUninstall(file ManagedFileState, force bool) (pendingFile, []string, error) {
	snapshot, err := snapshotFile(file.Path)
	if err != nil {
		return pendingFile{}, nil, err
	}
	if !snapshot.existed {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	if !force && hashBytes(snapshot.data) != file.InstalledSHA256 {
		return pendingFile{}, []string{filepath.Base(file.Path)}, nil
	}
	if !file.ConfigExisted {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	if file.BackupPath == "" {
		return pendingFile{}, nil, fmt.Errorf("Claude Desktop profile 原配置备份不存在")
	}
	backup, err := snapshotFile(file.BackupPath)
	if err != nil {
		return pendingFile{}, nil, fmt.Errorf("读取 Claude Desktop profile 备份: %w", err)
	}
	if !backup.existed {
		return pendingFile{}, nil, fmt.Errorf("Claude Desktop profile 备份不存在: %s", file.BackupPath)
	}
	return pendingFile{path: file.Path, data: backup.data, mode: backup.mode, snapshot: snapshot}, nil, nil
}

func prepareClaudeDesktopUninstall(state *ToolState, force bool) ([]pendingFile, []string, error) {
	if len(state.Files) == 0 {
		return nil, nil, fmt.Errorf("Claude Desktop 安装状态缺少受管文件")
	}
	pending := make([]pendingFile, 0, len(state.Files))
	var conflicts []string
	for _, file := range state.Files {
		var item pendingFile
		var itemConflicts []string
		var err error
		if file.InstalledSHA256 != "" {
			item, itemConflicts, err = prepareDesktopProfileUninstall(file, force)
		} else if _, isMeta := file.Fields["entries"]; isMeta {
			item, itemConflicts, err = prepareDesktopMetaUninstall(file, force)
		} else {
			item, itemConflicts, err = prepareDesktopFieldUninstall(file, force)
		}
		if err != nil {
			return nil, nil, err
		}
		pending = append(pending, item)
		conflicts = append(conflicts, itemConflicts...)
	}
	sort.Strings(conflicts)
	return pending, conflicts, nil
}

func checkClaudeDesktopDoctor(report *doctorReport, state *ToolState, token string) {
	if len(state.Files) == 0 {
		report.fail("Claude Desktop 安装状态缺少受管文件")
		return
	}
	for _, file := range state.Files {
		snapshot, err := snapshotFile(file.Path)
		if err != nil || !snapshot.existed {
			report.fail("Claude Desktop 配置不可读: %s", file.Path)
			continue
		}
		if file.InstalledSHA256 != "" {
			if hashBytes(snapshot.data) != file.InstalledSHA256 {
				report.fail("Claude Desktop profile 与安装状态不一致: %s", file.Path)
				continue
			}
			config, _, err := readJSONObject(file.Path)
			if err != nil || config["inferenceGatewayApiKey"] != token || config["inferenceGatewayBaseUrl"] != RelayRootURL {
				report.fail("Claude Desktop profile 的 AiEngine 地址或密钥不一致: %s", file.Path)
				continue
			}
			if detail, err := checkCredentialSecurity(file.Path); err != nil {
				report.fail("Claude Desktop profile 权限不安全: %v", err)
			} else {
				report.ok("Claude Desktop profile 可用，%s", detail)
			}
			continue
		}
		config, _, err := readJSONObject(file.Path)
		matched := err == nil && reportFieldsMatchJSON(config, file.Fields)
		if _, isMeta := file.Fields["entries"]; isMeta && err == nil {
			entries, ok := config["entries"].([]any)
			entry, _, exists := desktopEntryWithID(entries, desktopProfileID)
			expected := map[string]any{"id": desktopProfileID, "name": desktopProfileName}
			actualValue, actualErr := storedValue(entry, exists)
			expectedValue, expectedErr := storedValue(expected, true)
			matched = ok && actualErr == nil && expectedErr == nil && equalStoredValue(actualValue, expectedValue) && config["appliedId"] == desktopProfileID
		}
		if !matched {
			report.fail("Claude Desktop 配置与安装状态不一致: %s", file.Path)
		} else {
			report.ok("Claude Desktop 配置完整: %s", file.Path)
		}
	}
}
