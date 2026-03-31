package services

import (
	"context"
	"testing"

	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployBuilderExportBuild(t *testing.T) {
	originalDeployCodeRepositoryBuild := deployCodeRepositoryBuildForBuilderExport
	t.Cleanup(func() {
		deployCodeRepositoryBuildForBuilderExport = originalDeployCodeRepositoryBuild
	})

	deployCodeRepositoryBuildForBuilderExport = func(ctx context.Context, repoID, buildID string, req *models.DeployCodeRepositoryBuildRequest) (*models.AppContext, error) {
		assert.Equal(t, "repo-1", repoID)
		assert.Equal(t, "build-1", buildID)
		assert.Equal(t, "env-1", req.TargetEnvID)
		assert.Equal(t, "app-1", req.AppID)
		return &models.AppContext{}, nil
	}

	appCtx, err := DeployBuilderExportBuild(context.Background(), &models.DeployBuilderExportBuildRequest{
		RepositoryID: "repo-1",
		BuildID:      "build-1",
		TargetEnvID:  "env-1",
		AppID:        "app-1",
	})
	require.NoError(t, err)
	require.NotNil(t, appCtx)
}
