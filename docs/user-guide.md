# AiEngine API 接入教程

这份教程面向第一次使用命令行的用户，适用于 Windows、macOS、Linux 和 WSL。完成后，你可以让 Claude Code、Claude Desktop 或 Codex 通过 AiEngine API 工作。

> AiEngine 安装器只负责配置已经安装好的客户端，不会替你安装 Claude Code、Claude Desktop 或 Codex。Claude Desktop 的 AiEngine 接入目前仅支持 Windows 和 macOS。

## 一、先确认你要使用什么

### 1. 选择客户端

| 你的需求 | 需要安装的客户端 | AiEngine 密钥要求 |
| --- | --- | --- |
| 只用 Codex | Codex CLI | 密钥必须能访问 `gpt-5.6-sol` |
| 用命令行版 Claude | Claude Code | 密钥必须能访问下方列出的 3 个 Claude 模型 |
| 用桌面版 Claude | Claude Desktop（Windows/macOS） | 密钥必须能访问下方列出的 3 个 Claude 模型和原生 Messages 接口 |
| 使用多个客户端 | 对应客户端 | 建议每个客户端准备一把密钥，分别配置 |

Claude Code 密钥需要同时能访问：

- `claude-sonnet-5`
- `claude-opus-5`
- `claude-haiku-4-5-20251001`

如果你的 AiEngine 密钥属于 GPT Pro 号池分组，它通常只能用于 Codex，不能用于 Claude。给 Claude 使用时，请在 AiEngine 控制台另外创建一把具有上述 Claude 模型权限的密钥。

### 2. 准备 AiEngine 密钥

1. 登录 AiEngine 控制台。
2. 打开令牌或 API 密钥页面。
3. 按准备使用的客户端选择正确的模型分组并创建密钥。
4. 暂时保留密钥页面，安装过程中再粘贴。

不要把密钥发给其他人，也不要把密钥直接写进命令、聊天记录、截图或工单。安装器要求输入密钥时，屏幕上不会显示字符或星号，这是正常的安全设计。

### 3. 根据系统选择终端

| 系统 | 应该打开什么 | AiEngine 安装脚本 |
| --- | --- | --- |
| Windows 原生 | Windows Terminal 中的 PowerShell，或 Windows PowerShell | `install.ps1` |
| macOS | 终端 Terminal | `install.sh` |
| Linux | Terminal | `install.sh` |
| Windows 中的 WSL | Ubuntu 等 WSL Linux 终端 | `install.sh` |

Windows 用户特别注意：

- 提示符是 `PS C:\Users\你的名字>`，说明你在 PowerShell，可以继续。
- 提示符是 `C:\Users\你的名字>`，说明你在 CMD，请先打开 PowerShell。
- 提示符类似 `user@computer:~$`，说明你在 WSL，应按照 WSL 教程操作。

## 二、Windows 教程

以下步骤适用于直接在 Windows 中运行 Claude Code、Claude Desktop 或 Codex 的用户。全程不需要使用管理员身份。

### 第 1 步：打开 PowerShell

1. 点击开始菜单。
2. 搜索 `Terminal` 或 `PowerShell`。
3. 打开后确认左侧提示符以 `PS` 开头。

不要在 CMD 中运行本节命令，也不建议使用 PowerShell ISE。

### 第 2 步：安装你需要的客户端

只使用 Codex，运行：

```powershell
irm https://chatgpt.com/codex/install.ps1 | iex
```

只使用 Claude Code，运行：

```powershell
irm https://claude.ai/install.ps1 | iex
```

两个命令行客户端都需要，就依次运行上面两条命令。安装结束后关闭当前 PowerShell，再打开一个新的 PowerShell，让系统重新读取命令路径。

