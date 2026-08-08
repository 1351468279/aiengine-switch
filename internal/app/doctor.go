package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type doctorReport struct {
	failures int
	warnings int
}

func (report *doctorReport) ok(format string, args ...any) {
	fmt.Printf("[OK] "+format+"\n", args...)
}

func (report *doctorReport) fail(format string, args ...any) {
	report.failures++
	fmt.Printf("[错误] "+format+"\n", args...)
}

func (report *doctorReport) warn(format string, args ...any) {
	report.warnings++
	fmt.Printf("[警告] "+format+"\n", args...)
}

func runDoctor(options commonOptions, version string) error {
	paths, err := ResolvePaths()
	if err != nil {
		return err
	}
	fmt.Printf("%s %s 诊断\n", ProductName, version)
	report := &doctorReport{}
	state, err := loadState(paths.State)
	if err != nil {
		report.fail("%v", err)
		return fmt.Errorf("诊断发现 %d 个错误", report.failures)
	}
	if state == nil {
		report.fail("未安装或状态文件不存在: %s", paths.State)
		return fmt.Errorf("诊断发现 %d 个错误", report.failures)
	}
	report.ok("安装状态有效（schema %d，安装器 %s）", state.SchemaVersion, state.InstallerVersion)

	if info, err := os.Stat(state.BinaryPath); err != nil {
		report.fail("安装器文件不可用: %s", state.BinaryPath)
	} else if !info.Mode().IsRegular() {
		report.fail("安装器路径不是普通文件: %s", state.BinaryPath)
	} else {
		report.ok("安装器路径: %s", state.BinaryPath)
	}

	if conflicts := claudeEnvironmentConflicts(); len(conflicts) > 0 && state.Tools["claude"] != nil {
		report.fail("环境变量会覆盖 Claude 配置: %s", strings.Join(conflicts, ", "))
	}

	toolNames := make([]string, 0, len(state.Tools))
	for name := range state.Tools {
		toolNames = append(toolNames, name)
	}
	sort.Strings(toolNames)
	for _, name := range toolNames {
		toolState := state.Tools[name]
		credentialPath := credentialPathForState(state, name, paths.credentialForTool(name))
		var token string
		if credentialPath == "" {
			report.fail("%s 凭据路径不存在", name)
		} else if tokenData, readErr := os.ReadFile(credentialPath); readErr != nil {
			report.fail("%s 凭据文件不可读: %s", name, credentialPath)
		} else if token, err = validateToken(string(tokenData)); err != nil {
			report.fail("%s 凭据文件无效: %v", name, err)
		} else if detail, securityErr := checkCredentialSecurity(credentialPath); securityErr != nil {
			report.fail("%s 凭据权限不安全: %v", name, securityErr)
		} else {
			report.ok("%s 凭据文件可用，%s", name, detail)
		}

		if name == desktopTool {
			if !claudeDesktopSupported() {
				report.fail("Claude Desktop 配置不能在当前系统使用")
			} else if !claudeDesktopDetected(paths) {
				report.warn("未检测到 Claude Desktop 应用；配置文件仍会保留")
			} else {
				report.ok("已检测到 Claude Desktop")
			}
		} else if versionText, err := toolVersion(name); err != nil {
			report.fail("%s CLI 不可用: %v", name, err)
		} else {
			report.ok("%s CLI: %s", name, versionText)
		}
		switch name {
		case "claude":
			checkClaudeDoctor(report, toolState)
		case "codex":
			checkCodexDoctor(report, toolState, paths)
		case desktopTool:
			checkClaudeDesktopDoctor(report, toolState, token)
		case "hermes", "opencode", "aider":
			checkGenericToolDoctor(report, name, toolState)
		default:
			report.warn("安装状态包含未知工具 %s", name)
		}
		if !options.skipAPICheck && token != "" && isSupportedTool(name) {
			if _, err := validateRequiredModels(token, requiredModelsForTool(name, configuredModel(name, toolState))); err != nil {
				report.fail("%s API 验证失败: %v", name, err)
			} else if name == desktopTool {
				if err := validateDesktopMessages(token); err != nil {
					report.fail("Claude Desktop Messages API 验证失败: %v", err)
				} else {
					report.ok("Claude Desktop API 密钥、模型和流式 Messages 接口均可用")
				}
			} else {
				report.ok("%s API 密钥有效，所需模型均可用", name)
			}
		}
	}

	if options.skipAPICheck {
		report.warn("已跳过 API 验证")
	}
	if state.Tools["codex"] != nil {
		report.ok("Codex 官方登录文件未纳入管理: %s", paths.CodexAuth)
	}
	if report.failures > 0 {
		return fmt.Errorf("诊断发现 %d 个错误、%d 个警告", report.failures, report.warnings)
	}
	fmt.Printf("诊断通过（%d 个警告）。\n", report.warnings)
	return nil
}

