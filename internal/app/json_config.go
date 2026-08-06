package app

import (
	"encoding/json"
	"fmt"
	"sort"
)

func prepareJSONToolInstall(path, label string, wanted map[string]any, previous *ToolState) ([]byte, fileSnapshot, *ToolState, error) {
	config, snapshot, err := readJSONObject(path)
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	if previous != nil && previous.ConfigPath != path {
		return nil, fileSnapshot{}, nil, fmt.Errorf("%s 配置目录已从 %s 改为 %s，请先在原环境卸载", label, previous.ConfigPath, path)
	}
	state := &ToolState{ConfigPath: path, ConfigExisted: snapshot.existed, Fields: make(map[string]FieldState)}
	if previous != nil {
		state.ConfigExisted = previous.ConfigExisted
		state.BackupPath = previous.BackupPath
	}
	for field, value := range wanted {
		current, exists := jsonPathGet(config, field)
		original, err := storedValue(current, exists)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		if old, ok := fieldFrom(previous, field); ok {
			original = old.Original
		}
		installed, err := storedValue(value, true)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
		state.Fields[field] = FieldState{Original: original, Installed: installed}
		if err := jsonPathSet(config, field, value); err != nil {
			return nil, fileSnapshot{}, nil, err
		}
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fileSnapshot{}, nil, err
	}
	return append(data, '\n'), snapshot, state, nil
}

func prepareJSONToolUninstall(state *ToolState, force bool) ([]byte, fileSnapshot, []string, bool, error) {
	config, snapshot, err := readJSONObject(state.ConfigPath)
	if err != nil {
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
	data, err := marshalJSON(config)
	return data, snapshot, nil, false, err
}

func checkJSONToolDoctor(report *doctorReport, label string, state *ToolState) {
	config, _, err := readJSONObject(state.ConfigPath)
	if err != nil {
		report.fail("%s 配置无效: %v", label, err)
		return
	}
	if !reportFieldsMatchJSON(config, state.Fields) {
		for field, expected := range state.Fields {
			current, exists := jsonPathGet(config, field)
			if !equalStored(current, exists, expected.Installed) {
				report.fail("%s 字段与安装状态不一致: %s", label, field)
			}
		}
		return
	}
	report.ok("%s 配置完整: %s", label, state.ConfigPath)
}