使用 Claude Desktop 时，到 [Claude 官方下载页](https://claude.com/download) 下载并安装 Windows 版本。安装后先打开一次，确认应用可以启动，然后从任务栏托盘完全退出 Claude Desktop；只关闭窗口可能仍会让应用留在后台。

这两条是命令行客户端的官方安装命令，可分别查看 [Codex CLI 官方文档](https://learn.chatgpt.com/docs/codex/cli) 和 [Claude Code 官方安装文档](https://code.claude.com/docs/en/installation)。

### 第 3 步：确认客户端安装成功

使用 Codex 时运行：

```powershell
codex --version
```

使用 Claude Code 时运行：

```powershell
claude --version
```

看到版本号就可以继续。如果提示“无法识别”或“不是 cmdlet”，先关闭并重新打开 PowerShell；仍然失败时，说明对应客户端还没有正确安装。

Claude Desktop 没有需要检查的命令。能从开始菜单打开 Claude，且安装器稍后显示“已检测到 Claude Desktop”，就可以继续。配置前请再次完全退出应用。

### 第 4 步：接入 AiEngine

如果电脑上只安装了一个客户端，直接运行：

```powershell
irm https://modelapi.aiaiaiaiai.cloud/install.ps1 | iex
```

如果安装了多个客户端，或者你想明确指定本次配置哪一个，使用下面的对应命令。

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

### 第 5 步：按提示完成安装

安装器会显示本次配置的客户端、API 地址和安装目录，然后询问：

```text
继续吗？[Y/n]
```

1. 输入 `y` 并按回车。直接按回车也表示继续。
2. 出现“请输入 AiEngine API 密钥”后，粘贴这次所选客户端对应的密钥。
3. 粘贴时屏幕不会显示任何字符，这是正常现象。
4. 按回车，等待密钥和模型权限验证。Claude Desktop 会额外发送一次最多生成 1 token 的流式测试请求。

看到权限验证通过，并出现 `doctor` 检查命令，就表示配置成功。Claude Desktop 还会多显示一行重新打开应用的提示：

```text
API 密钥和模型权限验证通过。
运行 ... doctor 可检查接入状态。
```

### 第 6 步：检查接入状态

运行：

```powershell
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
```

最后出现“诊断通过”就表示检查通过，括号里会显示警告数量：

```text
诊断通过（N 个警告）。
```

### 第 7 步：开始使用

使用 Codex：

```powershell
codex
```

使用 Claude Code：

```powershell
claude
```

使用 Claude Desktop 时，重新从开始菜单打开 Claude。应用会读取 `AiEngine` 3P profile；在模型菜单中选择 AiEngine 提供的 Claude 模型后即可开始对话。

建议先进入一个自己的项目目录，再启动客户端。例如：

```powershell
cd C:\Users\你的用户名\Documents\你的项目
codex
```

## 三、macOS 教程

Apple 芯片和 Intel 芯片都会由脚本自动识别，无需手动选择安装包。

### 第 1 步：打开终端

按 `Command + 空格`，搜索 `Terminal` 或“终端”，然后打开。

### 第 2 步：安装你需要的客户端

只使用 Codex，运行：

```sh
curl -fsSL https://chatgpt.com/codex/install.sh | sh
```

只使用 Claude Code，运行：

```sh
curl -fsSL https://claude.ai/install.sh | bash
```

两个都需要，就依次运行上面两条命令。完成后关闭终端，再打开一个新的终端。

使用 Claude Desktop 时，到 [Claude 官方下载页](https://claude.com/download) 下载 macOS 版本，安装到“应用程序”并打开一次。配置前按 `Command + Q` 完全退出 Claude；只点窗口左上角红色按钮并不会退出应用。

### 第 3 步：确认客户端安装成功

```sh
# 使用哪个客户端，就检查哪个；两个都用就执行两行
codex --version
claude --version
```

看到版本号后继续。

### 第 4 步：接入 AiEngine

只配置 Codex：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
```

只配置 Claude Code：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude
```

只配置 Claude Desktop：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude-desktop
```

如果电脑上只安装了一个客户端，也可以使用自动识别命令：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

出现提示后，输入 `y` 并回车，再粘贴对应的 AiEngine 密钥并回车。密钥输入过程没有任何屏幕显示是正常的。

### 第 5 步：检查并启动

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
```

诊断通过后，按需启动：

```sh
codex
```

或：

```sh
claude
```

Claude Desktop 用户重新打开 Claude，在模型菜单中选择 AiEngine 提供的 Claude 模型后即可开始对话。

## 四、Linux 教程

支持常见的 64 位 x86 和 ARM Linux。安装器会自动识别 `x86_64/amd64` 或 `arm64/aarch64`。

本节只配置 Claude Code 或 Codex CLI。AiEngine 的 Claude Desktop 3P 配置暂不支持 Linux；即使系统中安装了 Claude Desktop，也不要在 Linux 或 WSL 使用 `--tools claude-desktop`。

### 第 1 步：打开终端并确认下载工具

先运行：

```sh
curl --version
```

如果能看到版本信息，直接进行下一步。如果提示找不到 `curl`，可根据发行版安装：

```sh
# Ubuntu / Debian
sudo apt update && sudo apt install -y curl

# Fedora
sudo dnf install -y curl

# Arch Linux
sudo pacman -S curl
```

### 第 2 步：安装你需要的官方客户端

Codex：

```sh
curl -fsSL https://chatgpt.com/codex/install.sh | sh
```

Claude Code：

```sh
curl -fsSL https://claude.ai/install.sh | bash
```

只运行你需要的命令。两个客户端都需要时，两条都运行。完成后关闭终端，再打开一个新终端。

### 第 3 步：确认客户端安装成功

```sh
codex --version
claude --version
```

只需要检查你准备使用的客户端。

### 第 4 步：接入 AiEngine

只配置 Codex：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
```

只配置 Claude Code：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude
```

按提示确认并粘贴对应密钥。输入密钥时终端不显示字符是正常的。

### 第 5 步：检查并启动

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
```

诊断通过后运行 `codex` 或 `claude`。

## 五、Windows WSL 教程

WSL 是运行在 Windows 里的 Linux 环境。只有当 Claude Code 或 Codex 安装在 WSL 内时，才按照本节操作。不要用 Windows 的 `install.ps1` 去配置安装在 WSL 里的客户端。

Codex 建议使用 WSL2；新版本不再支持 WSL1。

### 第 1 步：安装或确认 WSL2

还没有 WSL 时，以管理员身份打开 PowerShell，运行：

```powershell
wsl --install
```

按系统提示重启电脑。安装完成后，从开始菜单打开 Ubuntu。

已经安装 WSL 时，可在 PowerShell 中检查版本：

```powershell
wsl -l -v
```

`VERSION` 一列应为 `2`。

### 第 2 步：确认你已经进入 WSL

在 Ubuntu/WSL 终端运行：

```sh
echo $WSL_DISTRO_NAME
```

如果输出 `Ubuntu` 或其他 Linux 发行版名称，说明位置正确。后续所有命令都在这个 WSL 终端中执行，不要切回 PowerShell。

### 第 3 步：安装官方客户端

Codex：

```sh
curl -fsSL https://chatgpt.com/codex/install.sh | sh
```

Claude Code：

```sh
curl -fsSL https://claude.ai/install.sh | bash
```

完成后关闭 WSL 窗口，再重新打开 Ubuntu，然后执行 `codex --version` 或 `claude --version` 检查。

### 第 4 步：接入 AiEngine

只配置 Codex：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
```

只配置 Claude Code：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude
```

### 第 5 步：检查并启动

```sh
~/.aiengine-setup/bin/aiengine-setup doctor
```

诊断通过后，在 WSL 终端内运行 `codex` 或 `claude`。

## 六、配置多个客户端

AiEngine 安装器每次只配置一个客户端。这是为了让不同模型分组和不同应用使用各自的密钥，避免把 GPT 专用密钥错误地用于 Claude。Claude Code 与 Claude Desktop 可以使用同一把有权限的 Claude 密钥，但为了分别统计和单独吊销，仍建议创建两把。

推荐顺序：

1. 先运行 Codex 配置命令，输入 GPT Pro 或其他能访问 `gpt-5.6-sol` 的密钥。
2. 配置完成后，再按需运行 Claude Code 或 Claude Desktop 配置命令。
3. 每次输入当前客户端对应的密钥。
4. 最后只需要运行一次 `doctor`，它会分别检查所有已配置客户端。

Windows PowerShell：

```powershell
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools codex
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools claude
& ([scriptblock]::Create((irm https://modelapi.aiaiaiaiai.cloud/install.ps1))) -Tools claude-desktop
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
```

macOS、Linux 或 WSL：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools codex
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude
# claude-desktop 这一行仅在 macOS 执行
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh -s -- --tools claude-desktop
~/.aiengine-setup/bin/aiengine-setup doctor
```

两次运行时不要混淆密钥。出现“该密钥缺少所需模型”不是程序故障，而是当前输入的密钥不属于所选客户端需要的模型分组。

## 七、日后更换密钥

重新执行对应客户端的安装命令即可更换密钥，不需要先卸载。安装器会重新验证新密钥，并且只更新这一个客户端的凭据。

Windows：

```powershell
# 更换 Codex 密钥
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools codex

# 更换 Claude 密钥
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools claude

# 更换 Claude Desktop 密钥
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" install --tools claude-desktop
```

macOS、Linux 或 WSL：

```sh
# 更换 Codex 密钥
~/.aiengine-setup/bin/aiengine-setup install --tools codex

# 更换 Claude 密钥
~/.aiengine-setup/bin/aiengine-setup install --tools claude

# 更换 Claude Desktop 密钥（仅 macOS）
~/.aiengine-setup/bin/aiengine-setup install --tools claude-desktop
```

## 八、卸载 AiEngine 配置

卸载只恢复 AiEngine 安装器管理的配置，不会卸载任何客户端，也不会删除 Codex 的官方登录文件。Claude Desktop 的其他 profile 和安装后新增的 profile 会保留。

Windows：

```powershell
# 卸载全部 AiEngine 配置
& "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" uninstall --tools all
```

macOS、Linux 或 WSL：

```sh
# 卸载全部 AiEngine 配置
~/.aiengine-setup/bin/aiengine-setup uninstall --tools all
```

只卸载一个客户端时，把 `all` 改为 `codex`、`claude` 或 `claude-desktop`。

如果安装之后手动修改过同一批配置字段，卸载器会停止以保护你的修改。确认要恢复安装前状态时，才在命令末尾添加 `--force`。

## 九、常见问题

### 1. `curl: (6) Could not resolve host: sh`

原因通常是把命令写成了：

```text
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh sh
```

网址和 `sh` 中间缺少管道符 `|`。正确命令是：

```sh
curl -fsSL https://modelapi.aiaiaiaiai.cloud/install.sh | sh
```

### 2. 命令只显示了一整页脚本，没有开始安装

也是因为命令末尾缺少 `| sh` 或 `| iex`。请不要手动输入，完整复制对应系统的整行命令。

### 3. Windows 出现乱码，或报错位置带有 `﻿[CmdletBinding()]`

先关闭当前窗口，使用 Windows Terminal 中的 PowerShell 重新运行最新脚本。若网络缓存仍返回旧脚本，可运行带动态缓存刷新参数的命令：

```powershell
irm ("https://modelapi.aiaiaiaiai.cloud/install.ps1?t=" + (Get-Date).Ticks) | iex
```

### 4. 输入 `y` 后，还没来得及输入密钥就出现 `Access is denied`

这是旧版 Windows 脚本或不兼容终端常见的问题。请关闭 PowerShell ISE，打开普通的 Windows Terminal/PowerShell，然后使用上面的带缓存刷新参数命令重试。

### 5. 输入密钥时没有显示任何内容

这是正常现象。直接粘贴完整密钥并按回车即可。不要因为看不到字符而重复粘贴。

### 6. 提示“该密钥缺少所需模型”

当前密钥和所选客户端不匹配：

- 配置 Codex 时，密钥必须能访问 `gpt-5.6-sol`。
- 配置 Claude 时，密钥必须能访问本教程开头列出的 3 个 Claude 模型。
- GPT Pro 号池分组密钥不要用于 Claude；请另建 Claude 分组密钥后重试。

### 7. 提示“未检测到 Claude Code、Claude Desktop 或 Codex”

AiEngine 脚本不会安装客户端。先按照对应系统章节安装官方客户端，关闭并重新打开终端，然后运行：

```text
codex --version
claude --version
```

至少有一个命令能输出版本号后，再运行 AiEngine 安装命令。

Claude Desktop 没有版本检查命令。Windows/macOS 用户确认应用已安装后，可以使用明确指定的 `--tools claude-desktop` 在线命令，不依赖自动检测。

### 8. 配置 Claude 时提示环境变量冲突

以下变量会优先于 AiEngine 写入的 Claude 配置，因此安装器会主动停止：

- `ANTHROPIC_API_KEY`
- `ANTHROPIC_AUTH_TOKEN`
- `CLAUDE_CODE_USE_BEDROCK`
- `CLAUDE_CODE_USE_FOUNDRY`
- `CLAUDE_CODE_USE_VERTEX`

只清理当前 macOS、Linux 或 WSL 终端：

```sh
unset ANTHROPIC_API_KEY ANTHROPIC_AUTH_TOKEN CLAUDE_CODE_USE_BEDROCK CLAUDE_CODE_USE_FOUNDRY CLAUDE_CODE_USE_VERTEX
```

只清理当前 Windows PowerShell：

```powershell
Remove-Item Env:ANTHROPIC_API_KEY -ErrorAction SilentlyContinue
Remove-Item Env:ANTHROPIC_AUTH_TOKEN -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_BEDROCK -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_FOUNDRY -ErrorAction SilentlyContinue
Remove-Item Env:CLAUDE_CODE_USE_VERTEX -ErrorAction SilentlyContinue
```

如果打开新终端后变量再次出现，还需要从 shell 启动文件或 Windows 用户环境变量中删除，再重新打开终端。

### 9. `doctor` 报 API 验证失败

常见原因是密钥已被删除、额度不足、模型权限发生变化，或者当前网络无法连接 AiEngine。先在 AiEngine 控制台确认密钥和分组，再重新执行对应客户端的 `install --tools ...` 命令更换密钥。

### 10. 下载失败或域名无法解析

先测试：

```text
https://modelapi.aiaiaiaiai.cloud/install.sh
```

能否在浏览器中打开。如果浏览器也打不开，请检查网络、DNS、代理或防火墙后重试。在线脚本会优先使用 AiEngine 下载源；主源不可用时会尝试 GitHub Release，并且两个来源都要通过 SHA-256 完整性校验。

### 11. Claude Desktop 配置后没有出现 AiEngine 模型

先完全退出 Claude Desktop 再重新打开。Windows 需要同时检查任务栏托盘，macOS 使用 `Command + Q`。然后运行 `doctor`：

```text
Windows: & "$env:LOCALAPPDATA\AiEngine\CLISetup\bin\aiengine-setup.exe" doctor
macOS:   ~/.aiengine-setup/bin/aiengine-setup doctor
```

如果诊断指出 `/v1/messages`、流式 SSE 或模型权限失败，说明当前密钥或 AiEngine 上游分组不能满足桌面版要求；这类密钥即使能通过 `/v1/models`，也不能用于 Claude Desktop。

## 十、安全和配置说明

- Claude Code 和 Codex 的密钥不会出现在客户端主配置文件里。
- Claude Desktop 的 3P gateway 协议要求密钥保存在其 AiEngine profile。安装器会在 macOS 设置 `0600` 权限，在 Windows 收紧为当前用户、管理员和系统账户可访问；安装状态只记录文件哈希，不记录密钥。
- Claude Code、Claude Desktop 和 Codex 的密钥分开保存，互不覆盖。
- Windows 密钥保存在当前用户的 `%LOCALAPPDATA%\AiEngine\CLISetup\credentials` 下，并通过 Windows ACL 收紧访问权限。
- macOS、Linux 和 WSL 密钥保存在 `~/.aiengine-setup/credentials` 下，文件权限为当前用户可读写。
- 修改客户端配置前会保留备份；安装器只合并自己需要的字段。
- 已有 Codex 官方登录不会被读取、替换或删除。
- 在线下载的安装包会先进行 SHA-256 校验，再运行安装器。

遇到问题需要寻求帮助时，可以提供系统类型、使用的终端、完整错误文字和 `doctor` 输出，但不要提供或截图自己的 API 密钥。
