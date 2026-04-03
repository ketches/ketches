package services

import (
	"encoding/json"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
)

type builderExecutionCatalog struct {
	ImageProfiles            map[string]builderImageProfileDefinition
	ExecutorPolicies         map[string]builderExecutorPolicyDefinition
	DefaultImageProfileKey   string
	DefaultExecutorPolicyKey string
}

type builderExecutionCatalogConfig struct {
	ImageProfiles            []builderImageProfileDefinition   `json:"image_profiles"`
	ExecutorPolicies         []builderExecutorPolicyDefinition `json:"executor_policies"`
	DefaultImageProfileKey   string                            `json:"default_image_profile_key"`
	DefaultExecutorPolicyKey string                            `json:"default_executor_policy_key"`
}

type builderImageProfileDefinition struct {
	Key          string   `json:"key"`
	Image        string   `json:"image"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
}

type builderExecutorPolicyDefinition struct {
	Key             string `json:"key"`
	ExecutorKind    string `json:"executor_kind"`
	ImageProfileKey string `json:"image_profile_key"`
}

func loadBuilderExecutionCatalog(config app.AppConfig) (*builderExecutionCatalog, error) {
	if strings.TrimSpace(config.BuilderExecutionCatalogJSON) != "" {
		var catalogConfig builderExecutionCatalogConfig
		if err := json.Unmarshal([]byte(config.BuilderExecutionCatalogJSON), &catalogConfig); err != nil {
			return nil, app.WrapErrorf(err, "parse builder execution catalog: %w", err)
		}
		return normalizeBuilderExecutionCatalog(catalogConfig, config.BuilderDefaultExecutorPolicyKey)
	}

	defaultImageProfile := builderImageProfileDefinition{
		Key:          "workspace-default-image",
		Image:        strings.TrimSpace(config.BuilderWorkspaceImage),
		Description:  "Default Builder workspace image",
		Capabilities: []string{"workspace", "default"},
	}
	if defaultImageProfile.Image == "" {
		defaultImageProfile.Image = "node:22-bookworm"
	}

	defaultPolicyKey := strings.TrimSpace(config.BuilderDefaultExecutorPolicyKey)
	if defaultPolicyKey == "" {
		defaultPolicyKey = "workspace-only"
	}

	catalogConfig := builderExecutionCatalogConfig{
		ImageProfiles: []builderImageProfileDefinition{
			defaultImageProfile,
			{
				Key:          "node-static",
				Image:        defaultImageProfile.Image,
				Description:  "Static frontend workspace image",
				Capabilities: []string{"node", "npm", "static-frontend"},
			},
			{
				Key:          "node-ssr",
				Image:        defaultImageProfile.Image,
				Description:  "Node SSR workspace image",
				Capabilities: []string{"node", "npm", "ssr"},
			},
			{
				Key:          "go-api",
				Image:        defaultImageProfile.Image,
				Description:  "Go API workspace image",
				Capabilities: []string{"go", "api"},
			},
			{
				Key:          "python-api",
				Image:        defaultImageProfile.Image,
				Description:  "Python API workspace image",
				Capabilities: []string{"python", "api"},
			},
			{
				Key:          "full-stack",
				Image:        defaultImageProfile.Image,
				Description:  "Full-stack workspace image",
				Capabilities: []string{"node", "full-stack"},
			},
		},
		ExecutorPolicies: []builderExecutorPolicyDefinition{
			{
				Key:             defaultPolicyKey,
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: defaultImageProfile.Key,
			},
			{
				Key:             "workspace-node-static",
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: "node-static",
			},
			{
				Key:             "workspace-node-ssr",
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: "node-ssr",
			},
			{
				Key:             "workspace-go-api",
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: "go-api",
			},
			{
				Key:             "workspace-python-api",
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: "python-api",
			},
			{
				Key:             "workspace-full-stack",
				ExecutorKind:    string(entities.BuilderExecutorHandleKindWorkspacePod),
				ImageProfileKey: "full-stack",
			},
		},
		DefaultImageProfileKey:   defaultImageProfile.Key,
		DefaultExecutorPolicyKey: defaultPolicyKey,
	}

	return normalizeBuilderExecutionCatalog(catalogConfig, defaultPolicyKey)
}

func normalizeBuilderExecutionCatalog(config builderExecutionCatalogConfig, legacyDefaultExecutorPolicyKey string) (*builderExecutionCatalog, error) {
	imageProfiles := make(map[string]builderImageProfileDefinition, len(config.ImageProfiles))
	for _, imageProfile := range config.ImageProfiles {
		imageProfile.Key = strings.TrimSpace(imageProfile.Key)
		imageProfile.Image = strings.TrimSpace(imageProfile.Image)
		imageProfile.Description = strings.TrimSpace(imageProfile.Description)
		if imageProfile.Key == "" {
			return nil, app.NewErrorf("builder image profile key is required")
		}
		if imageProfile.Image == "" {
			return nil, app.NewErrorf("builder image profile %q image is required", imageProfile.Key)
		}
		if _, exists := imageProfiles[imageProfile.Key]; exists {
			return nil, app.NewErrorf("duplicate builder image profile key %q", imageProfile.Key)
		}
		imageProfiles[imageProfile.Key] = imageProfile
	}

	executorPolicies := make(map[string]builderExecutorPolicyDefinition, len(config.ExecutorPolicies))
	for _, executorPolicy := range config.ExecutorPolicies {
		executorPolicy.Key = strings.TrimSpace(executorPolicy.Key)
		executorPolicy.ExecutorKind = strings.TrimSpace(executorPolicy.ExecutorKind)
		executorPolicy.ImageProfileKey = strings.TrimSpace(executorPolicy.ImageProfileKey)
		if executorPolicy.Key == "" {
			return nil, app.NewErrorf("builder executor policy key is required")
		}
		if executorPolicy.ImageProfileKey == "" {
			return nil, app.NewErrorf("builder executor policy %q image profile key is required", executorPolicy.Key)
		}
		if _, exists := executorPolicies[executorPolicy.Key]; exists {
			return nil, app.NewErrorf("duplicate builder executor policy key %q", executorPolicy.Key)
		}
		if err := validateBuilderExecutorKind(executorPolicy.ExecutorKind); err != nil {
			return nil, app.WrapErrorf(err, "builder executor policy %q: %w", executorPolicy.Key, err)
		}
		if _, exists := imageProfiles[executorPolicy.ImageProfileKey]; !exists {
			return nil, app.NewErrorf("builder executor policy %q references unknown image profile %q", executorPolicy.Key, executorPolicy.ImageProfileKey)
		}
		executorPolicies[executorPolicy.Key] = executorPolicy
	}

	defaultImageProfileKey := strings.TrimSpace(config.DefaultImageProfileKey)
	if defaultImageProfileKey == "" && len(imageProfiles) == 1 {
		for key := range imageProfiles {
			defaultImageProfileKey = key
		}
	}
	if defaultImageProfileKey == "" {
		return nil, app.NewErrorf("builder default image profile key is required")
	}
	if _, exists := imageProfiles[defaultImageProfileKey]; !exists {
		return nil, app.NewErrorf("unknown builder default image profile key %q", defaultImageProfileKey)
	}

	defaultExecutorPolicyKey := strings.TrimSpace(config.DefaultExecutorPolicyKey)
	if defaultExecutorPolicyKey == "" {
		defaultExecutorPolicyKey = strings.TrimSpace(legacyDefaultExecutorPolicyKey)
	}
	if defaultExecutorPolicyKey == "" && len(executorPolicies) == 1 {
		for key := range executorPolicies {
			defaultExecutorPolicyKey = key
		}
	}
	if defaultExecutorPolicyKey == "" {
		return nil, app.NewErrorf("builder default executor policy key is required")
	}
	if _, exists := executorPolicies[defaultExecutorPolicyKey]; !exists {
		return nil, app.NewErrorf("unknown builder default executor policy key %q", defaultExecutorPolicyKey)
	}

	return &builderExecutionCatalog{
		ImageProfiles:            imageProfiles,
		ExecutorPolicies:         executorPolicies,
		DefaultImageProfileKey:   defaultImageProfileKey,
		DefaultExecutorPolicyKey: defaultExecutorPolicyKey,
	}, nil
}

func validateBuilderExecutorKind(kind string) error {
	switch strings.TrimSpace(kind) {
	case string(entities.BuilderExecutorHandleKindWorkspacePod),
		string(entities.BuilderExecutorHandleKindBuildJob),
		string(entities.BuilderExecutorHandleKindSandbox):
		return nil
	default:
		return app.NewErrorf("unsupported builder executor kind %q", kind)
	}
}
