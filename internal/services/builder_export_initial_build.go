package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/ketches/ketches/internal/models"
)

var promoteBuilderExportToRepositoryForBuild = PromoteBuilderSessionExportToCodeRepository
var createBuilderExportBuildSetting = CreateRepoBuildSetting
var triggerBuilderExportInitialBuild = TriggerCodeRepositoryBuild

func PromoteBuilderSessionExportToInitialBuild(ctx context.Context, projectID, sessionID, exportID, userID string, req *models.PromoteBuilderExportToInitialBuildRequest) (*models.BuilderExportInitialBuildPromotionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("builder export initial build request is required")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf("builder export initial build user is required")
	}

	promotion, err := promoteBuilderExportToRepositoryForBuild(ctx, projectID, sessionID, exportID, &models.PromoteBuilderExportToCodeRepositoryRequest{
		Name:        req.Name,
		Slug:        req.Slug,
		GitRepoURL:  req.GitRepoURL,
		GitUsername: req.GitUsername,
		GitPassword: req.GitPassword,
	})
	if err != nil {
		return nil, err
	}

	buildSetting, err := createBuilderExportBuildSetting(promotion.Repository.ID, &models.CreateBuildSettingRequest{
		Name:           defaultBuilderExportBuildSettingName(req.BuildSettingName),
		GitRef:         defaultBuilderExportGitRef(req.GitRef),
		DockerfilePath: defaultBuilderExportDockerfilePath(req.DockerfilePath),
		BuildContext:   defaultBuilderExportBuildContext(req.BuildContext),
		ImageName:      defaultBuilderExportImageName(req.ImageName, promotion.Repository.Slug),
		RegistryID:     req.RegistryID,
	})
	if err != nil {
		return nil, err
	}

	build, err := triggerBuilderExportInitialBuild(promotion.Repository.ID, userID, &models.TriggerCodeRepositoryBuildRequest{
		BuildSettingID: buildSetting.ID,
		BuildEnvID:     req.BuildEnvID,
		GitRef:         defaultBuilderExportGitRef(req.GitRef),
	})
	if err != nil {
		return nil, err
	}

	return &models.BuilderExportInitialBuildPromotionResponse{
		Promotion:    *promotion,
		BuildSetting: ToBuildSettingResponse(buildSetting),
		Build:        ToBuildResponse(ctx, build),
	}, nil
}

func defaultBuilderExportBuildSettingName(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "builder-default"
}

func defaultBuilderExportImageName(value, repositorySlug string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if strings.TrimSpace(repositorySlug) != "" {
		return strings.TrimSpace(repositorySlug)
	}
	return "builder-export"
}

func defaultBuilderExportDockerfilePath(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "Dockerfile"
}

func defaultBuilderExportBuildContext(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "."
}

func defaultBuilderExportGitRef(value string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "main"
}
