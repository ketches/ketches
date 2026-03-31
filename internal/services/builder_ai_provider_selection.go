package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
)

const (
	builderProviderScopeProject = "project"
	builderProviderScopeUser    = "user"
)

type builderResolvedAIProvider struct {
	Scope           string
	ProviderKey     string
	ProviderLabel   string
	BaseURL         string
	APIKey          string
	ModelProfileKey string
}

type builderRunGenerationContextKey struct{}

type builderRunGenerationContext struct {
	ProjectID     string
	UserID        string
	ProviderScope string
}

func buildBuilderModelOptionKey(scope, providerKey, modelProfileKey string) string {
	return fmt.Sprintf("%s:%s:%s", scope, providerKey, modelProfileKey)
}

func parseBuilderModelSelectionScope(selectedModelKey string) string {
	parts := strings.SplitN(strings.TrimSpace(selectedModelKey), ":", 3)
	if len(parts) != 3 {
		return ""
	}
	if parts[0] != builderProviderScopeProject && parts[0] != builderProviderScopeUser {
		return ""
	}

	return parts[0]
}

func withBuilderRunGenerationContext(ctx context.Context, projectID string, run *entities.BuilderRun) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if run == nil {
		return ctx
	}

	return context.WithValue(ctx, builderRunGenerationContextKey{}, builderRunGenerationContext{
		ProjectID:     strings.TrimSpace(projectID),
		UserID:        strings.TrimSpace(run.RequestedBy),
		ProviderScope: strings.TrimSpace(stringPointerValue(run.ProviderScope)),
	})
}

func builderRunGenerationSelectionFromContext(ctx context.Context) (builderRunGenerationContext, bool) {
	if ctx == nil {
		return builderRunGenerationContext{}, false
	}

	selection, ok := ctx.Value(builderRunGenerationContextKey{}).(builderRunGenerationContext)
	if !ok {
		return builderRunGenerationContext{}, false
	}

	return selection, true
}

func loadBuilderSessionProjectID(ctx context.Context, sessionID string) (string, error) {
	var session entities.BuilderSession
	if err := db.DB.WithContext(ctx).
		Select("id", "project_id").
		Where("id = ?", sessionID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("builder session %s not found", sessionID)
		}
		return "", err
	}

	return strings.TrimSpace(session.ProjectID), nil
}

func resolveBuilderAIProviderForExecution(
	ctx context.Context,
	projectID,
	userID,
	providerScope,
	providerKey,
	modelProfileKey string,
) (*builderResolvedAIProvider, error) {
	tx := db.DB.WithContext(ctx)
	return resolveBuilderAIProviderForExecutionTx(
		tx,
		projectID,
		userID,
		providerScope,
		providerKey,
		modelProfileKey,
	)
}

func resolveBuilderAIProviderForExecutionTx(
	tx *gorm.DB,
	projectID,
	userID,
	providerScope,
	providerKey,
	modelProfileKey string,
) (*builderResolvedAIProvider, error) {
	resolvedScope := strings.TrimSpace(providerScope)
	resolvedProviderKey := strings.TrimSpace(providerKey)
	resolvedModelProfileKey := strings.TrimSpace(modelProfileKey)

	if resolvedProviderKey == "" && resolvedModelProfileKey == "" {
		return resolveDefaultBuilderAIProviderTx(tx, projectID, userID)
	}
	if resolvedProviderKey == "" || resolvedModelProfileKey == "" {
		return nil, errors.New("builder provider key and model profile key must both be provided")
	}

	switch resolvedScope {
	case builderProviderScopeProject:
		provider, err := loadProjectAIProviderForSelectionTx(tx, projectID, resolvedProviderKey, resolvedModelProfileKey)
		if err != nil {
			return nil, err
		}
		return provider, nil
	case builderProviderScopeUser:
		provider, err := loadUserAIProviderForSelectionTx(tx, userID, resolvedProviderKey, resolvedModelProfileKey)
		if err != nil {
			return nil, err
		}
		return provider, nil
	case "":
		return resolveUnscopedBuilderAIProviderTx(tx, projectID, userID, resolvedProviderKey, resolvedModelProfileKey)
	default:
		return nil, fmt.Errorf("unsupported builder provider scope %q", resolvedScope)
	}
}

