package ai

import (
	"os"
	"path/filepath"
)

// providerEnvMap maps provider identifiers to their API key environment variables.
var providerEnvMap = map[Provider]string{
	ProviderOpenAI:               "OPENAI_API_KEY",
	ProviderAzureOpenAIResponses: "AZURE_OPENAI_API_KEY",
	ProviderGoogle:               "GEMINI_API_KEY",
	ProviderDeepSeek:             "DEEPSEEK_API_KEY",
	ProviderGroq:                 "GROQ_API_KEY",
	ProviderCerebras:             "CEREBRAS_API_KEY",
	ProviderXAI:                  "XAI_API_KEY",
	ProviderOpenRouter:           "OPENROUTER_API_KEY",
	ProviderVercelAIGateway:      "AI_GATEWAY_API_KEY",
	ProviderZAI:                  "ZAI_API_KEY",
	ProviderMistral:              "MISTRAL_API_KEY",
	ProviderMinimax:              "MINIMAX_API_KEY",
	ProviderMinimaxCN:            "MINIMAX_CN_API_KEY",
	ProviderHuggingFace:          "HF_TOKEN",
	ProviderOpenCode:             "OPENCODE_API_KEY",
	ProviderKimiCoding:           "KIMI_API_KEY",
}

// GetEnvApiKey returns the API key for a provider from environment variables.
// Returns empty string if no key is found.
func GetEnvApiKey(provider Provider) string {
	switch provider {
	case ProviderGitHubCopilot:
		if v := os.Getenv("COPILOT_GITHUB_TOKEN"); v != "" {
			return v
		}
		if v := os.Getenv("GH_TOKEN"); v != "" {
			return v
		}
		return os.Getenv("GITHUB_TOKEN")

	case ProviderAnthropic:
		if v := os.Getenv("ANTHROPIC_OAUTH_TOKEN"); v != "" {
			return v
		}
		return os.Getenv("ANTHROPIC_API_KEY")

	case ProviderGoogleVertex:
		if hasVertexADCCredentials() &&
			(os.Getenv("GOOGLE_CLOUD_PROJECT") != "" || os.Getenv("GCLOUD_PROJECT") != "") &&
			os.Getenv("GOOGLE_CLOUD_LOCATION") != "" {
			return "<authenticated>"
		}
		return ""

	case ProviderAmazonBedrock:
		if os.Getenv("AWS_PROFILE") != "" ||
			(os.Getenv("AWS_ACCESS_KEY_ID") != "" && os.Getenv("AWS_SECRET_ACCESS_KEY") != "") ||
			os.Getenv("AWS_BEARER_TOKEN_BEDROCK") != "" ||
			os.Getenv("AWS_CONTAINER_CREDENTIALS_RELATIVE_URI") != "" ||
			os.Getenv("AWS_CONTAINER_CREDENTIALS_FULL_URI") != "" ||
			os.Getenv("AWS_WEB_IDENTITY_TOKEN_FILE") != "" {
			return "<authenticated>"
		}
		return ""
	}

	envVar, ok := providerEnvMap[provider]
	if !ok {
		return ""
	}
	return os.Getenv(envVar)
}

// hasVertexADCCredentials checks if Google Vertex AI Application Default Credentials exist.
func hasVertexADCCredentials() bool {
	// Check GOOGLE_APPLICATION_CREDENTIALS env var first
	if gacPath := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); gacPath != "" {
		_, err := os.Stat(gacPath)
		return err == nil
	}

	// Fall back to default ADC path
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	adcPath := filepath.Join(home, ".config", "gcloud", "application_default_credentials.json")
	_, err = os.Stat(adcPath)
	return err == nil
}
