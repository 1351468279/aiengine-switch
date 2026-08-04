# Codex CLI 中转站配置

本仓库提供只配置 Codex CLI 的在线脚本。脚本会把 Codex 的 API 地址设置为：

`https://modelapi.aiaiaiaiai.cloud/v1`

## macOS / Linux

先安装 Codex CLI，然后运行：

```bash
curl -fsSL https://raw.githubusercontent.com/1351468279/aiengine-switch/main/scripts/setup-codex.sh | bash
```

脚本会创建 `~/.codex` 和 `config.toml`（如果不存在），并在修改已有配置前创建备份。随后它会隐藏输入 API Key，并调用 `codex login --with-api-key` 完成认证。

只修改配置、不登录时：

```bash
curl -fsSL https://raw.githubusercontent.com/1351468279/aiengine-switch/main/scripts/setup-codex.sh | bash -s -- --skip-login
```

如果设置了 `CODEX_HOME`，脚本会使用该目录下的 `config.toml`。自动化场景可以通过临时环境变量 `AIARE_CODEX_API_KEY` 提供密钥；这个变量不会被写入配置文件。

## Windows PowerShell

```powershell
$p = Join-Path $env:TEMP "aiare-setup-codex.ps1"
Invoke-RestMethod https://raw.githubusercontent.com/1351468279/aiengine-switch/main/scripts/setup-codex.ps1 -OutFile $p
Set-ExecutionPolicy -Scope Process Bypass
& $p
```

只修改配置时：

```powershell
& $p -SkipLogin
```

## 脚本行为

- 不存在 `~/.codex` 或 `config.toml` 时自动创建。
- 已有配置会保留其他设置，只更新顶层 `model_provider` 和 `openai_base_url`。
- 已有配置先备份为 `config.toml.bak.YYYYMMDD-HHMMSS`。
- API Key 交给 Codex 的认证机制保存，不写入 `config.toml`，也不会出现在命令行参数中。

脚本只负责配置 Codex CLI，不会安装 Codex，也不会修改 Claude、Gemini 或其他工具的配置。
