package services

import (
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/models"
)

var deployCodeRepositoryBuildForBuilderExport = func(ctx context.Context, repoID, buildID string, req *models.DeployCodeRepositoryBuildRequest) (*models.AppContext, error) {
	_, appCtx, err := DeployCodeRepositoryBuild(ctx, repoID, buildID, req)
	return appCtx, err
}

func DeployBuilderExportBuild(ctx context.Context, req *models.DeployBuilderExportBuildRequest) (*models.AppContext, error) {
	if req == nil {
		return nil, fmt.Errorf("builder export deploy request is required")
	}

	appCtx, err := deployCodeRepositoryBuildForBuilderExport(ctx, req.RepositoryID, req.BuildID, &models.DeployCodeRepositoryBuildRequest{
		TargetEnvID: req.TargetEnvID,
		AppID:       req.AppID,
		Name:        req.Name,
		Slug:        req.Slug,
	})
	if err != nil {
		return nil, err
	}

	return appCtx, nil
}
