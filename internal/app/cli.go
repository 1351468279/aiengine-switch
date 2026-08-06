package app

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

type commonOptions struct {
	tools        string
	model        string
	dryRun       bool
	force        bool
	yes          bool
	tokenStdin   bool
	skipAPICheck bool
}

func Run(args []string, version string) error {
	if len(args) == 0 {
		printUsage(os.Stdout)
		return nil
	}
	switch args[0] {
	case "install":
		options, err := parseInstallOptions(args[1:])
		if err != nil {
			return err
		}
		return runInstall(options, version)
	case "doctor":
		options, err := parseDoctorOptions(args[1:])
		if err != nil {
			return err
		}
		return runDoctor(options, version)
	case "uninstall":
		options, err := parseUninstallOptions(args[1:])
		if err != nil {
			return err
		}
		return runUninstall(options)
	case "credential":
		if len(args) < 2 || len(args) > 3 || args[1] != "print" {
			return fmt.Errorf("credential 仅支持内部命令 print [claude|codex]")
		}
		tool := ""
		if len(args) == 3 {
			tool = args[2]
		}
		return printCredential(tool)
	case "version", "--version", "-v":
		fmt.Printf("aiengine-setup %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("未知命令 %q，请运行 aiengine-setup help", args[0])
	}
}

func newFlagSet(name string) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)
	set.SetOutput(io.Discard)
	return set
}

func parseInstallOptions(args []string) (commonOptions, error) {
	set := newFlagSet("install")
	var options commonOptions
	set.StringVar(&options.tools, "tools", "auto", "auto|claude|claude-desktop|codex|hermes|opencode|aider")
	set.StringVar(&options.model, "model", "", "Hermes、OpenCode 或 Aider 使用的模型")
	set.BoolVar(&options.yes, "yes", false, "跳过确认")
	set.BoolVar(&options.tokenStdin, "token-stdin", false, "从标准输入读取密钥")
	set.BoolVar(&options.dryRun, "dry-run", false, "只显示计划")
	set.BoolVar(&options.skipAPICheck, "skip-api-check", false, "跳过 API 验证")
	if err := set.Parse(args); err != nil {
		return options, fmt.Errorf("安装参数无效: %w", err)
	}
	if set.NArg() != 0 {
		return options, fmt.Errorf("安装命令不接受位置参数")
	}
	return options, nil
}

func parseDoctorOptions(args []string) (commonOptions, error) {
	set := newFlagSet("doctor")
	var options commonOptions
	set.BoolVar(&options.skipAPICheck, "skip-api-check", false, "跳过 API 验证")
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return options, fmt.Errorf("doctor 参数无效")
	}
	return options, nil
}

func parseUninstallOptions(args []string) (commonOptions, error) {
	set := newFlagSet("uninstall")
	var options commonOptions
	set.StringVar(&options.tools, "tools", "all", "all|claude|claude-desktop|codex|hermes|opencode|aider")
	set.BoolVar(&options.force, "force", false, "覆盖冲突并恢复原值")
	set.BoolVar(&options.dryRun, "dry-run", false, "只显示计划")
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return options, fmt.Errorf("卸载参数无效")
	}
	return options, nil
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, `AiEngine Setup - 配置主流 AI 客户端接入 AiEngine API

用法:
  aiengine-setup install [--tools auto|claude|claude-desktop|codex|hermes|opencode|aider] [--model MODEL] [--token-stdin] [--dry-run]
  aiengine-setup doctor [--skip-api-check]
  aiengine-setup uninstall [--tools all|claude|claude-desktop|codex|hermes|opencode|aider] [--force] [--dry-run]
  aiengine-setup version

每次安装只配置一个客户端；再次运行可添加另一个客户端或轮换对应密钥。
Hermes、OpenCode 和 Aider 可用 --model 指定该密钥能访问的模型。
安装器不会安装客户端。Claude Desktop 3P 接入仅支持 Windows 和 macOS。`)
}

