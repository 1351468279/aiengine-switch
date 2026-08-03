/**
 * Deployment-specific settings for the NewAPI relay used by this build.
 * Keep relay credentials out of the application bundle; users enter their own
 * API key in the provider form.
 */
export const RELAY_STATION = {
  name: "AIARE NewAPI",
  baseUrl: "https://modelapi.aiaiaiaiai.cloud",
  websiteUrl: "https://modelapi.aiaiaiaiai.cloud",
  description:
    "AIARE NewAPI 中转站，统一配置 Claude 和 Codex；Gemini 可按需启用。",
} as const;

export const RELAY_STATION_APPS = {
  claude: true,
  codex: true,
  gemini: false,
} as const;

export const RELAY_STATION_UPDATES_ENABLED = false;
