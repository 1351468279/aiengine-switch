package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var tomlTablePattern = regexp.MustCompile(`^\s*\[([^]]+)]\s*(?:#.*)?$`)

func parseTOML(data []byte, path string) (map[string]any, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := toml.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("解析 Codex 配置 %s: %w", path, err)
	}
	if value == nil {
		value = map[string]any{}
	}
	return value, nil
}

func codexValues() map[string]any {
	return map[string]any{
		"model":                    CodexModel,
		"model_provider":           codexProviderID,
		"model_reasoning_effort":   "high",
		"disable_response_storage": true,
	}
}

func tomlScalar(value any) (string, error) {
	switch value.(type) {
	case string, bool, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, json.Number:
		data, err := json.Marshal(value)
		return string(data), err
	default:
		return "", fmt.Errorf("不支持的 TOML 标量类型 %T", value)
	}
}

func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(text, "\n")
}

func tableStart(lines []string) int {
	for i, line := range lines {
		if tomlTablePattern.MatchString(line) {
			return i
		}
	}
	return len(lines)
}

func setTopLevel(text, key string, value any) (string, error) {
	scalar, err := tomlScalar(value)
	if err != nil {
		return "", err
	}
	lines := splitLines(text)
	limit := tableStart(lines)
	pattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
	match := -1
	for i := 0; i < limit; i++ {
		if pattern.MatchString(lines[i]) {
			if match >= 0 {
				return "", fmt.Errorf("Codex 顶层字段 %s 重复，无法安全编辑", key)
			}
			match = i
		}
	}
	assignment := key + " = " + scalar
	if match >= 0 {
		lines[match] = assignment
		return strings.Join(lines, "\n"), nil
	}
	lines = append(lines[:limit], append([]string{assignment}, lines[limit:]...)...)
	return strings.Join(lines, "\n"), nil
}

func removeTopLevel(text, key string) (string, error) {
	lines := splitLines(text)
	limit := tableStart(lines)
	pattern := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
	match := -1
	for i := 0; i < limit; i++ {
		if pattern.MatchString(lines[i]) {
			if match >= 0 {
				return "", fmt.Errorf("Codex 顶层字段 %s 重复，无法安全编辑", key)
			}
			match = i
		}
	}
	if match >= 0 {
		lines = append(lines[:match], lines[match+1:]...)
	}
	return strings.Join(lines, "\n"), nil
}

func providerHeading(line, providerID string) (bool, bool) {
	match := tomlTablePattern.FindStringSubmatch(line)
	if match == nil {
		return false, false
	}
	heading := strings.TrimSpace(match[1])
	root := "model_providers." + providerID
	return heading == root, strings.HasPrefix(heading, root+".")
}

func extractProviderBlock(text, providerID string) (before, block string, found bool, err error) {
	lines := splitLines(text)
	start, end := -1, -1
	for i, line := range lines {
		root, child := providerHeading(line, providerID)
		if start < 0 {
			if child {
				return "", "", false, fmt.Errorf("发现孤立的 [model_providers.%s.*] 配置", providerID)
			}
			if root {
				start = i
			}
			continue
		}
		if tomlTablePattern.MatchString(line) && !child {
			end = i
			break
		}
	}
	if start < 0 {
		return text, "", false, nil
	}
	if end < 0 {
		end = len(lines)
	}
	blockLines := append([]string(nil), lines[start:end]...)
	for len(blockLines) > 0 && strings.TrimSpace(blockLines[len(blockLines)-1]) == "" {
		blockLines = blockLines[:len(blockLines)-1]
	}
	remaining := append(append([]string(nil), lines[:start]...), lines[end:]...)
	return strings.TrimRight(strings.Join(remaining, "\n"), "\n"), strings.Join(blockLines, "\n") + "\n", true, nil
}

func providerIDFromBlock(block string) string {
	for _, providerID := range []string{codexProviderID, legacyProviderID} {
		for _, line := range splitLines(block) {
			root, _ := providerHeading(line, providerID)
			if root {
				return providerID
			}
		}
	}
	return codexProviderID
}

func codexProviderBlock(paths Paths) string {
	command, _ := tomlScalar(paths.Binary)
	return fmt.Sprintf(`[model_providers.%s]
name = "AiEngine NewAPI"
base_url = %q
wire_api = "responses"

[model_providers.%s.auth]
command = %s
args = ["credential", "print", "codex"]
timeout_ms = 5000
refresh_interval_ms = 0
`, codexProviderID, RelayV1URL, codexProviderID, command)
}

func appendProviderBlock(text, block string) string {
	trimmed := strings.TrimRight(text, "\n")
	if strings.TrimSpace(trimmed) == "" {
		return block
	}
	return trimmed + "\n\n" + block
}

