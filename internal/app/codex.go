package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

var tomlTablePattern = regexp.MustCompile(`^\s*\[([^]]+)]\s*(?:#.*)?$`)
var codexProviderHeadingPattern = regexp.MustCompile(`^\s*\[model_providers\.([A-Za-z0-9_-]+)]\s*(?:#.*)?$`)
var codexProviderIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

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

func codexValues(providerID string) map[string]any {
	return map[string]any{
		"model":                    CodexModel,
		"model_provider":           providerID,
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
	for _, line := range splitLines(block) {
		match := codexProviderHeadingPattern.FindStringSubmatch(line)
		if match != nil && validCodexProviderID(match[1]) {
			return match[1]
		}
	}
	return codexProviderID
}

func providerIDFromState(state *ToolState) string {
	if state != nil && validCodexProviderID(state.ProviderID) {
		return state.ProviderID
	}
	if state != nil {
		return providerIDFromBlock(state.InstalledBlock)
	}
	return codexProviderID
}

func validCodexProviderID(providerID string) bool {
	return providerID != "" && codexProviderIDPattern.MatchString(providerID)
}

func reusableCodexProviderID(providerID string) bool {
	return validCodexProviderID(providerID) && providerID != codexBuiltinProviderID && providerID != legacyProviderID
}

func codexProviderBlock(paths Paths, providerID string) string {
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
`, providerID, RelayV1URL, providerID, command)
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

type codexSessionMeta struct {
	ModelProvider   string `json:"model_provider"`
	ModelProviderID string `json:"model_provider_id"`
	ThreadSettings  struct {
		ModelProviderID string `json:"model_provider_id"`
	} `json:"thread_settings"`
}

type codexSessionEvent struct {
	Type            string           `json:"type"`
	ModelProvider   string           `json:"model_provider"`
	ModelProviderID string           `json:"model_provider_id"`
	Payload         codexSessionMeta `json:"payload"`
}

func codexProviderFromSessionLine(line []byte) string {
	var event codexSessionEvent
	if err := json.Unmarshal(line, &event); err != nil {
		return ""
	}
	providerID := event.Payload.ModelProvider
	if providerID == "" {
		providerID = event.Payload.ModelProviderID
	}
	if providerID == "" {
		providerID = event.Payload.ThreadSettings.ModelProviderID
	}
	if providerID == "" {
		providerID = event.ModelProvider
	}
	if providerID == "" {
		providerID = event.ModelProviderID
	}
	if event.Type != "session_meta" && providerID == "" {
		return ""
	}
	if !reusableCodexProviderID(providerID) {
		return ""
	}
	return providerID
}

func codexHistoryProviderCounts(paths Paths) (map[string]int, error) {
	counts := make(map[string]int)
	if paths.CodexSessions == "" {
		return counts, nil
	}
	info, err := os.Stat(paths.CodexSessions)
	if errors.Is(err, os.ErrNotExist) {
		return counts, nil
	}
	if err != nil {
		return counts, err
	}
	if !info.IsDir() {
		return counts, fmt.Errorf("Codex 历史路径不是目录: %s", paths.CodexSessions)
	}

	var firstErr error
	err = filepath.WalkDir(paths.CodexSessions, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if firstErr == nil {
				firstErr = walkErr
			}
			return nil
		}
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			if firstErr == nil {
				firstErr = openErr
			}
			return nil
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			if providerID := codexProviderFromSessionLine(scanner.Bytes()); providerID != "" {
				counts[providerID]++
			}
		}
		if scanErr := scanner.Err(); scanErr != nil && firstErr == nil {
			firstErr = scanErr
		}
		if closeErr := file.Close(); closeErr != nil && firstErr == nil {
			firstErr = closeErr
		}
		return nil
	})
	if err != nil && firstErr == nil {
		firstErr = err
	}
	return counts, firstErr
}

func sortedCodexProviderIDs(counts map[string]int) []string {
	providerIDs := make([]string, 0, len(counts))
	for providerID := range counts {
		providerIDs = append(providerIDs, providerID)
	}
	sort.Slice(providerIDs, func(i, j int) bool {
		left, right := counts[providerIDs[i]], counts[providerIDs[j]]
		if left != right {
			return left > right
		}
		if providerIDs[i] == "OpenAI" {
			return true
		}
		if providerIDs[j] == "OpenAI" {
			return false
		}
		return providerIDs[i] < providerIDs[j]
	})
	return providerIDs
}

func historyCodexProviderID(paths Paths) string {
	counts, _ := codexHistoryProviderCounts(paths)
	providerIDs := sortedCodexProviderIDs(counts)
	if len(providerIDs) == 0 {
		return ""
	}
	return providerIDs[0]
}

func selectCodexProviderID(paths Paths, parsed map[string]any, previous *ToolState) string {
	if previous != nil {
		// New state records are authoritative. Older state files did not record
		// ProviderID, so their generated aiengine/aiare provider must not hide
		// the provider ID stored in existing session metadata.
		if previous.ProviderID != "" && reusableCodexProviderID(previous.ProviderID) {
			return previous.ProviderID
		}
		if previous.ProviderID == "" {
			previousID := providerIDFromBlock(previous.InstalledBlock)
			if reusableCodexProviderID(previousID) && previousID != codexProviderID {
				return previousID
			}
		}
	}
	currentID, currentOK := parsed["model_provider"].(string)
	if currentOK && reusableCodexProviderID(currentID) && currentID != codexProviderID {
		return currentID
	}
	if historyID := historyCodexProviderID(paths); historyID != "" {
		return historyID
	}
	if currentOK && reusableCodexProviderID(currentID) {
		return currentID
	}
	return codexProviderID
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
	managedProviderID := selectCodexProviderID(paths, parsed, previous)
	sourceProviderID := managedProviderID
	if previous != nil {
		previousProviderID := previous.ProviderID
		if previousProviderID == "" {
			previousProviderID = providerIDFromBlock(previous.InstalledBlock)
		}
		if previousProviderID == legacyProviderID || previousProviderID == codexProviderID {
			// Remove the provider generated by an older installer while allowing
			// session history to choose the ID used by the user's old sessions.
			sourceProviderID = previousProviderID
		} else if reusableCodexProviderID(previousProviderID) {
			sourceProviderID = previousProviderID
			managedProviderID = previousProviderID
		}
	}
	withoutProvider, oldBlock, blockFound, err := extractProviderBlock(text, sourceProviderID)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	if providerExists(parsed, sourceProviderID) && !blockFound {
		return nil, fileSnapshot{}, nil, fmt.Errorf("已有 %s provider 使用了无法安全编辑的内联或点号语法", sourceProviderID)
	}
	if sourceProviderID != managedProviderID {
		parsedWithoutSource, err := parseTOML([]byte(withoutProvider), paths.CodexConfig)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		_, _, currentFound, err := extractProviderBlock(withoutProvider, managedProviderID)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		if currentFound || providerExists(parsedWithoutSource, managedProviderID) {
			return nil, fileSnapshot{}, nil, fmt.Errorf("迁移旧配置时发现已有 %s provider，请先处理该配置", managedProviderID)
		}
	}
	state := &ToolState{ConfigPath: paths.CodexConfig, ProviderID: managedProviderID, ConfigExisted: snapshot.existed, Fields: make(map[string]FieldState)}
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
	wantedValues := codexValues(managedProviderID)
	for _, key := range keys {
		wanted := wantedValues[key]
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
	state.InstalledBlock = codexProviderBlock(paths, managedProviderID)
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
	providerID := providerIDFromState(state)
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