func isSupportedTool(name string) bool {
	return name == "claude" || name == desktopTool || name == "codex" || genericTools[name]
}

func checkClaudeDoctor(report *doctorReport, state *ToolState) {
	config, _, err := readJSONObject(state.ConfigPath)
	if err != nil {
		report.fail("Claude 配置无效: %v", err)
		return
	}
	for field, expected := range state.Fields {
		current, exists := jsonPathGet(config, field)
		if !equalStored(current, exists, expected.Installed) {
			report.fail("Claude 字段与安装状态不一致: %s", field)
		}
	}
	if reportFieldsMatchJSON(config, state.Fields) {
		report.ok("Claude 配置完整: %s", state.ConfigPath)
	}
}

func reportFieldsMatchJSON(config map[string]any, fields map[string]FieldState) bool {
	for field, expected := range fields {
		current, exists := jsonPathGet(config, field)
		if !equalStored(current, exists, expected.Installed) {
			return false
		}
	}
	return true
}

func checkCodexDoctor(report *doctorReport, state *ToolState, paths Paths) {
	snapshot, err := snapshotFile(state.ConfigPath)
	if err != nil || !snapshot.existed {
		report.fail("Codex 配置不可读: %s", state.ConfigPath)
		return
	}
	parsed, err := parseTOML(snapshot.data, state.ConfigPath)
	if err != nil {
		report.fail("Codex 配置无效: %v", err)
		return
	}
	matched := true
	for field, expected := range state.Fields {
		current, exists := parsed[field]
		if !equalStored(current, exists, expected.Installed) {
			report.fail("Codex 字段与安装状态不一致: %s", field)
			matched = false
		}
	}
	providerID := providerIDFromState(state)
	_, block, found, err := extractProviderBlock(string(snapshot.data), providerID)
	if err != nil || !found || block != state.InstalledBlock {
		report.fail("Codex %s provider 配置与安装状态不一致", providerID)
		matched = false
	}
	if currentProvider, ok := parsed["model_provider"].(string); !ok || currentProvider != providerID {
		report.fail("Codex 当前 model_provider 与已安装 provider 不一致: %s", providerID)
		matched = false
	} else {
		report.ok("Codex provider: %s", providerID)
	}

	historyPaths := paths
	if historyPaths.CodexConfig != state.ConfigPath {
		historyPaths.CodexHome = filepath.Dir(state.ConfigPath)
		historyPaths.CodexSessions = filepath.Join(filepath.Dir(state.ConfigPath), "sessions")
		report.warn("当前 CODEX_HOME 与安装状态不一致；正在检查安装状态对应的历史目录")
	}
	historyInfo, historyErr := os.Stat(historyPaths.CodexSessions)
	if errors.Is(historyErr, os.ErrNotExist) {
		report.ok("Codex 历史会话目录尚未创建: %s", historyPaths.CodexSessions)
	} else if historyErr != nil {
		report.warn("Codex 历史会话目录不可读: %s", historyPaths.CodexSessions)
	} else if !historyInfo.IsDir() {
		report.warn("Codex 历史会话路径不是目录: %s", historyPaths.CodexSessions)
	} else {
		counts, scanErr := codexHistoryProviderCounts(historyPaths)
		if scanErr != nil {
			report.warn("扫描 Codex 历史 provider 时遇到不可读文件: %v", scanErr)
		}
		providerIDs := sortedCodexProviderIDs(counts)
		if len(providerIDs) == 0 {
			report.ok("Codex 历史会话目录: %s（未发现 provider 元数据）", historyPaths.CodexSessions)
		} else if counts[providerID] == 0 {
			report.warn("Codex 当前 provider 为 %s，但历史会话使用 [%s]；历史文件未删除，可能暂时不在列表中显示", providerID, strings.Join(providerIDs, ", "))
		} else {
			report.ok("Codex 历史会话目录: %s（provider: %s）", historyPaths.CodexSessions, strings.Join(providerIDs, ", "))
		}
	}
	if matched {
		report.ok("Codex 配置完整: %s", state.ConfigPath)
	}
}
