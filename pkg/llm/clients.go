package llm

import (
	"fmt"

	"github.com/akshaydubey29/mimirInsights/pkg/config"
	"github.com/sirupsen/logrus"
)

// NewOpenAIClient creates a new OpenAI client
func NewOpenAIClient() (LLMClient, error) {
	cfg := config.Get()

	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("OpenAI API key not configured")
	}

	// TODO: Implement actual OpenAI client
	logrus.Info("OpenAI integration not yet implemented")
	return nil, fmt.Errorf("OpenAI integration not yet implemented")
}

// NewAnthropicClient creates a new Anthropic client
func NewAnthropicClient() (LLMClient, error) {
	cfg := config.Get()

	if cfg.LLM.APIKey == "" {
		return nil, fmt.Errorf("Anthropic API key not configured")
	}

	// TODO: Implement actual Anthropic client
	logrus.Info("Anthropic integration not yet implemented")
	return nil, fmt.Errorf("Anthropic integration not yet implemented")
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient() (LLMClient, error) {
	cfg := config.Get()

	if cfg.LLM.Endpoint == "" {
		return nil, fmt.Errorf("Ollama URL not configured")
	}

	// TODO: Implement actual Ollama client
	logrus.Info("Ollama integration not yet implemented")
	return nil, fmt.Errorf("Ollama integration not yet implemented")
}