func providerExists(parsed map[string]any, providerID string) bool {
	providers, ok := parsed["model_providers"].(map[string]any)
	if !ok {
		return false
	}
	_, ok = providers[providerID]
	return ok
}

func prepareCodexInstall(paths Paths, previous *ToolState) ([]byte, fileSnapshot, *ToolState, error) {
	snapshot, err := snapshotFile(paths.CodexConfig)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	if previous != nil && previous.ConfigPath != paths.CodexConfig {
		return nil, fileSnapshot{}, nil, fmt.Errorf("Codex 配置目录已从 %s 改为 %s，请先在原环境卸载", previous.ConfigPath, paths.CodexConfig)
	}
	parsed, err := parseTOML(snapshot.data, paths.CodexConfig)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	text := string(snapshot.data)
	managedProviderID := codexProviderID
	if previous != nil {
		managedProviderID = providerIDFromBlock(previous.InstalledBlock)
	}
	withoutProvider, oldBlock, blockFound, err := extractProviderBlock(text, managedProviderID)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	if providerExists(parsed, managedProviderID) && !blockFound {
		return nil, fileSnapshot{}, nil, fmt.Errorf("已有 %s provider 使用了无法安全编辑的内联或点号语法", managedProviderID)
	}
	if managedProviderID != codexProviderID {
		parsedWithoutLegacy, err := parseTOML([]byte(withoutProvider), paths.CodexConfig)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		_, _, currentFound, err := extractProviderBlock(withoutProvider, codexProviderID)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		if currentFound || providerExists(parsedWithoutLegacy, codexProviderID) {
			return nil, fileSnapshot{}, nil, fmt.Errorf("迁移旧配置时发现已有 %s provider，请先处理该配置", codexProviderID)
		}
	}
	state := &ToolState{ConfigPath: paths.CodexConfig, ConfigExisted: snapshot.existed, Fields: make(map[string]FieldState)}
	if previous != nil {
		state.ConfigExisted = previous.ConfigExisted
		state.BackupPath = previous.BackupPath
		state.OriginalBlock = previous.OriginalBlock
		state.OriginalBlockOK = previous.OriginalBlockOK
	} else {
		state.OriginalBlock = oldBlock
		state.OriginalBlockOK = blockFound
	}
	keys := append([]string(nil), managedCodexFields...)
	sort.Strings(keys)
	for _, key := range keys {
		wanted := codexValues()[key]
		current, exists := parsed[key]
		original, err := storedValue(current, exists)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		if previousField, ok := fieldFrom(previous, key); ok {
			original = previousField.Original
		}
		installed, err := storedValue(wanted, true)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		state.Fields[key] = FieldState{Original: original, Installed: installed}
		withoutProvider, err = setTopLevel(withoutProvider, key, wanted)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
	}
	state.InstalledBlock = codexProviderBlock(paths)
	result := appendProviderBlock(withoutProvider, state.InstalledBlock)
	if _, err := parseTOML([]byte(result), paths.CodexConfig); err != nil {
		return nil, fileSnapshot{}, nil, fmt.Errorf("生成的 Codex 配置无效: %w", err)
	}
	return []byte(result), snapshot, state, nil
}

func prepareCodexUninstall(state *ToolState, force bool) ([]byte, fileSnapshot, []string, bool, error) {
	snapshot, err := snapshotFile(state.ConfigPath)
	if err != nil {
		return nil, fileSnapshot{}, nil, false, err
	}
	if !snapshot.existed {
		return nil, snapshot, nil, true, nil
	}
	parsed, err := parseTOML(snapshot.data, state.ConfigPath)
	if err != nil {
		return nil, fileSnapshot{}, nil, false, err
	}
	providerID := providerIDFromBlock(state.InstalledBlock)
	text, currentBlock, found, err := extractProviderBlock(string(snapshot.data), providerID)
	if err != nil {
		return nil, fileSnapshot{}, nil, false, err
	}
	var conflicts []string
	for key, record := range state.Fields {
		current, exists := parsed[key]
		if !force && !equalStored(current, exists, record.Installed) {
			conflicts = append(conflicts, key)
			continue
		}
		original, existed, err := record.Original.decoded()
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
		if existed {
			text, err = setTopLevel(text, key, original)
		} else {
			text, err = removeTopLevel(text, key)
		}
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
	}
	if !force && (!found || currentBlock != state.InstalledBlock) {
		conflicts = append(conflicts, "model_providers."+providerID)
	} else if state.OriginalBlockOK {
		text = appendProviderBlock(text, state.OriginalBlock)
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, snapshot, conflicts, false, nil
	}
	text = strings.TrimSpace(text)
	if !state.ConfigExisted && text == "" {
		return nil, snapshot, nil, true, nil
	}
	if text != "" {
		text += "\n"
		if _, err := parseTOML([]byte(text), state.ConfigPath); err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
	}
	return []byte(text), snapshot, nil, false, nil
}