func resolveDefaultBuilderAIProviderTx(tx *gorm.DB, projectID, userID string) (*builderResolvedAIProvider, error) {
	if provider, err := loadDefaultProjectAIProviderTx(tx, projectID); err == nil {
		return provider, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if provider, err := loadDefaultUserAIProviderTx(tx, userID); err == nil {
		return provider, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	options, err := ListBuilderAvailableModelOptions(projectID, userID)
	if err != nil {
		return nil, err
	}
	if len(options) == 1 {
		return resolveBuilderAIProviderForExecutionTx(
			tx,
			projectID,
			userID,
			options[0].Scope,
			options[0].ProviderKey,
			options[0].ModelProfileKey,
		)
	}

	if len(options) == 0 {
		return nil, errors.New("no enabled builder AI providers available")
	}

	return nil, errors.New("no builder model selected and no effective default provider configured")
}

func resolveUnscopedBuilderAIProviderTx(
	tx *gorm.DB,
	projectID,
	userID,
	providerKey,
	modelProfileKey string,
) (*builderResolvedAIProvider, error) {
	projectProvider, projectErr := loadProjectAIProviderForSelectionTx(tx, projectID, providerKey, modelProfileKey)
	if projectErr != nil && !errors.Is(projectErr, gorm.ErrRecordNotFound) {
		return nil, projectErr
	}

	userProvider, userErr := loadUserAIProviderForSelectionTx(tx, userID, providerKey, modelProfileKey)
	if userErr != nil && !errors.Is(userErr, gorm.ErrRecordNotFound) {
		return nil, userErr
	}

	projectFound := projectErr == nil
	userFound := userErr == nil

	switch {
	case projectFound && userFound:
		return nil, fmt.Errorf(
			"ambiguous builder provider selection for provider %q and model %q; send selected_model_key to preserve project or user scope",
			providerKey,
			modelProfileKey,
		)
	case projectFound:
		return projectProvider, nil
	case userFound:
		return userProvider, nil
	default:
		return nil, fmt.Errorf(
			"builder provider selection not found for provider %q and model %q",
			providerKey,
			modelProfileKey,
		)
	}
}

func loadProjectAIProviderForSelectionTx(tx *gorm.DB, projectID, providerKey, modelProfileKey string) (*builderResolvedAIProvider, error) {
	var provider entities.ProjectAIProvider
	if err := tx.
		Where(
			"project_id = ? AND provider_key = ? AND default_model_profile_key = ?",
			projectID,
			providerKey,
			modelProfileKey,
		).
		Order("is_default DESC, created_at ASC, id ASC").
		First(&provider).Error; err != nil {
		return nil, err
	}

	return &builderResolvedAIProvider{
		Scope:           builderProviderScopeProject,
		ProviderKey:     provider.ProviderKey,
		ProviderLabel:   provider.DisplayName,
		BaseURL:         provider.BaseURL,
		APIKey:          provider.APIKey,
		ModelProfileKey: provider.DefaultModelProfileKey,
	}, nil
}

func loadUserAIProviderForSelectionTx(tx *gorm.DB, userID, providerKey, modelProfileKey string) (*builderResolvedAIProvider, error) {
	var provider entities.UserAIProvider
	if err := tx.
		Where(
			"user_id = ? AND provider_key = ? AND default_model_profile_key = ?",
			userID,
			providerKey,
			modelProfileKey,
		).
		Order("is_default DESC, created_at ASC, id ASC").
		First(&provider).Error; err != nil {
		return nil, err
	}

	return &builderResolvedAIProvider{
		Scope:           builderProviderScopeUser,
		ProviderKey:     provider.ProviderKey,
		ProviderLabel:   provider.DisplayName,
		BaseURL:         provider.BaseURL,
		APIKey:          provider.APIKey,
		ModelProfileKey: provider.DefaultModelProfileKey,
	}, nil
}

func loadDefaultProjectAIProviderTx(tx *gorm.DB, projectID string) (*builderResolvedAIProvider, error) {
	var provider entities.ProjectAIProvider
	if err := tx.
		Where("project_id = ? AND enabled = ? AND is_default = ?", projectID, true, true).
		Order("created_at ASC, id ASC").
		First(&provider).Error; err != nil {
		return nil, err
	}

	return &builderResolvedAIProvider{
		Scope:           builderProviderScopeProject,
		ProviderKey:     provider.ProviderKey,
		ProviderLabel:   provider.DisplayName,
		BaseURL:         provider.BaseURL,
		APIKey:          provider.APIKey,
		ModelProfileKey: provider.DefaultModelProfileKey,
	}, nil
}

func loadDefaultUserAIProviderTx(tx *gorm.DB, userID string) (*builderResolvedAIProvider, error) {
	var provider entities.UserAIProvider
	if err := tx.
		Where("user_id = ? AND enabled = ? AND is_default = ?", userID, true, true).
		Order("created_at ASC, id ASC").
		First(&provider).Error; err != nil {
		return nil, err
	}

	return &builderResolvedAIProvider{
		Scope:           builderProviderScopeUser,
		ProviderKey:     provider.ProviderKey,
		ProviderLabel:   provider.DisplayName,
		BaseURL:         provider.BaseURL,
		APIKey:          provider.APIKey,
		ModelProfileKey: provider.DefaultModelProfileKey,
	}, nil
}
