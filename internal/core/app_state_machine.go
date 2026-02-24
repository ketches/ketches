package core

import (
	"errors"
	"fmt"

	"github.com/ketches/ketches/internal/app"
)

var (
	ErrInvalidAction           = errors.New("invalid action")
	ErrInvalidStateTransition  = errors.New("invalid state transition")
	ErrActionNotAllowedInState = errors.New("action not allowed in current state")
)

type StateTransition struct {
	FromStatus         app.AppStatus
	Action             app.AppAction
	IntermediateStatus app.AppStatus
	FinalStatus        app.AppStatus
}

var stateTransitions = map[app.AppStatus]map[app.AppAction]StateTransition{
	app.AppStatusUndeployed: {
		app.AppActionDeploy: {
			FromStatus:         app.AppStatusUndeployed,
			Action:             app.AppActionDeploy,
			IntermediateStatus: app.AppStatusStarting,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusUndeployed,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusStarting: {
		app.AppActionRedeploy: {
			FromStatus:         app.AppStatusStarting,
			Action:             app.AppActionRedeploy,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusStarting,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusRunning: {
		app.AppActionStop: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionStop,
			IntermediateStatus: app.AppStatusStopping,
			FinalStatus:        app.AppStatusStopped,
		},
		app.AppActionUpdate: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionUpdate,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionRedeploy: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionRedeploy,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionRollback: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionRollback,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDebug: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionDebug,
			IntermediateStatus: app.AppStatusDebugging,
			FinalStatus:        app.AppStatusDebugging,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusRunning,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusStopping: {
		app.AppActionDelete: {
			FromStatus:         app.AppStatusStopping,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusStopped: {
		app.AppActionStart: {
			FromStatus:         app.AppStatusStopped,
			Action:             app.AppActionStart,
			IntermediateStatus: app.AppStatusStarting,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusStopped,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusUpdating: {
		app.AppActionUpdate: {
			FromStatus:         app.AppStatusUpdating,
			Action:             app.AppActionUpdate,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDebug: {
			FromStatus:         app.AppStatusUpdating,
			Action:             app.AppActionDebug,
			IntermediateStatus: app.AppStatusDebugging,
			FinalStatus:        app.AppStatusDebugging,
		},
		app.AppActionRedeploy: {
			FromStatus:         app.AppStatusUpdating,
			Action:             app.AppActionRedeploy,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionStop: {
			FromStatus:         app.AppStatusUpdating,
			Action:             app.AppActionStop,
			IntermediateStatus: app.AppStatusStopping,
			FinalStatus:        app.AppStatusStopped,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusUpdating,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusAbnormal: {
		app.AppActionUpdate: {
			FromStatus:         app.AppStatusAbnormal,
			Action:             app.AppActionUpdate,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionRollback: {
			FromStatus:         app.AppStatusAbnormal,
			Action:             app.AppActionRollback,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusAbnormal,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusDebugging: {
		app.AppActionDebugOff: {
			FromStatus:         app.AppStatusDebugging,
			Action:             app.AppActionDebugOff,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusDebugging,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusCompleted: {
		app.AppActionDelete: {
			FromStatus:         app.AppStatusCompleted,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
	app.AppStatusUnknown: {
		app.AppActionUpdate: {
			FromStatus:         app.AppStatusUnknown,
			Action:             app.AppActionUpdate,
			IntermediateStatus: app.AppStatusUpdating,
			FinalStatus:        app.AppStatusRunning,
		},
		app.AppActionDelete: {
			FromStatus:         app.AppStatusUnknown,
			Action:             app.AppActionDelete,
			IntermediateStatus: "",
			FinalStatus:        "",
		},
	},
}

func ValidateStateTransition(currentStatus app.AppStatus, action app.AppAction) (*StateTransition, error) {
	statusTransitions, ok := stateTransitions[currentStatus]
	if !ok {
		return nil, fmt.Errorf("%w: no transitions defined for status %s", ErrInvalidStateTransition, currentStatus)
	}

	transition, ok := statusTransitions[action]
	if !ok {
		return nil, fmt.Errorf("%w: action '%s' not allowed in status '%s'", ErrActionNotAllowedInState, action, currentStatus)
	}

	return &transition, nil
}

func GetAllowedActions(status app.AppStatus) []app.AppAction {
	transitions, ok := stateTransitions[status]
	if !ok {
		return []app.AppAction{}
	}

	actions := make([]app.AppAction, 0, len(transitions))
	for action := range transitions {
		actions = append(actions, action)
	}
	return actions
}

func GetIntermediateStatus(currentStatus app.AppStatus, action app.AppAction) (app.AppStatus, error) {
	transition, err := ValidateStateTransition(currentStatus, action)
	if err != nil {
		return "", err
	}
	return transition.IntermediateStatus, nil
}

func GetFinalStatus(currentStatus app.AppStatus, action app.AppAction) (app.AppStatus, error) {
	transition, err := ValidateStateTransition(currentStatus, action)
	if err != nil {
		return "", err
	}
	return transition.FinalStatus, nil
}
