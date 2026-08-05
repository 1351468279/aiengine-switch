package app

import (
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
		if len(args) != 2 || args[1] != "print" {
			return fmt.Errorf("credential 仅支持内部命令 print")
		}
		return printCredential()
	case "version", "--version", "-v":
		fmt.Printf("aiare-setup %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		return fmt.Errorf("未知命令 %q，请运行 aiare-setup help", args[0])
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
	set.StringVar(&options.tools, "tools", "auto", "auto|all|claude|codex")
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
	fmt.Fprintln(writer, `AIARE CLI Setup - 配置 Claude Code 和 Codex 接入 AIARE API

用法:
  aiare-setup install [--tools auto|all|claude|codex] [--token-stdin] [--dry-run]
  aiare-setup doctor [--skip-api-check]
  aiare-setup uninstall [--tools all|claude|codex] [--force] [--dry-run]
  aiare-setup version

安装器只配置已安装的 CLI，不会安装 Claude Code 或 Codex。`)
}

func detectTools(selection string, installed map[string]*ToolState, uninstall bool) ([]string, error) {
	selection = strings.ToLower(strings.TrimSpace(selection))
	if selection == "" {
		selection = "auto"
	}
	var requested []string
	switch selection {
	case "auto", "all":
		requested = []string{"claude", "codex"}
	case "claude", "codex":
		requested = []string{selection}
	default:
		return nil, fmt.Errorf("--tools 必须是 auto、all、claude 或 codex")
	}
	var result []string
	for _, tool := range requested {
		if uninstall {
			if installed[tool] != nil {
				result = append(result, tool)
			} else if selection != "all" && selection != "auto" {
				return nil, fmt.Errorf("%s 没有由本工具安装的配置", tool)
			}
			continue
		}
		if _, err := findTool(tool); err == nil {
			result = append(result, tool)
		} else if selection != "auto" && selection != "all" {
			return nil, fmt.Errorf("未检测到 %s，请先安装该 CLI", tool)
		}
	}
	if len(result) == 0 {
		if uninstall {
			return nil, fmt.Errorf("没有可卸载的受管配置")
		}
		return nil, fmt.Errorf("未检测到 Claude Code 或 Codex，请先至少安装一个 CLI")
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
