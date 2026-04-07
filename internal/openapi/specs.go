package openapi

import (
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

type exportAppsQuery struct {
	Format string `form:"format"`
	AppIDs string `form:"app_ids"`
}

func defaultOperationSpecs() []OperationSpec {
	return []OperationSpec{
		{
			Method:       "GET",
			Path:         "/api/v1/users/sign-up/config",
			ResponseBody: models.PublicSignUpSettingsResponse{},
		},
		{
			Method:       "POST",
			Path:         "/api/v1/users/sign-up/verification-code",
			RequestBody:  models.SignUpVerificationCodeRequest{},
			ResponseBody: models.SignUpVerificationCodeResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/users/sign-up",
			RequestBody:   models.SignUpRequest{},
			ResponseBody:  models.UserResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "POST",
			Path:         "/api/v1/users/sign-in",
			RequestBody:  models.SignInRequest{},
			ResponseBody: models.SignInResponse{},
		},
		{
			Method:       "POST",
			Path:         "/api/v1/users/refresh-token",
			ResponseBody: models.SignInResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/users/logout",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/envs/{envID}/apps",
			Query:        models.PaginationRequest{},
			ResponseBody: models.ListAppResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/envs/{envID}/apps/simple",
			ResponseBody: []models.SimpleApp{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/envs/{envID}/apps",
			RequestBody:   models.CreateAppRequest{},
			ResponseBody:  models.AppResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}",
			ResponseBody: models.AppResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/apps/{appID}",
			SuccessStatus: 204,
		},
		{
			Method:        "POST",
			Path:          "/api/v1/apps/batch-delete",
			RequestBody:   models.BatchDeleteAppsRequest{},
			SuccessStatus: 204,
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/basic",
			RequestBody:  models.UpdateBasicInfoRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/image",
			RequestBody:  models.UpdateAppImageRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/image-tags",
			ResponseBody: models.AppImageTagsResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/replicas",
			RequestBody:  models.UpdateAppReplicasRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/resources",
			RequestBody:  models.UpdateAppResourcesRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/auto-scaling",
			RequestBody:  models.UpdateAppAutoScalingRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/health",
			RequestBody:  models.UpdateAppHealthRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/scheduling",
			RequestBody:  models.UpdateAppSchedulingRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "PATCH",
			Path:         "/api/v1/apps/{appID}/command",
			RequestBody:  models.UpdateAppCommandRequest{},
			ResponseBody: models.AppResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/available-actions",
			ResponseBody: models.AvailableActionsResponse{},
		},
		{
			Method:       "POST",
			Path:         "/api/v1/apps/{appID}/action",
			RequestBody:  models.AppActionRequest{},
			ResponseBody: models.AppActionResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/instances",
			ResponseBody: []models.AppInstanceResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/apps/{appID}/instances/{instanceName}",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/instances/{instanceName}/events",
			ResponseBody: []models.AppInstanceEventResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/volumes",
			ResponseBody: []models.AppVolumeResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/apps/{appID}/volumes",
			RequestBody:   models.CreateVolumeRequest{},
			ResponseBody:  models.AppVolumeResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "PUT",
			Path:         "/api/v1/volumes/{id}",
			RequestBody:  models.UpdateVolumeRequest{},
			ResponseBody: models.AppVolumeResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/volumes/{id}",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/env-vars",
			ResponseBody: []models.AppEnvVarResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/apps/{appID}/env-vars",
			RequestBody:   models.AppEnvVarRequest{},
			ResponseBody:  models.AppEnvVarResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "PUT",
			Path:         "/api/v1/env-vars/{id}",
			RequestBody:  models.AppEnvVarRequest{},
			ResponseBody: models.AppEnvVarResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/env-vars/{id}",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/config-files",
			ResponseBody: []models.AppConfigFileResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/apps/{appID}/config-files",
			RequestBody:   models.CreateConfigFileRequest{},
			ResponseBody:  models.AppConfigFileResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "PUT",
			Path:         "/api/v1/config-files/{id}",
			RequestBody:  models.UpdateConfigFileRequest{},
			ResponseBody: models.AppConfigFileResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/config-files/{id}",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/gateways",
			ResponseBody: []models.AppGatewayResponse{},
		},
		{
			Method:        "POST",
			Path:          "/api/v1/apps/{appID}/gateways",
			RequestBody:   models.CreateGatewayRequest{},
			ResponseBody:  models.AppGatewayResponse{},
			SuccessStatus: 201,
		},
		{
			Method:       "PUT",
			Path:         "/api/v1/gateways/{id}",
			RequestBody:  models.UpdateGatewayRequest{},
			ResponseBody: models.AppGatewayResponse{},
		},
		{
			Method:        "DELETE",
			Path:          "/api/v1/gateways/{id}",
			SuccessStatus: 204,
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/topology",
			ResponseBody: models.AppTopologyResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/topology/nodes/{nodeID}/resource-yaml",
			ResponseBody: models.AppTopologyResourceYAMLResponse{},
		},
		{
			Method:       "POST",
			Path:         "/api/v1/envs/{envID}/apps/import",
			RequestBody:  models.AppImportRequest{},
			ResponseBody: services.ImportResult{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/apps/{appID}/export",
			Query:        exportAppsQuery{},
			ResponseBody: models.AppExportResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/v1/envs/{envID}/apps/export",
			Query:        exportAppsQuery{},
			ResponseBody: models.AppExportResponse{},
		},
	}
}
