# AiEngine Setup

为 AiEngine API 中转站配置现有的 Claude Code、Claude Desktop 或 Codex。用户只需运行一条在线命令并输入自己的 API 密钥，不需要安装桌面切换工具。

第一次接入请阅读：[Windows、macOS、Linux、WSL 保姆级用户教程](docs/user-guide.md)。教程包含客户端安装、密钥选择、多客户端配置、验证和常见报错处理。

## 一键接入

开始前请先安装至少一个目标客户端，并在 AiEngine 控制台创建 API 密钥。Claude Desktop 接入仅支持 Windows 和 macOS。

macOS、Linux 或 WSL：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://modelapi.aiaiaiaiai.cloud/install.ps1 | iex
```

安装器每次只配置一个客户端。如果只检测到一个客户端，会直接选择它；检测到多个时会要求选择本次配置哪一个。确认后在本机隐藏输入密钥，并只验证该客户端需要的模型。Claude Desktop 还会对原生 `/v1/messages` 流式接口发起一次最小请求，确认网关确实可用。

不同客户端使用不同权限的密钥时，请为每个客户端分别运行一次安装命令。Claude Code、Claude Desktop 和 Codex 的配置与密钥会独立保存，互不覆盖。

## 安装器做什么

- Claude Code：合并修改 `~/.claude/settings.json`（或 `CLAUDE_CONFIG_DIR`），配置 AiEngine 地址、默认模型和 `apiKeyHelper`。
- Codex：合并修改 `~/.codex/config.toml`（或 `CODEX_HOME`），添加 `aiengine` Responses provider 和凭据命令。
- Claude Desktop：切换到 Claude 的 3P inference gateway 模式，创建独立的 `AiEngine` profile，并保留其他 profile。
- 密钥：三个客户端分别使用独立凭据。Claude Code 和 Codex 的密钥不进入客户端主配置；Claude Desktop 的协议要求密钥存在于其 3P profile 中，安装器会在 macOS 设置 `0600` 权限、在 Windows 收紧 ACL，且密钥不会进入安装状态。
- 备份：首次修改前保留原配置；卸载时只恢复本工具管理且未被用户再次修改的字段。
- Codex 登录：不会读取或修改 `~/.codex/auth.json`，官方登录状态会保留。

默认模型：

| 客户端 | 模型 |
| --- | --- |
| Claude Code | `claude-sonnet-5` |
| Claude Code Opus | `claude-opus-5` |
| Claude Code Haiku | `claude-haiku-4-5-20251001` |
| Claude Desktop | 上述 3 个 Claude 模型 |
| Codex | `gpt-5.6-sol` |

## 日常命令

macOS、Linux 和 WSL：

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
~/.aiengine-setup/bin/aiengine-setup install --tools codex
~/.aiengine-setup/bin/aiengine-setup install --tools claude
~/.aiengine-setup/bin/aiengine-setup install --tools claude-desktop # 仅 macOS
~/.aiengine-setup/bin/aiengine-setup uninstall --tools all
```

Windows PowerShell：

```powershell
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools codex
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools claude
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools claude-desktop
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools all
```

再次对同一客户端执行 `install` 可轮换它的密钥。安装不支持 `--tools all`，必须一次配置一个客户端；卸载仍可使用 `--tools all`。`doctor` 会分别检查每个客户端的应用或 CLI、配置、凭据权限和远端模型可用性。

卸载默认采用冲突保护：如果受管字段在安装后被手动修改，安装器会停止并说明冲突。确认要恢复安装前值时才使用 `uninstall --force`。用户的其他配置和首次安装备份都会保留。

## 环境变量冲突

Claude Code 的认证环境变量优先级可能高于配置文件。安装时若检测到 `ANTHROPIC_API_KEY`、`ANTHROPIC_AUTH_TOKEN`、Bedrock、Vertex 或 Foundry 相关变量，安装器会停止并列出冲突；请从当前终端及 shell 启动文件中移除后重试。

## 非交互与排障

在线脚本优先从 AiEngine 域名下载，并在不可用时回退到 GitHub Release。两个来源都必须通过发行版 `CHECKSUMS.txt` 的 SHA-256 校验。

安装器目录和可执行文件可分别通过 `AIENGINE_SETUP_HOME`、`AIENGINE_SETUP_BINARY` 覆盖。旧版使用的 `AIARE_SETUP_HOME`、`AIARE_SETUP_BINARY` 仅为兼容已有安装而保留，新部署请使用 AiEngine 字段。

可用参数：

```sh
# 仅查看将执行的操作，不读取密钥、不写文件
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --dry-run

# 在线安装时明确选择 Codex；Claude Code 使用 --tools claude
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex

# macOS 上配置 Claude Desktop；运行前先完全退出 Claude Desktop
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude-desktop

# 自动确认，并从标准输入读取密钥（仅建议在受控自动化环境使用）
printf '%s\n' "$AIENGINE_API_KEY" | ~/.aiengine-setup/bin/aiengine-setup install --tools codex --yes --token-stdin
```

PowerShell 在线安装也可明确选择客户端：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools codex

# Windows 上配置 Claude Desktop；运行前先完全退出 Claude Desktop
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools claude-desktop
```

不要把真实密钥直接写进命令、脚本、工单或日志。安装器内部的 `credential print` 命令只供 Claude Code 和 Codex 的认证辅助机制调用。

## 开发与发布

本地要求 Go 1.22 或更高版本：

```sh
go test ./...
go build ./cmd/aiengine-setup
```

推送 `setup-v*` 标签会由 GitHub Actions 测试并构建 Linux、macOS、Windows 的 amd64/arm64 发布包。Release 完成后，在 AiEngine 服务器执行一次：

```sh
sudo ./deploy/publish.sh setup-v1.3.0
```

发布脚本会下载并校验全部资产，通过原子符号链接切换 `current`，校验 Nginx 后重载；失败时恢复原入口。详细发布步骤见 [docs/release.md](docs/release.md)。

已有旧版 CLI 安装会继续使用原安装目录，并在再次运行安装器时迁移受管的 Codex provider；无需手动清理旧配置。

## 许可证

[MIT](LICENSE)
