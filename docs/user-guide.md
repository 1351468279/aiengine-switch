# AiEngine API 接入教程

这份教程面向第一次使用命令行的用户，适用于 Windows、macOS、Linux 和 WSL。AiEngine 安装器负责配置已经安装好的客户端，不会替你安装客户端。

自动配置支持 Claude Code、Claude Desktop、Codex、Hermes Agent、OpenCode 和 Aider。Continue、Cline、Roo Code、Cherry Studio 等图形客户端，以及 Cursor 的有限接入说明，请看[主流客户端手动接入](mainstream-clients.md)。

## 一、开始前的准备

### 1. 选择客户端和密钥

| 客户端 | AiEngine 密钥要求 | 自动配置参数 |
| --- | --- | --- |
| Codex | 能访问 `gpt-5.6-sol` | `codex` |
| Claude Code | 同时拥有 3 个 Claude 模型 | `claude` |
| Claude Desktop | 同时拥有 3 个 Claude 模型和原生 Messages 接口 | `claude-desktop` |
| Hermes Agent | 能访问你通过 `--model` 指定的模型 | `hermes` |
| OpenCode | 能访问你通过 `--model` 指定的模型 | `opencode` |
| Aider | 能访问你通过 `--model` 指定的模型 | `aider` |

Claude Code 和 Claude Desktop 需要以下 3 个模型：

- `claude-sonnet-5`
- `claude-opus-5`
- `claude-haiku-4-5-20251001`

GPT Pro 号池分组密钥通常只能访问 GPT 模型。这类密钥可直接配置 Codex；配置 Hermes、OpenCode 或 Aider 时，指定 `gpt-5.6-sol`。不要用它配置 Claude Code 或 Claude Desktop，应在 AiEngine 控制台另外创建具有 Claude 模型权限的密钥。

不同客户端建议使用不同密钥，便于分别统计、限额和吊销。安装器要求输入密钥时，终端不会显示字符或星号，这是正常的安全设计。

### 2. 理解 `--model`

Hermes、OpenCode 和 Aider 可以使用任意 AiEngine 模型，因此需要告诉安装器这把密钥具体使用哪个模型。

```text
GPT Pro 分组示例: gpt-5.6-sol
Claude 分组示例:  claude-sonnet-5
```

模型 ID 必须与 AiEngine 控制台或 `/v1/models` 返回的内容完全一致。省略 `--model` 时，这 3 个客户端默认使用 `claude-sonnet-5`。Claude Code、Claude Desktop、Codex 使用固定模型规则，不要给它们添加 `--model`。

### 3. 选择正确的终端

| 系统 | 使用的终端 | AiEngine 脚本 |
| --- | --- | --- |
| Windows 原生 | Windows Terminal 中的 PowerShell | `install.ps1` |
| macOS | Terminal/终端 | `install.sh` |
| Linux | Terminal | `install.sh` |
| Windows 中的 WSL2 | Ubuntu 等 WSL 终端 | `install.sh` |

Windows 提示符以 `PS C:\...>` 开头才是 PowerShell。若是 `C:\...>`，说明当前是 CMD，不要直接运行 PowerShell 命令。

## 二、Windows 原生教程

以下命令在普通 PowerShell 中运行，不要求管理员权限。OpenCode 官方建议 Windows 用户优先使用 WSL；若你把 OpenCode 安装在 WSL 中，请跳到 WSL 章节，不要混用两套配置。

### 第 1 步：安装客户端

只执行你需要的客户端安装命令。

Codex：

```powershell
irm https://chatgpt.com/codex/install.ps1 | iex
```

Claude Code：

```powershell
irm https://claude.ai/install.ps1 | iex
```

Hermes Agent：

```powershell
iex (irm https://hermes-agent.nousresearch.com/install.ps1)
```

OpenCode 原生安装需要先安装 Node.js，然后运行：

```powershell
npm install -g opencode-ai
```

Aider：

```powershell
powershell -ExecutionPolicy ByPass -c "irm https://aider.chat/install.ps1 | iex"
```

