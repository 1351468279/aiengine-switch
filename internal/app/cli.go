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
	set.StringVar(&options.tools, "tools", "auto", "auto|claude|codex")
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
	set.StringVar(&options.tools, "tools", "all", "all|claude|codex")
	set.BoolVar(&options.force, "force", false, "覆盖冲突并恢复原值")
	set.BoolVar(&options.dryRun, "dry-run", false, "只显示计划")
	if err := set.Parse(args); err != nil || set.NArg() != 0 {
		return options, fmt.Errorf("卸载参数无效")
	}
	return options, nil
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, `AiEngine CLI Setup - 配置 Claude Code 或 Codex 接入 AiEngine API

用法:
  aiengine-setup install [--tools auto|claude|codex] [--token-stdin] [--dry-run]
  aiengine-setup doctor [--skip-api-check]
  aiengine-setup uninstall [--tools all|claude|codex] [--force] [--dry-run]
  aiengine-setup version

每次安装只配置一个客户端；再次运行可添加另一个客户端或轮换对应密钥。
安装器只配置已安装的 CLI，不会安装 Claude Code 或 Codex。`)
}

func detectInstallTool(selection string) (string, error) {
	selection = strings.ToLower(strings.TrimSpace(selection))
	if selection == "" {
		selection = "auto"
	}
	switch selection {
	case "claude", "codex":
		if _, err := findTool(selection); err != nil {
			return "", fmt.Errorf("未检测到 %s，请先安装该 CLI", selection)
		}
		return selection, nil
	case "all":
		return "", fmt.Errorf("安装时一次只能配置一个客户端，请使用 --tools claude 或 --tools codex")
	case "auto":
		var found []string
		for _, tool := range []string{"codex", "claude"} {
			if _, err := findTool(tool); err == nil {
				found = append(found, tool)
			}
		}
		switch len(found) {
		case 0:
			return "", fmt.Errorf("未检测到 Claude Code 或 Codex，请先至少安装一个 CLI")
		case 1:
			return found[0], nil
		default:
			return promptInstallTool()
		}
	default:
		return "", fmt.Errorf("--tools 必须是 auto、claude 或 codex")
	}
}

func promptInstallTool() (string, error) {
	terminal, err := openTerminalInput()
	if err != nil {
		return "", fmt.Errorf("检测到 Claude Code 和 Codex；无法打开交互终端，请使用 --tools claude 或 --tools codex")
	}
	defer terminal.Close()
	return promptInstallToolFrom(terminal, os.Stdout)
}

func promptInstallToolFrom(input io.Reader, output io.Writer) (string, error) {
	fmt.Fprintln(output, "检测到多个客户端，请选择本次要配置的客户端：")
	fmt.Fprintln(output, "  1) Codex")
	fmt.Fprintln(output, "  2) Claude Code")
	reader := bufio.NewReader(input)
	for {
		fmt.Fprint(output, "请输入 1 或 2: ")
		line, readErr := reader.ReadString('\n')
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "1", "codex":
			return "codex", nil
		case "2", "claude":
			return "claude", nil
		}
		if readErr != nil {
			return "", fmt.Errorf("读取客户端选择: %w", readErr)
		}
		fmt.Fprintln(output, "选择无效，请输入 1 或 2。")
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
		requested = []string{"claude", "codex"}
	case "claude", "codex":
		requested = []string{selection}
	default:
		return nil, fmt.Errorf("--tools 必须是 all、claude 或 codex")
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
