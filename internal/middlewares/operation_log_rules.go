package middlewares

import "github.com/ketches/ketches/internal/db/entities"

type operationLogRule struct {
	Action              string
	ResourceType        string
	ResourceIDParam     string
	Sensitivity         string
	BodyActionField     string
	BodyActionToAction  map[string]string
	BodyActionSensitive map[string]string
}

func operationLogRouteRules() map[string]operationLogRule {
	return map[string]operationLogRule{
		"POST /api/v1/users/sign-in": {
			Action:       "sign_in",
			ResourceType: "session",
			Sensitivity:  entities.OperationLogSensitivityPublic,
		},
		"POST /api/v1/users/sign-up": {
			Action:       "sign_up",
			ResourceType: "user",
			Sensitivity:  entities.OperationLogSensitivityPublic,
		},
		"POST /api/v1/projects": {
			Action:       "create",
			ResourceType: "project",
			Sensitivity:  entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID": {
			Action:          "update",
			ResourceType:    "project",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID": {
			Action:          "delete",
			ResourceType:    "project",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/envs/:envID/apps": {
			Action:          "create",
			ResourceType:    "app",
			ResourceIDParam: "envID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/apps/:appID": {
			Action:          "delete",
			ResourceType:    "app",
			ResourceIDParam: "appID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/apps/:appID/action": {
			Action:          "action",
			ResourceType:    "app",
			ResourceIDParam: "appID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
			BodyActionField: "action",
			BodyActionToAction: map[string]string{
				"deploy":   "deploy",
				"start":    "start",
				"stop":     "stop",
				"rollback": "rollback",
				"update":   "update",
				"redeploy": "redeploy",
				"debug":    "debug",
				"debugOff": "debug_off",
				"delete":   "delete",
			},
			BodyActionSensitive: map[string]string{
				"delete":   entities.OperationLogSensitivitySensitive,
				"rollback": entities.OperationLogSensitivitySensitive,
				"debug":    entities.OperationLogSensitivityInternal,
				"debugOff": entities.OperationLogSensitivityInternal,
			},
		},
		"POST /api/v1/code-repositories/:repoID/builds": {
			Action:          "build",
			ResourceType:    "code_repository",
			ResourceIDParam: "repoID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/code-repositories/:repoID/builds/:buildID/deploy": {
			Action:          "deploy",
			ResourceType:    "code_repository",
			ResourceIDParam: "repoID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/code-repositories/:repoID": {
			Action:          "delete",
			ResourceType:    "code_repository",
			ResourceIDParam: "repoID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
	}
}
