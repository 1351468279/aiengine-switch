package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
)

func readJSONObject(path string) (map[string]any, fileSnapshot, error) {
	snapshot, err := snapshotFile(path)
	if err != nil {
		return nil, fileSnapshot{}, err
	}
	if !snapshot.existed || len(bytes.TrimSpace(snapshot.data)) == 0 {
		return map[string]any{}, snapshot, nil
	}
	var value map[string]any
	decoder := json.NewDecoder(bytes.NewReader(snapshot.data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, fileSnapshot{}, fmt.Errorf("解析 Claude 配置 %s: %w", path, err)
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, snapshot, nil
}

func jsonPathGet(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var current any = root
	for _, part := range parts {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func jsonPathSet(root map[string]any, path string, value any) error {
	parts := strings.Split(path, ".")
	current := root
	for _, part := range parts[:len(parts)-1] {
		next, exists := current[part]
		if !exists {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			return fmt.Errorf("Claude 配置字段 %s 不是对象，无法安全合并", part)
		}
		current = child
	}
	current[parts[len(parts)-1]] = value
	return nil
}

func jsonPathDelete(root map[string]any, path string) {
	parts := strings.Split(path, ".")
	deleteJSONPath(root, parts)
}

func deleteJSONPath(current map[string]any, parts []string) bool {
	if len(parts) == 1 {
		delete(current, parts[0])
		return len(current) == 0
	}
	child, ok := current[parts[0]].(map[string]any)
	if !ok {
		return false
	}
	if deleteJSONPath(child, parts[1:]) {
		delete(current, parts[0])
	}
	return len(current) == 0
}

func claudeValues(paths Paths) map[string]any {
	return map[string]any{
		"apiKeyHelper":                       shellCommand(paths.Binary, "credential", "print"),
		"model":                              ClaudeModel,
		"env.ANTHROPIC_BASE_URL":             RelayRootURL,
		"env.ANTHROPIC_MODEL":                ClaudeModel,
		"env.ANTHROPIC_DEFAULT_SONNET_MODEL": ClaudeModel,
		"env.ANTHROPIC_DEFAULT_OPUS_MODEL":   ClaudeOpusModel,
		"env.ANTHROPIC_DEFAULT_HAIKU_MODEL":  ClaudeHaikuModel,
	}
}

func shellCommand(command string, args ...string) string {
	all := append([]string{command}, args...)
	quoted := make([]string, 0, len(all))
	for _, item := range all {
		if runtime.GOOS == "windows" {
			quoted = append(quoted, `"`+strings.ReplaceAll(item, `"`, `\"`)+`"`)
		} else {
			quoted = append(quoted, "'"+strings.ReplaceAll(item, "'", "'\\''")+"'")
		}
	}
	return strings.Join(quoted, " ")
}

func prepareClaudeInstall(paths Paths, previous *ToolState) ([]byte, fileSnapshot, *ToolState, error) {
	config, snapshot, err := readJSONObject(paths.ClaudeSettings)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	if previous != nil && previous.ConfigPath != paths.ClaudeSettings {
		return nil, fileSnapshot{}, nil, fmt.Errorf("Claude 配置目录已从 %s 改为 %s，请先在原环境卸载", previous.ConfigPath, paths.ClaudeSettings)
	}
	state := &ToolState{
		ConfigPath:    paths.ClaudeSettings,
		ConfigExisted: snapshot.existed,
		Fields:        make(map[string]FieldState),
	}
	if previous != nil {
		state.ConfigExisted = previous.ConfigExisted
		state.BackupPath = previous.BackupPath
	}
	for field, wanted := range claudeValues(paths) {
		current, exists := jsonPathGet(config, field)
		original, err := storedValue(current, exists)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		if previousField, ok := fieldFrom(previous, field); ok {
			original = previousField.Original
		}
		installed, err := storedValue(wanted, true)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		state.Fields[field] = FieldState{Original: original, Installed: installed}
		if err := jsonPathSet(config, field, wanted); err != nil {
			return nil, fileSnapshot{}, nil, err
		}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	return append(data, '\n'), snapshot, state, nil
}

func prepareClaudeUninstall(paths Paths, state *ToolState, force bool) ([]byte, fileSnapshot, []string, bool, error) {
	config, snapshot, err := readJSONObject(state.ConfigPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, snapshot, nil, true, nil
		}
		return nil, fileSnapshot{}, nil, false, err
	}
	if !snapshot.existed {
		return nil, snapshot, nil, true, nil
	}
	var conflicts []string
	for field, record := range state.Fields {
		current, exists := jsonPathGet(config, field)
		if !force && !equalStored(current, exists, record.Installed) {
			conflicts = append(conflicts, field)
			continue
		}
		original, existed, err := record.Original.decoded()
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
		if existed {
			if err := jsonPathSet(config, field, original); err != nil {
				return nil, fileSnapshot{}, nil, false, err
			}
		} else {
			jsonPathDelete(config, field)
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, snapshot, conflicts, false, nil
	}
	if !state.ConfigExisted && len(config) == 0 {
		return nil, snapshot, nil, true, nil
	}
	data, err := json.MarshalIndent(config, "", "  ")
	return append(data, '\n'), snapshot, nil, false, err
}

func fieldFrom(state *ToolState, field string) (FieldState, bool) {
	if state == nil {
		return FieldState{}, false
	}
	value, ok := state.Fields[field]
	return value, ok
}
