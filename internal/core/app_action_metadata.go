package core

import (
	"sort"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
)

var actionPriority = map[app.AppAction]int{
	app.AppActionDeploy:   10,
	app.AppActionStart:    20,
	app.AppActionStop:     30,
	app.AppActionUpdate:   40,
	app.AppActionRedeploy: 50,
	app.AppActionRollback: 60,
	app.AppActionDebug:    70,
	app.AppActionDebugOff: 80,
	app.AppActionDelete:   90,
}

var actionMetadataMap = map[app.AppAction]models.ActionMetadata{
	app.AppActionDeploy: {
		Action:   string(app.AppActionDeploy),
		Label:    "Deploy",
		Icon:     "rocket",
		Category: "primary",
		Variant:  "default",
	},
	app.AppActionStart: {
		Action:   string(app.AppActionStart),
		Label:    "Start",
		Icon:     "play",
		Category: "primary",
		Variant:  "default",
	},
	app.AppActionStop: {
		Action:   string(app.AppActionStop),
		Label:    "Stop",
		Icon:     "pause",
		Category: "primary",
		Variant:  "destructive",
	},
	app.AppActionUpdate: {
		Action:   string(app.AppActionUpdate),
		Label:    "Update",
		Icon:     "update",
		Category: "primary",
		Variant:  "default",
	},
	app.AppActionRedeploy: {
		Action:   string(app.AppActionRedeploy),
		Label:    "Redeploy",
		Icon:     "cloud-sync",
		Category: "primary",
		Variant:  "outline",
	},
	app.AppActionRollback: {
		Action:   string(app.AppActionRollback),
		Label:    "Rollback",
		Icon:     "cloud-backup",
		Category: "secondary",
		Variant:  "outline",
	},
	app.AppActionDebug: {
		Action:   string(app.AppActionDebug),
		Label:    "Debug",
		Icon:     "bug",
		Category: "secondary",
		Variant:  "outline",
	},
	app.AppActionDebugOff: {
		Action:   string(app.AppActionDebugOff),
		Label:    "Debug Off",
		Icon:     "bug-off",
		Category: "secondary",
		Variant:  "outline",
	},
	app.AppActionDelete: {
		Action:   string(app.AppActionDelete),
		Label:    "Delete",
		Icon:     "trash-2",
		Category: "secondary",
		Variant:  "destructive",
	},
}

func GetAvailableActions(status app.AppStatus) []models.ActionMetadata {
	allowedActions := GetAllowedActions(status)

	sort.Slice(allowedActions, func(i, j int) bool {
		return actionPriority[allowedActions[i]] < actionPriority[allowedActions[j]]
	})

	metadata := make([]models.ActionMetadata, 0, len(allowedActions))
	for _, action := range allowedActions {
		if meta, ok := actionMetadataMap[action]; ok {
			metadata = append(metadata, meta)
		}
	}

	return metadata
}
