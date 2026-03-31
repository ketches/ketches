package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

var writeBuilderExportSnapshotArchive = func(ctx context.Context, snapshot *entities.BuilderOutputSnapshot, writer io.Writer) error {
	return WriteBuilderOutputSnapshotArchive(ctx, snapshot, writer)
}

var writeBuilderExportWorkspaceArchive = func(ctx context.Context, projectID, sessionID string, writer io.Writer) error {
	return DownloadBuilderWorkspace(ctx, projectID, sessionID, writer)
}

func CreateBuilderSessionExport(ctx context.Context, projectID, sessionID, createdBy string) (*models.BuilderExportResponse, error) {
	if strings.TrimSpace(createdBy) == "" {
		return nil, errors.New("builder export creator is required")
	}

	session, err := loadBuilderSession(db.DB.WithContext(ctx), projectID, sessionID)
	if err != nil {
		return nil, err
	}

	var (
		runID       *string
		workspaceID *string
		snapshotID  *string
		sourceRoot  string
		fileName    string
	)

	snapshot, err := GetLatestSuccessfulBuilderOutputSnapshot(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	if snapshot != nil {
		runID = builderStringPtr(snapshot.RunID)
		workspaceID = builderStringPtr(snapshot.WorkspaceID)
		snapshotID = builderStringPtr(snapshot.ID)
		sourceRoot = snapshot.OutputRoot
		fileName = fmt.Sprintf("builder-output-%s.tar.gz", snapshot.RunID)
	} else {
		workspace, err := getBuilderWorkspaceForProjectSession(ctx, projectID, session.ID)
		if err != nil {
			return nil, err
		}
		workspaceID = builderStringPtr(workspace.ID)
		sourceRoot = workspace.WorkspaceRoot
		fileName = fmt.Sprintf("builder-workspace-%s.tar.gz", workspace.SessionID)
	}

	export := &entities.BuilderExport{
		ID:           uuid.New(),
		SessionID:    session.ID,
		RunID:        runID,
		WorkspaceID:  workspaceID,
		SnapshotID:   snapshotID,
		Kind:         "session_archive",
		Status:       entities.BuilderExportStatusReady,
		FileName:     fileName,
		StoragePath:  filepath.ToSlash(filepath.Join("builder-exports", session.ID, fileName)),
		SourceRoot:   sourceRoot,
		FileCount:    0,
		SizeBytes:    0,
		MetadataJSON: "",
		CreatedBy:    createdBy,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := db.DB.WithContext(ctx).Create(export).Error; err != nil {
		return nil, err
	}

	return toBuilderExportResponse(export), nil
}

func ListBuilderSessionExports(ctx context.Context, projectID, sessionID string) ([]models.BuilderExportResponse, error) {
	session, err := loadBuilderSession(db.DB.WithContext(ctx), projectID, sessionID)
	if err != nil {
		return nil, err
	}

	var exports []entities.BuilderExport
	if err := db.DB.WithContext(ctx).
		Where("session_id = ?", session.ID).
		Order("created_at DESC, id DESC").
		Find(&exports).Error; err != nil {
		return nil, err
	}

	responses := make([]models.BuilderExportResponse, 0, len(exports))
	for i := range exports {
		responses = append(responses, *toBuilderExportResponse(&exports[i]))
	}
	return responses, nil
}

func DownloadBuilderSessionExport(ctx context.Context, projectID, sessionID, exportID string, writer io.Writer) error {
	if writer == nil {
		return errors.New("builder export archive writer is required")
	}

	session, err := loadBuilderSession(db.DB.WithContext(ctx), projectID, sessionID)
	if err != nil {
		return err
	}

	var export entities.BuilderExport
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND session_id = ?", exportID, session.ID).
		First(&export).Error; err != nil {
		return err
	}

	if export.SnapshotID != nil && strings.TrimSpace(*export.SnapshotID) != "" {
		var snapshot entities.BuilderOutputSnapshot
		if err := db.DB.WithContext(ctx).
			Where("id = ? AND session_id = ?", *export.SnapshotID, session.ID).
			First(&snapshot).Error; err != nil {
			return err
		}
		return writeBuilderExportSnapshotArchive(ctx, &snapshot, writer)
	}

	return writeBuilderExportWorkspaceArchive(ctx, projectID, sessionID, writer)
}

func GetBuilderSessionExportPromotionPlan(ctx context.Context, projectID, sessionID, exportID string) (*models.BuilderExportPromotionPlanResponse, error) {
	session, err := loadBuilderSession(db.DB.WithContext(ctx), projectID, sessionID)
	if err != nil {
		return nil, err
	}

	var export entities.BuilderExport
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND session_id = ?", exportID, session.ID).
		First(&export).Error; err != nil {
		return nil, err
	}

	plannedProjectKind := ""
	if export.RunID != nil && strings.TrimSpace(*export.RunID) != "" {
		var run entities.BuilderRun
		if err := db.DB.WithContext(ctx).Where("id = ?", *export.RunID).First(&run).Error; err == nil {
			plannedProjectKind = stringPointerValue(run.PlannedProjectKind)
		}
	}

	sourceKind := "workspace_source"
	if export.SnapshotID != nil && strings.TrimSpace(*export.SnapshotID) != "" {
		sourceKind = "snapshot_output"
	}

	repositoryName := strings.TrimSpace(session.Title)
	if repositoryName == "" {
		repositoryName = "Builder Export"
	}
	repositorySlug := RepoSlugFromName(repositoryName)
	imageName := repositorySlug

	missingRequirements := make([]string, 0, 2)
	canTriggerInitialBuild := sourceKind == "workspace_source"
	if sourceKind != "workspace_source" {
		missingRequirements = append(missingRequirements, "workspace source export is required for initial build promotion")
	}
	if !builderSessionHasWorkspaceDockerfile(ctx, session.ID) {
		canTriggerInitialBuild = false
		missingRequirements = append(missingRequirements, "Dockerfile is required in the workspace export")
	}

	return &models.BuilderExportPromotionPlanResponse{
		Export:                    *toBuilderExportResponse(&export),
		SourceKind:                sourceKind,
		PlannedProjectKind:        plannedProjectKind,
		SuggestedRepositoryName:   repositoryName,
		SuggestedRepositorySlug:   repositorySlug,
		SuggestedBuildEnvID:       session.BuildEnvID,
		SuggestedBuildSettingName: "builder-default",
		SuggestedImageName:        imageName,
		SuggestedDockerfilePath:   "Dockerfile",
		SuggestedBuildContext:     ".",
		CanTriggerInitialBuild:    canTriggerInitialBuild,
		RequiresRegistrySelection: true,
		MissingRequirements:       missingRequirements,
	}, nil
}

func builderSessionHasWorkspaceDockerfile(ctx context.Context, sessionID string) bool {
	var count int64
	if err := db.DB.WithContext(ctx).
		Model(&entities.BuilderArtifact{}).
		Where("session_id = ? AND kind = ? AND path = ?", sessionID, entities.BuilderArtifactKindWorkspaceFile, "Dockerfile").
		Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

func toBuilderExportResponse(export *entities.BuilderExport) *models.BuilderExportResponse {
	if export == nil {
		return nil
	}

	return &models.BuilderExportResponse{
		ID:           export.ID,
		SessionID:    export.SessionID,
		RunID:        stringPointerValue(export.RunID),
		WorkspaceID:  stringPointerValue(export.WorkspaceID),
		SnapshotID:   stringPointerValue(export.SnapshotID),
		Kind:         export.Kind,
		Status:       string(export.Status),
		FileName:     export.FileName,
		StoragePath:  export.StoragePath,
		SourceRoot:   export.SourceRoot,
		FileCount:    export.FileCount,
		SizeBytes:    export.SizeBytes,
		MetadataJSON: export.MetadataJSON,
		ErrorMessage: export.ErrorMessage,
		CreatedBy:    export.CreatedBy,
		CreatedAt:    export.CreatedAt,
		UpdatedAt:    export.UpdatedAt,
	}
}
