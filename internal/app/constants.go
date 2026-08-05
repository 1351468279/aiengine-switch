package app

const (
	ProductName      = "AiEngine CLI Setup"
	RelayRootURL     = "https://modelapi.aiaiaiaiai.cloud"
	RelayV1URL       = RelayRootURL + "/v1"
	ClaudeModel      = "claude-sonnet-5"
	ClaudeOpusModel  = "claude-opus-5"
	ClaudeHaikuModel = "claude-haiku-4-5-20251001"
	CodexModel       = "gpt-5.6-sol"
	stateSchema      = 2
	codexProviderID  = "aiengine"
	legacyProviderID = "aiare"
)

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
