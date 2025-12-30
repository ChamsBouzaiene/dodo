package main

import (
	"os"

	"github.com/ChamsBouzaiene/dodo/internal/config"
)

func applyConfigToEnv(cfg *config.Config) {
	if cfg.LLMProvider != "" {
		os.Setenv("LLM_PROVIDER", cfg.LLMProvider)
	}
	if cfg.APIKey != "" {
		switch cfg.LLMProvider {
		case "openai":
			os.Setenv("OPENAI_API_KEY", cfg.APIKey)
		case "anthropic":
			os.Setenv("ANTHROPIC_API_KEY", cfg.APIKey)
		case "kimi":
			os.Setenv("KIMI_API_KEY", cfg.APIKey)
		}
	}
	if cfg.Model != "" {
		switch cfg.LLMProvider {
		case "openai":
			os.Setenv("OPENAI_MODEL", cfg.Model)
		case "anthropic":
			os.Setenv("ANTHROPIC_MODEL", cfg.Model)
		case "kimi":
			os.Setenv("KIMI_MODEL", cfg.Model)
		}
	}
	if cfg.BaseURL != "" {
		switch cfg.LLMProvider {
		case "openai":
			os.Setenv("OPENAI_BASE_URL", cfg.BaseURL)
		case "kimi":
			os.Setenv("KIMI_BASE_URL", cfg.BaseURL)
		}
	}
}
