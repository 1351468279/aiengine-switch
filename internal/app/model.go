package app

import (
	"fmt"
	"strings"
)

func modelForInstall(tool, requested string) (string, error) {
	requested = strings.TrimSpace(requested)
	if !genericTools[tool] {
		if requested != "" {
			return "", fmt.Errorf("--model 仅适用于 Hermes、OpenCode 和 Aider")
		}
		return defaultModelForTool(tool), nil
	}
	if requested == "" {
		requested = GeneralModel
	}
	if len(requested) > 256 || strings.ContainsAny(requested, " \t\r\n\x00") {
		return "", fmt.Errorf("模型 ID 格式无效: %q", requested)
	}
	return requested, nil
}

func defaultModelForTool(tool string) string {
	switch tool {
	case "claude", desktopTool:
		return ClaudeModel
	case "codex":
		return CodexModel
	case "hermes", "opencode", "aider":
		return GeneralModel
	default:
		return ""
	}
}

func configuredModel(tool string, state *ToolState) string {
	if state != nil && state.Model != "" {
		return state.Model
	}
	return defaultModelForTool(tool)
}

func requiredModelsForTool(tool, model string) []string {
	switch tool {
	case "claude", desktopTool:
		return []string{ClaudeModel, ClaudeOpusModel, ClaudeHaikuModel}
	case "codex":
		return []string{CodexModel}
	case "hermes", "opencode", "aider":
		if model != "" {
			return []string{model}
		}
	}
	return nil
}