Claude Desktop 请从 [Claude 官方下载页](https://claude.com/download) 安装。第一次打开确认程序可运行后，从任务栏托盘完全退出；只关闭窗口可能仍在后台运行。

安装命令行客户端后，关闭 PowerShell，再打开一个新窗口，使系统重新读取命令路径。

### 第 2 步：确认安装成功

使用哪个客户端就检查哪一行：

```powershell
codex --version
claude --version
hermes --version
opencode --version
aider --version
```

看到版本号即可继续。若提示“无法识别”或“不是 cmdlet”，说明客户端未正确安装，或者需要重新打开 PowerShell。Claude Desktop 没有这一步，确认应用已安装即可。

### 第 3 步：运行 AiEngine 配置

只配置 Codex：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools codex
```

只配置 Claude Code：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools claude
```

只配置 Claude Desktop：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools claude-desktop
```

使用 GPT Pro 密钥配置 Hermes：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools hermes -Model gpt-5.6-sol
```

使用 GPT Pro 密钥配置 OpenCode：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools opencode -Model gpt-5.6-sol
```

使用 GPT Pro 密钥配置 Aider：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools aider -Model gpt-5.6-sol
```

若电脑只安装了一个受支持的客户端，也可让安装器自动识别：

```powershell
irm https://modelapi.aiaiaiaiai.cloud/install.ps1 | iex
```

### 第 4 步：输入密钥

安装器会显示客户端、模型、API 地址和安装目录，然后询问：

```text
继续吗？[Y/n]
```

输入 `y` 并按回车。出现“请输入 AiEngine API 密钥”后粘贴密钥，再按回车。粘贴期间屏幕没有变化是正常的。安装器会先检查密钥和模型权限，通过后才写入配置。

### 第 5 步：检查并使用

```powershell
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
```

出现“诊断通过”后，按需启动：

```powershell
codex
claude
hermes
opencode
aider
```

Claude Desktop 用户重新打开应用，在模型菜单中选择 AiEngine 提供的模型。

## 三、macOS 教程

AiEngine 安装器本身支持 Apple 芯片和 Intel 芯片。各客户端的系统要求可能不同，例如 Hermes 官方当前主要支持 Apple Silicon；请以对应客户端的官方安装结果为准。

### 第 1 步：打开终端

按 `Command + 空格`，搜索 `Terminal` 或“终端”，然后打开。

### 第 2 步：安装客户端

只执行需要的命令：

```sh
# Codex
curl -fsSL https://chatgpt.com/codex/install.sh | sh

# Claude Code
curl -fsSL https://claude.ai/install.sh | bash

# Hermes Agent
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash

# OpenCode
curl -fsSL https://opencode.ai/install | bash

# Aider
curl -LsSf https://aider.chat/install.sh | sh
```

Claude Desktop 请从 [Claude 官方下载页](https://claude.com/download) 安装到“应用程序”。配置前使用 `Command + Q` 完全退出 Claude。

关闭并重新打开终端，再用对应的 `--version` 命令确认 CLI 已安装。

### 第 3 步：接入 AiEngine

```sh
# Codex
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex

# Claude Code
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude

# Claude Desktop，仅 macOS
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude-desktop

# Hermes，GPT Pro 密钥示例
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools hermes --model gpt-5.6-sol

# OpenCode，GPT Pro 密钥示例
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools opencode --model gpt-5.6-sol

# Aider，GPT Pro 密钥示例
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools aider --model gpt-5.6-sol
```

每次只执行本次要配置的一行，并输入对应密钥。若用 Claude 分组密钥配置后三个客户端，可把模型改成 `claude-sonnet-5`，或直接省略 `--model`。

### 第 4 步：检查并启动

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
```

诊断通过后运行对应客户端命令。Claude Desktop 用户重新打开应用。

## 四、Linux 教程

AiEngine 安装器支持常见的 64 位 x86 和 ARM Linux。Claude Desktop 的 AiEngine 3P 配置不支持 Linux，其他 5 个 CLI 客户端可按本节操作。

### 第 1 步：确认下载工具

```sh
curl --version
```

若系统没有 `curl`，先安装：

```sh
# Ubuntu / Debian
sudo apt update && sudo apt install -y curl

# Fedora
sudo dnf install -y curl

# Arch Linux
sudo pacman -S curl
```

### 第 2 步：安装客户端

```sh
# Codex
curl -fsSL https://chatgpt.com/codex/install.sh | sh

# Claude Code
curl -fsSL https://claude.ai/install.sh | bash

# Hermes Agent
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash

# OpenCode
curl -fsSL https://opencode.ai/install | bash

# Aider
curl -LsSf https://aider.chat/install.sh | sh
```

只执行需要的命令。完成后重新打开终端，并运行相应的 `--version` 命令。

### 第 3 步：接入 AiEngine

```sh
# 固定模型客户端
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude

# 可选模型客户端，以下使用 GPT Pro 密钥示例
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools hermes --model gpt-5.6-sol
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools opencode --model gpt-5.6-sol
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools aider --model gpt-5.6-sol
```

每次只执行需要配置的一行。按提示确认并粘贴对应密钥。

### 第 4 步：检查并启动

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
```

诊断通过后，运行 `codex`、`claude`、`hermes`、`opencode` 或 `aider`。

## 五、Windows WSL2 教程

WSL 是 Windows 内的独立 Linux 环境。客户端安装在 WSL 中时，配置文件和 Windows 原生环境完全分开，必须在 WSL 终端运行 `install.sh`。

### 第 1 步：确认 WSL2

在管理员 PowerShell 中安装：

```powershell
wsl --install
```

已有 WSL 时检查：

```powershell
wsl -l -v
```

目标发行版的 `VERSION` 应为 `2`。然后从开始菜单打开 Ubuntu，并确认：

```sh
echo $WSL_DISTRO_NAME
```

### 第 2 步：安装客户端

在 WSL 终端中执行对应命令：

```sh
curl -fsSL https://chatgpt.com/codex/install.sh | sh
curl -fsSL https://claude.ai/install.sh | bash
curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
curl -fsSL https://opencode.ai/install | bash
curl -LsSf https://aider.chat/install.sh | sh
```

只运行需要的行。重新打开 WSL 终端后确认客户端版本。

### 第 3 步：接入并检查

使用的 AiEngine 命令与 Linux 完全相同。例如：

```sh
# OpenCode + GPT Pro 密钥
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools opencode --model gpt-5.6-sol

~/.aiengine-setup/bin/aiengine-setup doctor
opencode
```

不要在 Windows PowerShell 运行 `install.ps1` 来配置 WSL 内的客户端。Claude Desktop 也不能在 WSL 中使用 `--tools claude-desktop`。

## 六、配置多个客户端

AiEngine 安装器每次只配置一个客户端。推荐流程：

1. 在 AiEngine 控制台为每个客户端创建独立密钥。
2. 逐条运行对应的配置命令。
3. Hermes、OpenCode、Aider 每次都填写与密钥权限一致的模型。
4. 最后运行一次 `doctor`，统一检查全部已配置客户端。

Windows 示例：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools codex
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools opencode -Model gpt-5.6-sol
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools aider -Model claude-sonnet-5
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
```

macOS、Linux、WSL 示例：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools opencode --model gpt-5.6-sol
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools aider --model claude-sonnet-5
~/.aiengine-setup/bin/aiengine-setup doctor
```

每次命令都会单独提示输入密钥。不要把第一把密钥直接用于所有客户端。

## 七、更换密钥或模型

重新执行对应客户端的安装命令即可，不需要先卸载。

```sh
# macOS/Linux/WSL：同时更换 OpenCode 的密钥和模型
~/.aiengine-setup/bin/aiengine-setup install --tools opencode --model claude-sonnet-5
```

```powershell
# Windows：同时更换 OpenCode 的密钥和模型
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools opencode --model claude-sonnet-5
```

安装器会先验证新密钥，只更新当前客户端。其他客户端的密钥和配置不受影响。

## 八、卸载 AiEngine 配置

卸载不会删除客户端，只恢复 AiEngine 安装器管理的字段。

Windows：

```powershell
# 全部卸载
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools all

# 只卸载 Hermes
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools hermes
```

macOS、Linux 或 WSL：

```sh
# 全部卸载
~/.aiengine-setup/bin/aiengine-setup uninstall --tools all

# 只卸载 Hermes
~/.aiengine-setup/bin/aiengine-setup uninstall --tools hermes
```

单独卸载时可使用 `claude`、`claude-desktop`、`codex`、`hermes`、`opencode` 或 `aider`。如果安装后手动修改了同一受管字段，卸载器会停止保护修改；确认覆盖时才添加 `--force`。

## 九、常见问题

### 1. `curl: (6) Could not resolve host: sh`

命令缺少管道符 `|`。正确命令是：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

### 2. Windows 出现乱码或 `﻿[CmdletBinding()]` 解析错误

关闭旧窗口，使用 Windows Terminal 中的 PowerShell，并刷新脚本缓存：

```powershell
irm ("https://modelapi.aiaiaiaiai.cloud/install.ps1?t=" + (Get-Date).Ticks) | iex
```

不要使用 PowerShell ISE，也不要把 PowerShell 命令放进 CMD。

### 3. 输入 `y` 后立刻出现 `Access is denied`

这通常来自旧版缓存脚本或不兼容终端。重新打开普通 PowerShell，然后运行上面的动态缓存刷新命令。

### 4. 输入密钥时屏幕没有显示

正常。粘贴一次完整密钥并按回车，不要因看不到字符而重复粘贴。

### 5. 提示“该密钥缺少所需模型”

当前密钥与客户端或 `--model` 不匹配：

- Codex 需要 `gpt-5.6-sol`。
- Claude Code 和 Claude Desktop 需要本教程开头列出的 3 个 Claude 模型。
- Hermes、OpenCode、Aider 只检查指定模型；确认模型 ID 拼写和密钥分组。

### 6. 提示“未检测到客户端”

安装器不会安装客户端。先运行对应的 `--version` 命令。客户端已安装但自动检测失败时，明确使用 `--tools` 指定；若仍提示未检测到，说明该 CLI 不在当前终端的 `PATH` 中。

Claude Desktop 没有版本命令。Windows/macOS 用户可完全退出应用后明确指定 `--tools claude-desktop`。

### 7. Claude Code 提示环境变量冲突

这些变量可能覆盖 AiEngine 配置：

- `ANTHROPIC_API_KEY`
- `ANTHROPIC_AUTH_TOKEN`
- `CLAUDE_CODE_USE_BEDROCK`
- `CLAUDE_CODE_USE_FOUNDRY`
- `CLAUDE_CODE_USE_VERTEX`

macOS、Linux、WSL 当前终端清理命令：

```sh
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN CLAUDE_CODE_USE_BEDROCK CLAUDE_CODE_USE_FOUNDRY CLAUDE_CODE_USE_VERTEX
```

Windows 当前 PowerShell 清理命令：

```powershell
Remove-Item Env:ANTHROPIC_API_KEY -ErrorAction SilentlyContinue
Remove-Item Env:ANTHROPIC_AUTH_TOKEN -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_BEDROCK -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_FOUNDRY -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_VERTEX -ErrorAction SilentlyContinue
```

如果新终端中变量再次出现，还需从 shell 启动文件或 Windows 用户环境变量中移除。

### 8. `doctor` 报 API 验证失败

常见原因是密钥被删除、额度不足、模型权限改变，或网络无法连接 AiEngine。先在 AiEngine 控制台确认密钥和分组，再重新执行该客户端的 `install --tools ...`。

### 9. OpenCode 配置文件无法解析

安装器读取的是标准 JSON。若现有 `opencode.json` 包含 JSONC 注释或尾随逗号，安装器会停止而不覆盖文件。先备份并改成有效 JSON，再重试。

### 10. Claude Desktop 没有出现 AiEngine 模型

先完全退出再重新打开 Claude Desktop，然后运行 `doctor`。若诊断提示 `/v1/messages`、流式 SSE 或模型权限失败，说明密钥或上游分组不满足桌面版原生协议要求。

### 11. 下载失败或域名无法解析

先在浏览器测试 `https://modelapi.aiaiaiaiai.cloud/install.sh`。在线脚本会先用 AiEngine 下载源，失败后尝试 GitHub Release，并对两者执行 SHA-256 校验。

## 十、安全和配置说明

- Claude Code、Codex、OpenCode 的密钥不写入客户端主配置，而是引用 AiEngine 独立凭据。
- Hermes 与 Aider 按客户端官方配置方式使用独立 `.env` 文件；安装器将其权限限制为当前用户。
- Claude Desktop 协议要求密钥保存在其 AiEngine profile；安装器会在 macOS 设置 `0600`，在 Windows 收紧 ACL。
- 安装状态不保存明文密钥，只保存必要的文件状态与哈希。
- 修改客户端配置前会保留备份，只合并 AiEngine 需要的字段。
- Codex 的官方登录文件 `~/.codex/auth.json` 不会被读取、替换或删除。
- 下载的安装包通过发行版 SHA-256 校验后才会运行。

寻求帮助时可以提供系统、终端、客户端名称、模型 ID、完整错误文字和 `doctor` 输出，但不要提供或截图 API 密钥。
