package services

import (
	"context"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromoteBuilderSessionExportToInitialBuild(t *testing.T) {
	originalPromote := promoteBuilderExportToRepositoryForBuild
	originalCreateBuildSetting := createBuilderExportBuildSetting
	originalTriggerBuild := triggerBuilderExportInitialBuild
	t.Cleanup(func() {
		promoteBuilderExportToRepositoryForBuild = originalPromote
		createBuilderExportBuildSetting = originalCreateBuildSetting
		triggerBuilderExportInitialBuild = originalTriggerBuild
	})

	promoteBuilderExportToRepositoryForBuild = func(ctx context.Context, projectID, sessionID, exportID string, req *models.PromoteBuilderExportToCodeRepositoryRequest) (*models.BuilderExportPromotionResponse, error) {
		return &models.BuilderExportPromotionResponse{
			Export: models.BuilderExportResponse{
				ID:        exportID,
				SessionID: sessionID,
				Kind:      "session_archive",
				Status:    "ready",
				FileName:  "builder-output.tar.gz",
			},
			Repository: models.CodeRepositoryResponse{
				ID:         "repo-1",
				ProjectID:  projectID,
				Name:       req.Name,
				Slug:       "builder-export-repo",
				GitRepoURL: req.GitRepoURL,
				CreatedAt:  time.Now().UTC(),
				UpdatedAt:  time.Now().UTC(),
			},
		}, nil
	}
	createBuilderExportBuildSetting = func(repoID string, req *models.CreateBuildSettingRequest) (*BuildSettingWithRegistry, error) {
		return &BuildSettingWithRegistry{
			BuildSetting: entities.BuildSetting{
				ID:               "setting-1",
				CodeRepositoryID: &repoID,
				Name:             req.Name,
				GitRef:           req.GitRef,
				DockerfilePath:   req.DockerfilePath,
				BuildContext:     req.BuildContext,
				ImageName:        req.ImageName,
				RegistryID:       req.RegistryID,
				Platforms:        "linux/amd64",
			},
		}, nil
	}
	triggerBuilderExportInitialBuild = func(repoID, userID string, req *models.TriggerCodeRepositoryBuildRequest) (*entities.Build, error) {
		return &entities.Build{
			ID:             "build-1",
			BuildSettingID: req.BuildSettingID,
			BuildNumber:    1,
			Status:         entities.BuildStatusPending,
			BuildEnvID:     req.BuildEnvID,
			GitRef:         req.GitRef,
			CreatedAt:      time.Now().UTC(),
		}, nil
	}

	resp, err := PromoteBuilderSessionExportToInitialBuild(context.Background(), "project-1", "session-1", "export-1", "user-1", &models.PromoteBuilderExportToInitialBuildRequest{
		Name:       "Builder Export Repo",
		GitRepoURL: "https://example.com/demo.git",
		BuildEnvID: "env-1",
		RegistryID: "registry-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "repo-1", resp.Promotion.Repository.ID)
	assert.Equal(t, "setting-1", resp.BuildSetting.ID)
	assert.Equal(t, "builder-default", resp.BuildSetting.Name)
	assert.Equal(t, "build-1", resp.Build.ID)
	assert.Equal(t, "env-1", resp.Build.BuildEnvID)
}
