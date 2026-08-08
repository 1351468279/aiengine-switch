# AiEngine Setup

AiEngine Setup 用一条在线命令，把已经安装在用户电脑上的 AI 客户端接入 AiEngine API 中转站。用户只需选择客户端、输入该客户端对应的 API 密钥；不需要安装常驻切换软件。

第一次接入请阅读：[Windows、macOS、Linux、WSL 保姆级教程](docs/user-guide.md)。Continue、Cline、Roo Code、Cherry Studio 等图形客户端，以及 Cursor 的有限接入说明，见[主流客户端手动接入](docs/mainstream-clients.md)。

## 支持范围

| 客户端 | 配置方式 | 默认/必需模型 | 系统 |
| --- | --- | --- | --- |
| Claude Code | 自动 | `claude-sonnet-5`、`claude-opus-5`、`claude-haiku-4-5-20251001` | Windows、macOS、Linux、WSL |
| Claude Desktop | 自动 | 上述 3 个 Claude 模型 | Windows、macOS |
| Codex | 自动 | `gpt-5.6-sol` | Windows、macOS、Linux、WSL |
| Hermes Agent | 自动，可用 `--model` | 默认 `claude-sonnet-5` | Windows、macOS、Linux、WSL；Hermes 自身支持范围以官方版本为准 |
| OpenCode | 自动，可用 `--model` | 默认 `claude-sonnet-5` | Windows、macOS、Linux、WSL |
| Aider | 自动，可用 `--model` | 默认 `claude-sonnet-5` | Windows、macOS、Linux、WSL |
| Continue、Cline、Roo Code、Cherry Studio | 手动 | 使用密钥有权限的准确模型 ID | 取决于客户端版本 |
| Cursor | 有限、按版本尝试，不属于正式支持 | 使用密钥有权限的标准聊天模型 | 取决于客户端是否提供 Base URL 覆盖 |

安装器只配置已经安装好的客户端，不负责安装客户端本身。每次只配置一个客户端，以免把 GPT 分组密钥误用于 Claude，或让不同客户端共享同一把不便吊销的密钥。

## 一键接入

macOS、Linux 或 WSL：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

Windows PowerShell：

```powershell
irm https://modelapi.aiaiaiaiai.cloud/install.ps1 | iex
```

如果只检测到一个支持的客户端，安装器会直接选择它；检测到多个时会显示选择菜单。确认后，密钥会在本机隐藏输入，并在写配置前验证对应模型权限。

建议明确指定客户端。以 GPT Pro 分组密钥配置 Hermes、OpenCode 或 Aider 时，还应明确指定该密钥可访问的模型：

macOS、Linux 或 WSL：

```sh
# Codex
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex

# Hermes，使用 GPT Pro 分组
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools hermes --model gpt-5.6-sol

# OpenCode，使用 GPT Pro 分组
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools opencode --model gpt-5.6-sol

# Aider，使用 GPT Pro 分组
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools aider --model gpt-5.6-sol
```

Windows PowerShell：

```powershell
# Codex
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools codex

# Hermes，使用 GPT Pro 分组
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools hermes -Model gpt-5.6-sol

# OpenCode，使用 GPT Pro 分组
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools opencode -Model gpt-5.6-sol

# Aider，使用 GPT Pro 分组
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools aider -Model gpt-5.6-sol
```

`--model`/`-Model` 必须填写 NewAPI 返回的准确模型 ID。Hermes、OpenCode 和 Aider 未指定时默认使用 `claude-sonnet-5`；它们只验证所选的一个模型。Claude Code、Claude Desktop 和 Codex 的模型由 AiEngine 固定，不接受 `--model`。

## 安装器会修改什么

| 客户端 | 受管配置 |
| --- | --- |
| Claude Code | `~/.claude/settings.json` 或 `CLAUDE_CONFIG_DIR`，以及独立凭据 |
| Claude Desktop | 独立 `AiEngine` 3P inference profile；保留其他 profile |
| Codex | `~/.codex/config.toml` 或 `CODEX_HOME`，以及独立凭据；不修改会话数据和 `auth.json` |
| Hermes | `~/.hermes/config.yaml` 与 `~/.hermes/.env`，支持 `HERMES_HOME` |
| OpenCode | `~/.config/opencode/opencode.json` 或 `OPENCODE_CONFIG`，以及独立凭据 |
| Aider | `~/.aider.conf.yml` 与 AiEngine 安装目录内的独立环境文件 |

