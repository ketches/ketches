package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

type builderResolvedExecutionSelection struct {
	ExecutorPolicyKey        string
	ExecutorKind             entities.BuilderExecutorHandleKind
	ExecutionImageProfileKey string
	ExecutionImageRef        string
}

func ResolveBuilderExecutionSelection(ctx context.Context, run *entities.BuilderRun) (*builderResolvedExecutionSelection, error) {
	if run == nil {
		return nil, fmt.Errorf("builder run is required")
	}

	catalog, err := loadBuilderExecutionCatalog(app.Config)
	if err != nil {
		return nil, err
	}

	resolvedPolicyKey := strings.TrimSpace(stringPointerValue(run.ExecutorPolicyKey))
	if resolvedPolicyKey == "" {
		plannedPolicyKey := strings.TrimSpace(stringPointerValue(run.PlannedExecutorPolicyKey))
		if _, ok := catalog.ExecutorPolicies[plannedPolicyKey]; ok && plannedPolicyKey != "" {
			resolvedPolicyKey = plannedPolicyKey
		} else {
			resolvedPolicyKey = catalog.DefaultExecutorPolicyKey
		}
	}

	policy, ok := catalog.ExecutorPolicies[resolvedPolicyKey]
	if !ok {
		return nil, fmt.Errorf("unknown builder executor policy key %q", resolvedPolicyKey)
	}

	resolvedImageProfileKey := strings.TrimSpace(stringPointerValue(run.ExecutionImageProfileKey))
	if resolvedImageProfileKey == "" {
		plannedImageProfileKey := strings.TrimSpace(stringPointerValue(run.PlannedImageProfileKey))
		if _, ok := catalog.ImageProfiles[plannedImageProfileKey]; ok && plannedImageProfileKey != "" {
			resolvedImageProfileKey = plannedImageProfileKey
		} else {
			resolvedImageProfileKey = policy.ImageProfileKey
		}
	}

	imageProfile, ok := catalog.ImageProfiles[resolvedImageProfileKey]
	if !ok {
		return nil, fmt.Errorf("unknown builder image profile key %q", resolvedImageProfileKey)
	}

	return &builderResolvedExecutionSelection{
		ExecutorPolicyKey:        resolvedPolicyKey,
		ExecutorKind:             entities.BuilderExecutorHandleKind(policy.ExecutorKind),
		ExecutionImageProfileKey: resolvedImageProfileKey,
		ExecutionImageRef:        imageProfile.Image,
	}, nil
}

func PersistBuilderRunExecutionSelection(ctx context.Context, runID string, selection *builderResolvedExecutionSelection) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("builder run id is required")
	}
	if selection == nil {
		return fmt.Errorf("builder execution selection is required")
	}

	return db.DB.WithContext(ctx).Model(&entities.BuilderRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"executor_policy_key":         nullableStringValue(builderStringPtr(selection.ExecutorPolicyKey)),
			"execution_image_profile_key": nullableStringValue(builderStringPtr(selection.ExecutionImageProfileKey)),
			"execution_image_ref":         nullableStringValue(builderStringPtr(selection.ExecutionImageRef)),
		}).Error
}