func detectInstallTool(selection string) (string, error) {
	selection = strings.ToLower(strings.TrimSpace(selection))
	if selection == "" {
		selection = "auto"
	}
	switch selection {
	case "claude", "codex", "hermes", "opencode", "aider":
		if _, err := findTool(selection); err != nil {
			return "", fmt.Errorf("未检测到 %s，请先安装该 CLI", selection)
		}
		return selection, nil
	case desktopTool:
		if err := requireClaudeDesktopSupported(); err != nil {
			return "", err
		}
		return selection, nil
	case "all":
		return "", fmt.Errorf("安装时一次只能配置一个客户端，请明确指定 --tools")
	case "auto":
		var found []string
		for _, tool := range cliTools {
			if _, err := findTool(tool); err == nil {
				found = append(found, tool)
			}
		}
		if paths, err := ResolvePaths(); err == nil && claudeDesktopDetected(paths) {
			found = append(found, desktopTool)
		}
		switch len(found) {
		case 0:
			return "", fmt.Errorf("未检测到受支持的客户端；请先安装客户端，Desktop 用户也可明确使用 --tools claude-desktop")
		case 1:
			return found[0], nil
		default:
			return promptInstallTool(found)
		}
	default:
		return "", fmt.Errorf("--tools 必须是 auto、claude、claude-desktop、codex、hermes、opencode 或 aider")
	}
}

func promptInstallTool(found []string) (string, error) {
	terminal, err := openTerminalInput()
	if err != nil {
		return "", fmt.Errorf("检测到多个客户端但无法打开交互终端；请使用 --tools 明确指定")
	}
	defer terminal.Close()
	return promptInstallToolFrom(terminal, os.Stdout, found)
}

func promptInstallToolFrom(input io.Reader, output io.Writer, found []string) (string, error) {
	fmt.Fprintln(output, "检测到多个客户端，请选择本次要配置的客户端：")
	for index, tool := range found {
		fmt.Fprintf(output, "  %d) %s\n", index+1, toolDisplayName(tool))
	}
	reader := bufio.NewReader(input)
	for {
		fmt.Fprintf(output, "请输入 1-%d: ", len(found))
		line, readErr := reader.ReadString('\n')
		answer := strings.ToLower(strings.TrimSpace(line))
		for index, tool := range found {
			if answer == fmt.Sprint(index+1) || answer == tool {
				return tool, nil
			}
		}
		if readErr != nil {
			return "", fmt.Errorf("读取客户端选择: %w", readErr)
		}
		fmt.Fprintf(output, "选择无效，请输入 1-%d。\n", len(found))
	}
}

func toolDisplayName(tool string) string {
	switch tool {
	case "claude":
		return "Claude Code"
	case desktopTool:
		return "Claude Desktop"
	case "codex":
		return "Codex"
	case "hermes":
		return "Hermes Agent"
	case "opencode":
		return "OpenCode"
	case "aider":
		return "Aider"
	default:
		return tool
	}
}

func detectUninstallTools(selection string, installed map[string]*ToolState) ([]string, error) {
	selection = strings.ToLower(strings.TrimSpace(selection))
	if selection == "" {
		selection = "all"
	}
	var requested []string
	switch selection {
	case "all":
		requested = []string{"claude", desktopTool, "codex", "hermes", "opencode", "aider"}
	case "claude", desktopTool, "codex", "hermes", "opencode", "aider":
		requested = []string{selection}
	default:
		return nil, fmt.Errorf("--tools 必须是 all、claude、claude-desktop、codex、hermes、opencode 或 aider")
	}
	var result []string
	for _, tool := range requested {
		if installed[tool] != nil {
			result = append(result, tool)
		} else if selection != "all" {
			return nil, fmt.Errorf("%s 没有由本工具安装的配置", tool)
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("没有可卸载的受管配置")
	}
	sort.Strings(result)
	return result, nil
}

func containsTool(tools []string, wanted string) bool {
	for _, tool := range tools {
		if tool == wanted {
			return true
		}
	}
	return false
}