安装器会在第一次修改前备份原文件，只合并自己管理的字段，并为凭据文件设置仅当前用户可读的权限。卸载时只恢复受管字段；若这些字段在安装后又被用户修改，默认停止并报告冲突。

Codex 的旧会话不会因安装而删除。安装器会尽量沿用已有的合法 provider ID（例如 `OpenAI`），并只读扫描 `CODEX_HOME/sessions` 中的 JSONL 元数据，避免旧版安装器写入 `aiengine` 后导致历史列表暂时不显示。遇到历史会话问题，请先完全退出 Codex，再重新运行最新安装器并执行 `doctor`；不要手动编辑 Codex 数据库。

## 日常命令

macOS、Linux 和 WSL：

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
~/.aiengine-setup/bin/aiengine-setup install --tools opencode --model gpt-5.6-sol
~/.aiengine-setup/bin/aiengine-setup uninstall --tools opencode
~/.aiengine-setup/bin/aiengine-setup uninstall --tools all
```

Windows PowerShell：

```powershell
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools opencode --model gpt-5.6-sol
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools opencode
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools all
```

再次执行同一客户端的 `install` 可轮换它的密钥或模型。`doctor` 会检查每个已配置客户端的程序、配置、凭据权限和远端模型权限。确认要覆盖安装后对受管字段所做的修改时，才使用 `uninstall --force`。

## 密钥与模型

- GPT Pro 号池密钥通常应配置 Codex，或为 Hermes、OpenCode、Aider 指定 `--model gpt-5.6-sol`。
- Claude Code 和 Claude Desktop 的密钥必须同时拥有表格中的 3 个 Claude 模型。
- Hermes、OpenCode 和 Aider 可以使用任意 AiEngine 密钥，但 `--model` 必须与该密钥实际权限一致。
- 不同客户端建议分别创建密钥，便于统计、限额和单独吊销。
- 不要把真实密钥直接写进命令、脚本、工单、截图或日志。

Claude Code 的认证环境变量可能覆盖配置文件。安装时若检测到 `ANTHROPIC_API_KEY`、`ANTHROPIC_AUTH_TOKEN`、Bedrock、Vertex 或 Foundry 变量，安装器会停止并列出冲突。

## 非交互与排障

在线脚本优先从 AiEngine 域名下载，不可用时回退到仓库的 `setup-assets` 备用分支。两个来源均须通过发行版 `CHECKSUMS.txt` 的 SHA-256 校验。

```sh
# 只查看操作计划，不读取密钥、不写文件
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools aider --model gpt-5.6-sol --dry-run

# 受控自动化中从标准输入读取密钥
printf '%s\n' "$AIENGINE_API_KEY" | ~/.aiengine-setup/bin/aiengine-setup install --tools aider --model gpt-5.6-sol --yes --token-stdin
```

安装器目录和可执行文件可通过 `AIENGINE_SETUP_HOME`、`AIENGINE_SETUP_BINARY` 覆盖。旧版的 `AIARE_SETUP_HOME`、`AIARE_SETUP_BINARY` 仅为已有安装兼容而保留。

## 开发与发布

要求 Go 1.22 或更高版本：

```sh
go test ./...
go build ./cmd/aiengine-setup
```

推送 `setup-v*` 标签会由 GitHub Actions 测试并构建 Linux、macOS、Windows 的 amd64/arm64 发布包。工作流也支持手动触发和 `release-v*` 恢复分支；不依赖 Actions 的 `setup-assets` 备用发布方式见 [发布说明](docs/release.md)。Release 或备用资产发布完成后，在 AiEngine 服务器执行：

```sh
sudo ./deploy/publish.sh setup-v1.4.0
```

详细步骤见 [docs/release.md](docs/release.md)。

## 许可证

[MIT](LICENSE)
