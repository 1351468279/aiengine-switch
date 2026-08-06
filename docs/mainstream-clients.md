# 主流客户端接入 AiEngine

AiEngine Setup 会自动配置 Claude Code、Claude Desktop、Codex、Hermes Agent、OpenCode 和 Aider。本文说明 Continue、Cline、Roo Code、Cherry Studio 等界面型客户端的手动接入方式，并单独说明 Cursor 的有限接入边界。

这些客户端的设置界面会随版本变化，因此安装器不会直接修改它们，也不会在 `doctor` 中检查它们。

## 通用参数

除 Cherry Studio 外，大多数 OpenAI 兼容客户端填写：

| 字段 | 内容 |
| --- | --- |
| Provider/API Provider | `OpenAI Compatible`；没有该项时选 `OpenAI` |
| Base URL/API Base | `https://modelapi.aiaiaiaiai.cloud/v1` |
| API Key | 在 AiEngine 控制台创建的密钥 |
| Model/Model ID | 该密钥有权限的准确模型 ID |

常用模型示例：

```text
GPT Pro 分组: gpt-5.6-sol
Claude 分组:  claude-sonnet-5
```

不要把示例当作固定模型列表。模型必须与 AiEngine 控制台授权和 `/v1/models` 返回结果一致。编程代理还依赖工具调用能力；模型能普通聊天，不代表一定能让代理正常编辑文件或执行命令。

## Continue

Continue 当前使用 YAML 配置。打开 Continue 的本地配置文件，在现有 `models` 列表中增加一个 AiEngine 模型：

```yaml
name: Local Config
version: 0.0.1
schema: v1

models:
  - name: AiEngine GPT
    provider: openai
    model: gpt-5.6-sol
    apiBase: https://modelapi.aiaiaiaiai.cloud/v1
    apiKey: YOUR_AIENGINE_API_KEY
```

如果使用 Claude 分组，把 `name`、`model` 和密钥一起改成对应值。保存后，在 Continue 的模型选择器中选择新模型并新建对话。

GPT-5 类模型若因 Responses API 兼容问题报错，可在该模型下增加：

```yaml
    useResponsesApi: false
```

Continue 配置字段会随版本演进，完整格式以 [Continue 官方 OpenAI 兼容配置](https://docs.continue.dev/customize/model-providers/top-level/openai) 为准。配置文件包含密钥，不要上传到 Git 仓库或发给他人。

## Cline

1. 在 VS Code 打开 Cline。
2. 点击 Cline 面板中的设置图标。
3. `API Provider` 选择 `OpenAI Compatible`。
4. `Base URL` 填写 `https://modelapi.aiaiaiaiai.cloud/v1`。
5. `API Key` 填写 AiEngine 密钥。
6. `Model ID` 填写准确模型，例如 `gpt-5.6-sol` 或 `claude-sonnet-5`。
7. 点击 `Verify`；通过后新建任务测试读取文件和编辑操作。

不要选择官方 `OpenAI` Provider 后只填密钥，这通常会把请求发到 OpenAI 官方地址。字段说明见 [Cline 官方 OpenAI Compatible 文档](https://docs.cline.bot/provider-config/openai-compatible)。

## Roo Code

1. 在 VS Code 打开 Roo Code。
2. 点击齿轮，进入 `Providers`。
3. 新建一个配置档案，名称可填 `AiEngine`。
4. `API Provider` 选择 `OpenAI Compatible`。
5. `Base URL` 填写 `https://modelapi.aiaiaiaiai.cloud/v1`。
6. 填入 AiEngine 密钥和准确模型 ID。
7. 保存，并在对话顶部选择 `AiEngine` 配置档案。

Roo Code 只使用原生工具调用。若普通问答正常但读写文件失败，应换用确认支持 OpenAI 兼容工具调用的模型。详细要求见 [Roo Code 官方 OpenAI Compatible 文档](https://docs.roocode.com/providers/openai-compatible) 和[配置档案文档](https://docs.roocode.com/features/api-configuration-profiles)。

## Cherry Studio

Cherry Studio 的 OneAPI/NewAPI 接入会自动拼接 `/v1/...`，这里的地址与前面几个客户端不同。

1. 打开 `设置`，进入 `模型服务`。
2. 点击供应商列表底部的 `添加`。
3. 名称填写 `AiEngine`，供应商类型选择 `OpenAI`。
4. API 密钥填写 AiEngine 密钥。
5. API 地址填写 `https://modelapi.aiaiaiaiai.cloud`，末尾不要加 `/v1`。
6. 点击 `管理` 或 `添加模型`，手动添加准确模型 ID。
7. 打开该供应商右上角开关，然后在聊天页选择对应模型。

如果当前版本明确要求完整 Base URL，而不是“API 地址/根地址”，再改为 `https://modelapi.aiaiaiaiai.cloud/v1`。官方 OneAPI 操作见 [Cherry Studio OneAPI 文档](https://docs.cherry-ai.com/pre-basic/providers/oneapi)。

## Cursor

Cursor 当前官方文档只保证通过 `Cursor Settings > Models` 使用它列出的内置供应商，自带密钥也只适用于标准聊天模型；Tab Completion 等功能仍使用 Cursor 内置模型。官方文档没有承诺任意 OpenAI 兼容 Base URL 的稳定接入，因此 AiEngine 不把 Cursor 列为正式支持客户端。

若你的 Cursor 版本中确实出现 `Override OpenAI Base URL` 或类似字段，可以自行尝试：

```text
OpenAI Base URL: https://modelapi.aiaiaiaiai.cloud/v1
OpenAI API Key:  你的 AiEngine 密钥
Model:           密钥有权限的标准聊天模型
```

若无法验证、无法添加自定义模型，或 Agent/Tab 功能仍走 Cursor 内置服务，这属于 Cursor 当前产品限制。不要修改系统 hosts 或用不透明代理强行替换官方地址。参考 [Cursor 官方 API Keys 文档](https://docs.cursor.com/settings/api-keys)。

## 其他 OpenAI 兼容客户端

对于未列出的客户端，先确认设置中同时存在以下 3 个字段：

1. 自定义 Base URL。
2. 自定义 API Key。
3. 自定义 Model ID。

三个字段都具备时，通常可以按本文“通用参数”尝试。若客户端只允许填写官方 OpenAI 密钥，不能修改 Base URL，它就不能直接接入 AiEngine。

## 验证与排错

配置后先发一个简单问题，再测试客户端特有能力，例如读取当前项目文件、提出修改并调用工具。

| 报错 | 检查内容 |
| --- | --- |
| `401`、`invalid api key` | 密钥是否完整，是否粘贴了空格 |
| `404` | Base URL 是否多写或少写 `/v1`；Cherry Studio 例外 |
| `model not found` | 模型 ID 拼写和密钥分组权限 |
| 普通聊天成功但工具失败 | 模型或上游渠道是否支持原生工具调用 |
| 一直请求官方服务 | 是否误选了固定官方 Provider，而不是 OpenAI Compatible |

手动配置的客户端不会出现在 AiEngine Setup 的 `doctor` 结果里。排错时可提供客户端名称、版本、模型 ID、HTTP 状态码和错误文字，但不要提供 API 密钥。
