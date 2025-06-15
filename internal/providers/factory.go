package providers

import (
	"fmt"
	"strings"
)

type ProviderFactory struct {
	providers map[string]Provider
}

func NewProviderFactory(apiKeys map[string]string) *ProviderFactory {

	providers := make(map[string]Provider)

	if apiKey, ok := apiKeys["openai"]; ok {
		providers["openai"] = NewOpenAIProvider(apiKey)
	}

	if apiKey, ok := apiKeys["anthropic"]; ok {
		providers["anthropic"] = NewAntropicProvider(apiKey)
	}

	return &ProviderFactory{
		providers: providers}
}

func (f *ProviderFactory) GetProvider(model string) (Provider, error) {
	providerType := f.getProviderTypeFromModel(model)

	provider, exists := f.providers[providerType]
	if !exists {
		return nil, fmt.Errorf("provider not found for model: %s", model)
	}

	return provider, nil
}

func (f *ProviderFactory) getProviderTypeFromModel(model string) string {
	modelLower := strings.ToLower(model)

	if strings.HasPrefix(modelLower, "gpt") {
		return "openai"
	}
	if strings.HasPrefix(modelLower, "claude") {
		return "anthropic"
	}

	return "openai"
}
