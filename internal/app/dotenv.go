package app

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type dotEnvAssignment struct {
	start int
	end   int
	value string
}

func findDotEnvAssignment(data []byte, key string) (dotEnvAssignment, bool, error) {
	start := 0
	var found dotEnvAssignment
	hadMatch := false
	for start < len(data) {
		end := bytes.IndexByte(data[start:], '\n')
		if end < 0 {
			end = len(data)
		} else {
			end += start + 1
		}
		lineEnd := end
		if lineEnd > start && data[lineEnd-1] == '\n' {
			lineEnd--
		}
		if lineEnd > start && data[lineEnd-1] == '\r' {
			lineEnd--
		}
		line := strings.TrimSpace(string(data[start:lineEnd]))
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		if line != "" && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[0]) == key {
				if hadMatch {
					return dotEnvAssignment{}, false, fmt.Errorf("环境文件字段 %s 重复，无法安全编辑", key)
				}
				value, err := parseDotEnvValue(strings.TrimSpace(parts[1]))
				if err != nil {
					return dotEnvAssignment{}, false, fmt.Errorf("解析环境文件字段 %s: %w", key, err)
				}
				found = dotEnvAssignment{start: start, end: end, value: value}
				hadMatch = true
			}
		}
		if end == len(data) {
			break
		}
		start = end
	}
	return found, hadMatch, nil
}

func parseDotEnvValue(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	if value[0] == '"' {
		parsed, err := strconv.Unquote(value)
		if err != nil {
			return "", err
		}
		return parsed, nil
	}
	if value[0] == '\'' {
		if len(value) < 2 || value[len(value)-1] != '\'' {
			return "", fmt.Errorf("单引号未闭合")
		}
		return value[1 : len(value)-1], nil
	}
	if index := strings.Index(value, " #"); index >= 0 {
		value = strings.TrimSpace(value[:index])
	}
	return value, nil
}

func setDotEnvValue(data []byte, key, value string) ([]byte, error) {
	assignment, exists, err := findDotEnvAssignment(data, key)
	if err != nil {
		return nil, err
	}
	line := []byte(key + "=" + strconv.Quote(value) + "\n")
	if exists {
		return append(append(append([]byte{}, data[:assignment.start]...), line...), data[assignment.end:]...), nil
	}
	result := append([]byte{}, data...)
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}
	return append(result, line...), nil
}

func removeDotEnvValue(data []byte, key string) ([]byte, error) {
	assignment, exists, err := findDotEnvAssignment(data, key)
	if err != nil || !exists {
		return data, err
	}
	return append(append([]byte{}, data[:assignment.start]...), data[assignment.end:]...), nil
}

func restoreDotEnvValue(data, backup []byte, key string) ([]byte, error) {
	original, exists, err := findDotEnvAssignment(backup, key)
	if err != nil {
		return nil, err
	}
	if !exists {
		return removeDotEnvValue(data, key)
	}
	current, currentExists, err := findDotEnvAssignment(data, key)
	if err != nil {
		return nil, err
	}
	originalLine := backup[original.start:original.end]
	if currentExists {
		return append(append(append([]byte{}, data[:current.start]...), originalLine...), data[current.end:]...), nil
	}
	result := append([]byte{}, data...)
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}
	return append(result, originalLine...), nil
}

func prepareDotEnvInstall(path, key, value string, previous *ManagedFileState) ([]byte, fileSnapshot, ManagedFileState, error) {
	snapshot, err := snapshotFile(path)
	if err != nil {
		return nil, fileSnapshot{}, ManagedFileState{}, err
	}
	if previous != nil && previous.Path != path {
		return nil, fileSnapshot{}, ManagedFileState{}, fmt.Errorf("环境文件已从 %s 改为 %s，请先在原环境卸载", previous.Path, path)
	}
	_, originalExists, err := findDotEnvAssignment(snapshot.data, key)
	if err != nil {
		return nil, fileSnapshot{}, ManagedFileState{}, err
	}
	state := ManagedFileState{
		Path:          path,
		ConfigExisted: snapshot.existed,
		SecretFields: map[string]SecretFieldState{
			key: {OriginalExists: originalExists, InstalledSHA256: hashBytes([]byte(value))},
		},
	}
	if previous != nil {
		state.ConfigExisted = previous.ConfigExisted
		state.BackupPath = previous.BackupPath
		if original, ok := previous.SecretFields[key]; ok {
			state.SecretFields[key] = SecretFieldState{OriginalExists: original.OriginalExists, InstalledSHA256: hashBytes([]byte(value))}
		}
	}
	data, err := setDotEnvValue(snapshot.data, key, value)
	return data, snapshot, state, err
}

func prepareDotEnvUninstall(file ManagedFileState, force bool) (pendingFile, []string, error) {
	snapshot, err := snapshotFile(file.Path)
	if err != nil {
		return pendingFile{}, nil, err
	}
	if !snapshot.existed {
		return pendingFile{path: file.Path, remove: true, snapshot: snapshot}, nil, nil
	}
	data := append([]byte{}, snapshot.data...)
	var conflicts []string
	for key, record := range file.SecretFields {
		current, exists, err := findDotEnvAssignment(data, key)
		if err != nil {
			return pendingFile{}, nil, err
		}
		if !force && (!exists || hashBytes([]byte(current.value)) != record.InstalledSHA256) {
			conflicts = append(conflicts, key)
			continue
		}
		if record.OriginalExists {
			if file.BackupPath == "" {
				return pendingFile{}, nil, fmt.Errorf("缺少 %s 的原配置备份", file.Path)
			}
			backup, err := os.ReadFile(file.BackupPath)
			if err != nil {
				return pendingFile{}, nil, fmt.Errorf("读取原配置备份 %s: %w", file.BackupPath, err)
			}
			data, err = restoreDotEnvValue(data, backup, key)
			if err != nil {
				return pendingFile{}, nil, err
			}
		} else {
			data, err = removeDotEnvValue(data, key)
			if err != nil {
				return pendingFile{}, nil, err
			}
		}
	}
	if len(conflicts) > 0 {
		return pendingFile{}, conflicts, nil
	}
	remove := !file.ConfigExisted && len(bytes.TrimSpace(data)) == 0
	mode := snapshot.mode
	if mode == 0 {
		mode = 0o600
	}
	return pendingFile{path: file.Path, data: data, mode: mode, remove: remove, snapshot: snapshot}, nil, nil
}

func checkDotEnvDoctor(report *doctorReport, label string, file ManagedFileState) {
	snapshot, err := snapshotFile(file.Path)
	if err != nil || !snapshot.existed {
		report.fail("%s 环境文件不可读: %s", label, file.Path)
		return
	}
	matched := true
	for key, expected := range file.SecretFields {
		current, exists, err := findDotEnvAssignment(snapshot.data, key)
		if err != nil || !exists || hashBytes([]byte(current.value)) != expected.InstalledSHA256 {
			report.fail("%s 环境字段与安装状态不一致: %s", label, key)
			matched = false
		}
	}
	if detail, err := checkCredentialSecurity(file.Path); err != nil {
		report.fail("%s 环境文件权限不安全: %v", label, err)
		matched = false
	} else if matched {
		report.ok("%s 环境文件可用，%s", label, detail)
	}
}
