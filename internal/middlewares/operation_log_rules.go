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
			Sensitivity:  entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/users/sign-up": {
			Action:       "sign_up",
			ResourceType: "user",
			Sensitivity:  entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/platform-update/config": {
			Action:       "update",
			ResourceType: "platform_update_config",
			Sensitivity:  entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/platform-update/check": {
			Action:       "check",
			ResourceType: "platform_update_check",
			Sensitivity:  entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/platform-update/rollout": {
			Action:       "rollout",
			ResourceType: "platform_update_rollout",
			Sensitivity:  entities.OperationLogSensitivitySensitive,
		},
		"PUT /api/v1/platform-settings/branding": {
			Action:       "update",
			ResourceType: "platform_branding",
			Sensitivity:  entities.OperationLogSensitivitySensitive,
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
		"POST /api/v1/projects/:projectID/sprints": {
			Action:          "create",
			ResourceType:    "sprint",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID/sprints/:sprintID": {
			Action:          "update",
			ResourceType:    "sprint",
			ResourceIDParam: "sprintID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID/sprints/:sprintID": {
			Action:          "delete",
			ResourceType:    "sprint",
			ResourceIDParam: "sprintID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/projects/:projectID/requirements": {
			Action:          "create",
			ResourceType:    "requirement",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/requirements/:requirementID/children": {
			Action:          "create_child",
			ResourceType:    "requirement",
			ResourceIDParam: "requirementID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID/requirements/:requirementID": {
			Action:          "update",
			ResourceType:    "requirement",
			ResourceIDParam: "requirementID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/requirements/:requirementID/transition": {
			Action:          "transition",
			ResourceType:    "requirement",
			ResourceIDParam: "requirementID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID/requirements/:requirementID": {
			Action:          "delete",
			ResourceType:    "requirement",
			ResourceIDParam: "requirementID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/projects/:projectID/backlog/reorder": {
			Action:          "reorder",
			ResourceType:    "backlog",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/backlog/plan-to-sprint": {
			Action:          "plan",
			ResourceType:    "backlog",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/backlog/return": {
			Action:          "return_to_backlog",
			ResourceType:    "backlog",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/tasks": {
			Action:          "create",
			ResourceType:    "task",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/tasks/:taskID/children": {
			Action:          "create_child",
			ResourceType:    "task",
			ResourceIDParam: "taskID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID/tasks/:taskID": {
			Action:          "update",
			ResourceType:    "task",
			ResourceIDParam: "taskID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID/tasks/:taskID": {
			Action:          "delete",
			ResourceType:    "task",
			ResourceIDParam: "taskID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/projects/:projectID/tasks/:taskID/transition": {
			Action:          "transition",
			ResourceType:    "task",
			ResourceIDParam: "taskID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/test-cases": {
			Action:          "create",
			ResourceType:    "test_case",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID/test-cases/:testCaseID": {
			Action:          "update",
			ResourceType:    "test_case",
			ResourceIDParam: "testCaseID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID/test-cases/:testCaseID": {
			Action:          "delete",
			ResourceType:    "test_case",
			ResourceIDParam: "testCaseID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/projects/:projectID/test-cases/:testCaseID/runs": {
			Action:          "create",
			ResourceType:    "test_run",
			ResourceIDParam: "testCaseID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"POST /api/v1/projects/:projectID/defects": {
			Action:          "create",
			ResourceType:    "defect",
			ResourceIDParam: "projectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"PUT /api/v1/projects/:projectID/defects/:defectID": {
			Action:          "update",
			ResourceType:    "defect",
			ResourceIDParam: "defectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
		"DELETE /api/v1/projects/:projectID/defects/:defectID": {
			Action:          "delete",
			ResourceType:    "defect",
			ResourceIDParam: "defectID",
			Sensitivity:     entities.OperationLogSensitivitySensitive,
		},
		"POST /api/v1/projects/:projectID/defects/:defectID/transition": {
			Action:          "transition",
			ResourceType:    "defect",
			ResourceIDParam: "defectID",
			Sensitivity:     entities.OperationLogSensitivityInternal,
		},
	}
}
