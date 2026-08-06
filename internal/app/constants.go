package app

const (
	ProductName        = "AiEngine Setup"
	RelayRootURL       = "https://modelapi.aiaiaiaiai.cloud"
	RelayV1URL         = RelayRootURL + "/v1"
	ClaudeModel        = "claude-sonnet-5"
	ClaudeOpusModel    = "claude-opus-5"
	ClaudeHaikuModel   = "claude-haiku-4-5-20251001"
	CodexModel         = "gpt-5.6-sol"
	GeneralModel       = ClaudeModel
	stateSchema        = 3
	codexProviderID    = "aiengine"
	legacyProviderID   = "aiare"
	desktopTool        = "claude-desktop"
	desktopProfileID   = "a1e00000-0000-4000-8000-000000000001"
	desktopProfileName = "AiEngine"
)

var cliTools = []string{"claude", "codex", "hermes", "opencode", "aider"}

var genericTools = map[string]bool{
	"hermes":   true,
	"opencode": true,
	"aider":    true,
}

var managedClaudeFields = []string{
	"apiKeyHelper",
	"model",
	"env.ANTHROPIC_BASE_URL",
	"env.ANTHROPIC_MODEL",
	"env.ANTHROPIC_DEFAULT_SONNET_MODEL",
	"env.ANTHROPIC_DEFAULT_OPUS_MODEL",
	"env.ANTHROPIC_DEFAULT_HAIKU_MODEL",
}

var managedCodexFields = []string{
	"model",
	"model_provider",
	"model_reasoning_effort",
	"disable_response_storage",
}
