# AIARE CLI Setup

为 AIARE API 中转站配置现有的 Claude Code 和 Codex CLI。用户只需运行一条在线命令并输入自己的 API 密钥，不需要安装桌面切换工具。

## 一键接入

开始前请先安装至少一个目标 CLI，并在 AIARE 控制台创建 API 密钥。

macOS、Linux 或 WSL：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://modelapi.aiaiaiaiai.cloud/install.ps1 | iex
```

安装器会自动检测 Claude Code 和 Codex，显示修改计划，要求确认，然后在本机隐藏输入密钥。输入的密钥会先通过 `https://modelapi.aiaiaiaiai.cloud/v1/models` 验证。

## 安装器做什么

- Claude Code：合并修改 `~/.claude/settings.json`（或 `CLAUDE_CONFIG_DIR`），配置 AIARE 地址、默认模型和 `apiKeyHelper`。
- Codex：合并修改 `~/.codex/config.toml`（或 `CODEX_HOME`），添加 `aiare` Responses provider 和凭据命令。
- 密钥：仅写入安装器专用的受保护文件，不出现在命令行参数、Claude/Codex 配置或安装状态中。
- 备份：首次修改前保留原配置；卸载时只恢复本工具管理且未被用户再次修改的字段。
- Codex 登录：不会读取或修改 `~/.codex/auth.json`，官方登录状态会保留。

默认模型：

| 客户端 | 模型 |
| --- | --- |
| Claude Code | `claude-sonnet-5` |
| Claude Code Opus | `claude-opus-5` |
| Claude Code Haiku | `claude-haiku-4-5-20251001` |
| Codex | `gpt-5.6-sol` |

## 日常命令

macOS、Linux 和 WSL：

```sh
~/.aiare-setup/bin/aiare-setup doctor
~/.aiare-setup/bin/aiare-setup install
~/.aiare-setup/bin/aiare-setup uninstall
```

Windows PowerShell：

```powershell
& "$env:LOCALAPPDATA\AIARE\CLISetup\bin\aiare-setup.exe" doctor
& "$env:LOCALAPPDATA\AIARE\CLISetup\bin\aiare-setup.exe" install
& "$env:LOCALAPPDATA\AIARE\CLISetup\bin\aiare-setup.exe" uninstall
```

再次执行 `install` 可轮换密钥。只配置单个工具时使用 `--tools claude` 或 `--tools codex`。`doctor` 会检查 CLI、配置、凭据权限以及远端模型可用性。

卸载默认采用冲突保护：如果受管字段在安装后被手动修改，安装器会停止并说明冲突。确认要恢复安装前值时才使用 `uninstall --force`。用户的其他配置和首次安装备份都会保留。

## 环境变量冲突

Claude Code 的认证环境变量优先级可能高于配置文件。安装时若检测到 `ANTHROPIC_API_KEY`、`ANTHROPIC_AUTH_TOKEN`、Bedrock、Vertex 或 Foundry 相关变量，安装器会停止并列出冲突；请从当前终端及 shell 启动文件中移除后重试。

## 非交互与排障

在线脚本优先从 AIARE 域名下载，并在不可用时回退到 GitHub Release。两个来源都必须通过发行版 `CHECKSUMS.txt` 的 SHA-256 校验。

可用参数：

```sh
# 仅查看将执行的操作，不读取密钥、不写文件
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --dry-run

# 自动确认，并从标准输入读取密钥（仅建议在受控自动化环境使用）
printf '%s\n' "$AIARE_API_KEY" | ~/.aiare-setup/bin/aiare-setup install --yes --token-stdin
```

不要把真实密钥直接写进命令、脚本、工单或日志。安装器内部的 `credential print` 命令只供 Claude Code 和 Codex 的认证辅助机制调用。

## 开发与发布

本地要求 Go 1.22 或更高版本：

```sh
go test ./...
go build ./cmd/aiare-setup
```

推送 `setup-v*` 标签会由 GitHub Actions 测试并构建 Linux、macOS、Windows 的 amd64/arm64 发布包。Release 完成后，在 AIARE 服务器执行一次：

```sh
sudo ./deploy/publish.sh setup-v1.0.0
```

发布脚本会下载并校验全部资产，通过原子符号链接切换 `current`，校验 Nginx 后重载；失败时恢复原入口。详细发布步骤见 [docs/release.md](docs/release.md)。

旧版桌面应用保存在 Git 分支 `legacy/desktop-v3.19.1` 和标签 `aiare-desktop-v3.19.1`。

## 许可证

[MIT](LICENSE)
