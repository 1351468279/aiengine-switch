package app

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func readYAMLDocument(path string) (*yaml.Node, fileSnapshot, error) {
	snapshot, err := snapshotFile(path)
	if err != nil {
		return nil, fileSnapshot{}, err
	}
	document := &yaml.Node{Kind: yaml.DocumentNode}
	if !snapshot.existed || len(bytes.TrimSpace(snapshot.data)) == 0 {
		document.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		return document, snapshot, nil
	}
	if err := yaml.Unmarshal(snapshot.data, document); err != nil {
		return nil, fileSnapshot{}, fmt.Errorf("解析 YAML 配置 %s: %w", path, err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return nil, fileSnapshot{}, fmt.Errorf("YAML 配置 %s 的根节点必须是对象", path)
	}
	return document, snapshot, nil
}

func yamlMapValue(node *yaml.Node, key string) (*yaml.Node, int, bool, error) {
	if node.Kind != yaml.MappingNode {
		return nil, -1, false, fmt.Errorf("字段父节点不是对象")
	}
	index := -1
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			if index >= 0 {
				return nil, -1, false, fmt.Errorf("字段 %s 重复，无法安全编辑", key)
			}
			index = i + 1
		}
	}
	if index < 0 {
		return nil, -1, false, nil
	}
	return node.Content[index], index, true, nil
}

func yamlPathNode(root *yaml.Node, path string) (*yaml.Node, bool, error) {
	current := root.Content[0]
	for _, part := range strings.Split(path, ".") {
		next, _, exists, err := yamlMapValue(current, part)
		if err != nil || !exists {
			return nil, false, err
		}
		current = next
	}
	return current, true, nil
}

func yamlPathGet(root *yaml.Node, path string) (any, bool, error) {
	node, exists, err := yamlPathNode(root, path)
	if err != nil || !exists {
		return nil, exists, err
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func yamlPathSet(root *yaml.Node, path string, value any) error {
	parts := strings.Split(path, ".")
	current := root.Content[0]
	for _, part := range parts[:len(parts)-1] {
		next, _, exists, err := yamlMapValue(current, part)
		if err != nil {
			return err
		}
		if !exists {
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, next)
		}
		if next.Kind != yaml.MappingNode {
			return fmt.Errorf("YAML 配置字段 %s 不是对象，无法安全合并", part)
		}
		current = next
	}
	var wanted yaml.Node
	if err := wanted.Encode(value); err != nil {
		return err
	}
	key := parts[len(parts)-1]
	existing, index, exists, err := yamlMapValue(current, key)
	if err != nil {
		return err
	}
	if exists {
		wanted.HeadComment = existing.HeadComment
		wanted.LineComment = existing.LineComment
		wanted.FootComment = existing.FootComment
		current.Content[index] = &wanted
	} else {
		current.Content = append(current.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, &wanted)
	}
	return nil
}

func yamlPathDelete(root *yaml.Node, path string) error {
	_, err := deleteYAMLPath(root.Content[0], strings.Split(path, "."))
	return err
}

func deleteYAMLPath(current *yaml.Node, parts []string) (bool, error) {
	_, index, exists, err := yamlMapValue(current, parts[0])
	if err != nil || !exists {
		return len(current.Content) == 0, err
	}
	keyIndex := index - 1
	if len(parts) == 1 {
		current.Content = append(current.Content[:keyIndex], current.Content[index+1:]...)
		return len(current.Content) == 0, nil
	}
	child := current.Content[index]
	if child.Kind != yaml.MappingNode {
		return false, nil
	}
	empty, err := deleteYAMLPath(child, parts[1:])
	if err != nil {
		return false, err
	}
	if empty {
		current.Content = append(current.Content[:keyIndex], current.Content[index+1:]...)
	}
	return len(current.Content) == 0, nil
}

func marshalYAML(document *yaml.Node) ([]byte, error) {
	var output bytes.Buffer
	encoder := yaml.NewEncoder(&output)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func prepareYAMLToolInstall(path, label string, wanted map[string]any, previous *ToolState) ([]byte, fileSnapshot, *ToolState, error) {
	document, snapshot, err := readYAMLDocument(path)
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
		current, exists, err := yamlPathGet(document, field)
		if err != nil {
			return nil, fileSnapshot{}, nil, err
		}
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
		if err := yamlPathSet(document, field, value); err != nil {
			return nil, fileSnapshot{}, nil, err
		}
	}
	data, err := marshalYAML(document)
	return data, snapshot, state, err
}

func prepareYAMLToolUninstall(state *ToolState, force bool) ([]byte, fileSnapshot, []string, bool, error) {
	document, snapshot, err := readYAMLDocument(state.ConfigPath)
	if err != nil {
		return nil, fileSnapshot{}, nil, false, err
	}
	if !snapshot.existed {
		return nil, snapshot, nil, true, nil
	}
	var conflicts []string
	for field, record := range state.Fields {
		current, exists, err := yamlPathGet(document, field)
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
		if !force && !equalStored(current, exists, record.Installed) {
			conflicts = append(conflicts, field)
			continue
		}
		original, existed, err := record.Original.decoded()
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
		if existed {
			err = yamlPathSet(document, field, original)
		} else {
			err = yamlPathDelete(document, field)
		}
		if err != nil {
			return nil, fileSnapshot{}, nil, false, err
		}
	}
	if len(conflicts) > 0 {
		sort.Strings(conflicts)
		return nil, snapshot, conflicts, false, nil
	}
	if !state.ConfigExisted && len(document.Content[0].Content) == 0 {
		return nil, snapshot, nil, true, nil
	}
	data, err := marshalYAML(document)
	return data, snapshot, nil, false, err
}

func checkYAMLToolDoctor(report *doctorReport, label string, state *ToolState) {
	document, snapshot, err := readYAMLDocument(state.ConfigPath)
	if err != nil || !snapshot.existed {
		report.fail("%s 配置不可读: %s", label, state.ConfigPath)
		return
	}
	matched := true
	for field, expected := range state.Fields {
		current, exists, err := yamlPathGet(document, field)
		if err != nil || !equalStored(current, exists, expected.Installed) {
			report.fail("%s 字段与安装状态不一致: %s", label, field)
			matched = false
		}
	}
	if matched {
		report.ok("%s 配置完整: %s", label, state.ConfigPath)
	}
}
