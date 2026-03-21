package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
)

type builderProviderRegistry struct {
	providers              map[string]builderProviderDefinition
	modelProfiles          map[string]builderModelProfileDefinition
	defaultProviderKey     string
	defaultModelProfileKey string
}

type builderProviderDefinition struct {
	Key     string `json:"key"`
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type builderModelProfileDefinition struct {
	Key   string `json:"key"`
	Model string `json:"model"`
}

type builderResolvedProviderProfile struct {
	Provider     builderProviderDefinition
	ModelProfile builderModelProfileDefinition
}

func loadBuilderProviderRegistry(config app.AppConfig) (*builderProviderRegistry, error) {
	providers, err := loadBuilderProviderDefinitions(config)
	if err != nil {
		return nil, err
	}

	modelProfiles, err := loadBuilderModelProfileDefinitions(config)
	if err != nil {
		return nil, err
	}

	registry := &builderProviderRegistry{
		providers:              providers,
		modelProfiles:          modelProfiles,
		defaultProviderKey:     normalizeBuilderProviderKey(config.BuilderDefaultProviderKey),
		defaultModelProfileKey: normalizeBuilderModelProfileKey(config.BuilderDefaultModelProfileKey),
	}

	if _, err := registry.resolveBuilderProviderProfile("", ""); err != nil {
		return nil, err
	}

	return registry, nil
}

func (r *builderProviderRegistry) resolveBuilderProviderProfile(providerKey, modelProfileKey string) (builderResolvedProviderProfile, error) {
	resolvedProviderKey := strings.TrimSpace(providerKey)
	if resolvedProviderKey == "" {
		resolvedProviderKey = r.defaultProviderKey
	}

	provider, ok := r.providers[resolvedProviderKey]
	if !ok {
		return builderResolvedProviderProfile{}, fmt.Errorf("unknown builder provider alias %q", resolvedProviderKey)
	}

	resolvedModelProfileKey := strings.TrimSpace(modelProfileKey)
	if resolvedModelProfileKey == "" {
		resolvedModelProfileKey = r.defaultModelProfileKey
	}

	modelProfile, ok := r.modelProfiles[resolvedModelProfileKey]
	if !ok {
		return builderResolvedProviderProfile{}, fmt.Errorf("unknown builder model profile alias %q", resolvedModelProfileKey)
	}

	return builderResolvedProviderProfile{
		Provider:     provider,
		ModelProfile: modelProfile,
	}, nil
}

func loadBuilderProviderDefinitions(config app.AppConfig) (map[string]builderProviderDefinition, error) {
	if strings.TrimSpace(config.BuilderProviderRegistryJSON) != "" {
		var providers []builderProviderDefinition
		if err := json.Unmarshal([]byte(config.BuilderProviderRegistryJSON), &providers); err != nil {
			return nil, fmt.Errorf("parse builder provider registry: %w", err)
		}

		return indexBuilderProviders(providers)
	}

	baseURL := strings.TrimSpace(config.BuilderAgentBaseURL)
	if baseURL == "" {
		return map[string]builderProviderDefinition{}, nil
	}

	key := normalizeBuilderProviderKey(config.BuilderDefaultProviderKey)
	return map[string]builderProviderDefinition{
		key: {
			Key:     key,
			BaseURL: baseURL,
			APIKey:  strings.TrimSpace(config.BuilderAgentAPIKey),
		},
	}, nil
}

func loadBuilderModelProfileDefinitions(config app.AppConfig) (map[string]builderModelProfileDefinition, error) {
	if strings.TrimSpace(config.BuilderModelProfileRegistryJSON) != "" {
		var modelProfiles []builderModelProfileDefinition
		if err := json.Unmarshal([]byte(config.BuilderModelProfileRegistryJSON), &modelProfiles); err != nil {
			return nil, fmt.Errorf("parse builder model profile registry: %w", err)
		}

		return indexBuilderModelProfiles(modelProfiles)
	}

	model := strings.TrimSpace(config.BuilderAgentModel)
	if model == "" {
		return map[string]builderModelProfileDefinition{}, nil
	}

	key := normalizeBuilderModelProfileKey(config.BuilderDefaultModelProfileKey)
	return map[string]builderModelProfileDefinition{
		key: {
			Key:   key,
			Model: model,
		},
	}, nil
}

func indexBuilderProviders(providers []builderProviderDefinition) (map[string]builderProviderDefinition, error) {
	indexed := make(map[string]builderProviderDefinition, len(providers))
	for _, provider := range providers {
		provider.Key = strings.TrimSpace(provider.Key)
		provider.BaseURL = strings.TrimSpace(provider.BaseURL)
		provider.APIKey = strings.TrimSpace(provider.APIKey)

		if provider.Key == "" {
			return nil, fmt.Errorf("builder provider key is required")
		}
		if provider.BaseURL == "" {
			return nil, fmt.Errorf("builder provider %q base URL is required", provider.Key)
		}
		if _, exists := indexed[provider.Key]; exists {
			return nil, fmt.Errorf("duplicate builder provider alias %q", provider.Key)
		}

		indexed[provider.Key] = provider
	}

	return indexed, nil
}

func indexBuilderModelProfiles(modelProfiles []builderModelProfileDefinition) (map[string]builderModelProfileDefinition, error) {
	indexed := make(map[string]builderModelProfileDefinition, len(modelProfiles))
	for _, modelProfile := range modelProfiles {
		modelProfile.Key = strings.TrimSpace(modelProfile.Key)
		modelProfile.Model = strings.TrimSpace(modelProfile.Model)

		if modelProfile.Key == "" {
			return nil, fmt.Errorf("builder model profile key is required")
		}
		if modelProfile.Model == "" {
			return nil, fmt.Errorf("builder model profile %q model is required", modelProfile.Key)
		}
		if _, exists := indexed[modelProfile.Key]; exists {
			return nil, fmt.Errorf("duplicate builder model profile alias %q", modelProfile.Key)
		}

		indexed[modelProfile.Key] = modelProfile
	}

	return indexed, nil
}

func normalizeBuilderProviderKey(key string) string {
	resolved := strings.TrimSpace(key)
	if resolved == "" {
		return "default"
	}

	return resolved
}

func normalizeBuilderModelProfileKey(key string) string {
	resolved := strings.TrimSpace(key)
	if resolved == "" {
		return "builder-default"
	}

	return resolved
}
