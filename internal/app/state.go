package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type StoredValue struct {
	Exists bool            `json:"exists"`
	Value  json.RawMessage `json:"value,omitempty"`
}

type FieldState struct {
	Original  StoredValue `json:"original"`
	Installed StoredValue `json:"installed"`
}

type ToolState struct {
	ConfigPath      string                `json:"config_path"`
	ConfigExisted   bool                  `json:"config_existed"`
	BackupPath      string                `json:"backup_path,omitempty"`
	Fields          map[string]FieldState `json:"fields"`
	OriginalBlock   string                `json:"original_block,omitempty"`
	OriginalBlockOK bool                  `json:"original_block_existed"`
	InstalledBlock  string                `json:"installed_block,omitempty"`
}

type State struct {
	SchemaVersion    int                   `json:"schema_version"`
	InstallerVersion string                `json:"installer_version"`
	InstalledAt      time.Time             `json:"installed_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	BinaryPath       string                `json:"binary_path"`
	CredentialPath   string                `json:"credential_path"`
	Tools            map[string]*ToolState `json:"tools"`
}

func newState(version string, paths Paths) *State {
	now := time.Now().UTC()
	return &State{
		SchemaVersion:    stateSchema,
		InstallerVersion: version,
		InstalledAt:      now,
		UpdatedAt:        now,
		BinaryPath:       paths.Binary,
		CredentialPath:   paths.Credential,
		Tools:            make(map[string]*ToolState),
	}
}

func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取安装状态: %w", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("安装状态损坏: %w", err)
	}
	if state.SchemaVersion != stateSchema {
		return nil, fmt.Errorf("不支持的安装状态版本 %d", state.SchemaVersion)
	}
	if state.Tools == nil {
		state.Tools = make(map[string]*ToolState)
	}
	return &state, nil
}

func storedValue(value any, exists bool) (StoredValue, error) {
	if !exists {
		return StoredValue{}, nil
	}
	b, err := json.Marshal(value)
	if err != nil {
		return StoredValue{}, err
	}
	return StoredValue{Exists: true, Value: b}, nil
}

func (v StoredValue) decoded() (any, bool, error) {
	if !v.Exists {
		return nil, false, nil
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(v.Value))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func equalStored(value any, exists bool, expected StoredValue) bool {
	actual, err := storedValue(value, exists)
	if err != nil || actual.Exists != expected.Exists {
		return false
	}
	return !actual.Exists || bytes.Equal(actual.Value, expected.Value)
}
