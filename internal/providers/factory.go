package providers

import (
	"context"
	"fmt"
	"os"

	"github.com/ChamsBouzaiene/dodo/internal/engine"
)

// NewLLMClientFromEnv creates an engine.LLMClient based on environment variables.
// This is a factory function that creates provider-specific clients without Eino.
func NewLLMClientFromEnv(ctx context.Context) (engine.LLMClient, string, error) {
	provider := os.Getenv("LLM_PROVIDER")
	if provider == "" {
		// Default to OpenAI if not set
		provider = "openai"
	}

	switch provider {
	case "openai":
		apiKey := os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("OPENAI_API_KEY not set")
		}

		modelName := os.Getenv("OPENAI_MODEL")
		if modelName == "" {
			modelName = "gpt-4o-mini"
		}

		baseURL := os.Getenv("OPENAI_BASE_URL") // For OpenAI-compatible APIs like Kimi

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create OpenAI client: %w", err)
		}

		return client, modelName, nil

	case "anthropic":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("ANTHROPIC_API_KEY not set")
		}

		modelName := os.Getenv("ANTHROPIC_MODEL")
		if modelName == "" {
			modelName = "claude-3-sonnet-20240229"
		}

		client, err := NewAnthropicClient(apiKey, modelName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Anthropic client: %w", err)
		}

		return client, modelName, nil

	case "kimi":
		// Kimi uses OpenAI-compatible API via BytePlus ModelArk
		// Base URL: https://ark.ap-southeast.bytepluses.com/api/v3
		apiKey := os.Getenv("KIMI_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("KIMI_API_KEY not set")
		}

		modelName := os.Getenv("KIMI_MODEL")
		if modelName == "" {
			modelName = "kimi-k2-250711"
		}

		// Use the correct BytePlus ModelArk base URL (matches old Eino implementation)
		// Can be overridden via KIMI_BASE_URL environment variable
		baseURL := os.Getenv("KIMI_BASE_URL")
		if baseURL == "" {
			baseURL = "https://ark.ap-southeast.bytepluses.com/api/v3"
		}

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Kimi client: %w", err)
		}

		return client, modelName, nil

	case "gemini":
		// Google Gemini via OpenAI-compatible endpoint
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("GEMINI_API_KEY not set")
		}

		modelName := os.Getenv("GEMINI_MODEL")
		if modelName == "" {
			modelName = "gemini-1.5-flash"
		}

		// Gemini OpenAI-compatible endpoint
		baseURL := "https://generativelanguage.googleapis.com/v1beta/openai"

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Gemini client: %w", err)
		}

		return client, modelName, nil

	case "lmstudio":
		// LM Studio local server (OpenAI-compatible)
		baseURL := os.Getenv("LMSTUDIO_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:1234/v1"
		}

		modelName := os.Getenv("LMSTUDIO_MODEL")
		if modelName == "" {
			modelName = "local-model"
		}

		// API key can be anything for local models
		apiKey := os.Getenv("LMSTUDIO_API_KEY")
		if apiKey == "" {
			apiKey = "lm-studio"
		}

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create LM Studio client: %w", err)
		}

		return client, modelName, nil

	case "ollama":
		// Ollama local server (OpenAI-compatible)
		baseURL := os.Getenv("OLLAMA_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost:11434/v1"
		}

		modelName := os.Getenv("OLLAMA_MODEL")
		if modelName == "" {
			modelName = "llama3.1"
		}

		apiKey := os.Getenv("OLLAMA_API_KEY")
		if apiKey == "" {
			apiKey = "ollama"
		}

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Ollama client: %w", err)
		}

		return client, modelName, nil

	case "glm":
		// ZhipuAI GLM-4 (OpenAI-compatible)
		apiKey := os.Getenv("GLM_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("GLM_API_KEY not set")
		}

		modelName := os.Getenv("GLM_MODEL")
		if modelName == "" {
			modelName = "glm-4-plus"
		}

		baseURL := "https://open.bigmodel.cn/api/paas/v4"

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create GLM client: %w", err)
		}

		return client, modelName, nil

	case "minimax":
		// MiniMax LLM (OpenAI-compatible)
		apiKey := os.Getenv("MINIMAX_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("MINIMAX_API_KEY not set")
		}

		modelName := os.Getenv("MINIMAX_MODEL")
		if modelName == "" {
			modelName = "abab6.5s-chat"
		}

		baseURL := "https://api.minimax.chat/v1"

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create MiniMax client: %w", err)
		}

		return client, modelName, nil

	case "deepseek":
		// DeepSeek (OpenAI-compatible)
		apiKey := os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("DEEPSEEK_API_KEY not set")
		}

		modelName := os.Getenv("DEEPSEEK_MODEL")
		if modelName == "" {
			modelName = "deepseek-chat"
		}

		baseURL := "https://api.deepseek.com/v1"

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create DeepSeek client: %w", err)
		}

		return client, modelName, nil

	case "groq":
		// Groq (OpenAI-compatible, very fast inference)
		apiKey := os.Getenv("GROQ_API_KEY")
		if apiKey == "" {
			return nil, "", fmt.Errorf("GROQ_API_KEY not set")
		}

		modelName := os.Getenv("GROQ_MODEL")
		if modelName == "" {
			modelName = "llama-3.1-70b-versatile"
		}

		baseURL := "https://api.groq.com/openai/v1"

		client, err := NewOpenAIClient(apiKey, modelName, baseURL)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create Groq client: %w", err)
		}

		return client, modelName, nil

	default:
		return nil, "", fmt.Errorf("unknown LLM_PROVIDER: %s (supported: openai, anthropic, kimi, gemini, lmstudio, ollama, glm, minimax, deepseek, groq)", provider)
	}
}
