package app

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

var claudeConflictVariables = []string{
	"ANTHROPIC_API_KEY",
	"ANTHROPIC_AUTH_TOKEN",
	"CLAUDE_CODE_USE_BEDROCK",
	"CLAUDE_CODE_USE_FOUNDRY",
	"CLAUDE_CODE_USE_VERTEX",
}

func findTool(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", err
	}
	return path, nil
}

func claudeEnvironmentConflicts() []string {
	var conflicts []string
	for _, name := range claudeConflictVariables {
		if strings.TrimSpace(os.Getenv(name)) != "" {
			conflicts = append(conflicts, name)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func requireNoClaudeEnvironmentConflict() error {
	conflicts := claudeEnvironmentConflicts()
	if len(conflicts) == 0 {
		return nil
	}
	return fmt.Errorf("当前环境变量会覆盖 Claude 的中转站配置: %s；请先 unset/remove 后重试", strings.Join(conflicts, ", "))
}

func toolVersion(name string) (string, error) {
	path, err := findTool(name)
	if err != nil {
		return "", err
	}
	output, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s --version 失败: %w", name, err)
	}
	return strings.TrimSpace(string(output)), nil
}
