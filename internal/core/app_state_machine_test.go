package core

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
)

func TestValidateStateTransition(t *testing.T) {
	tests := []struct {
		name          string
		currentStatus app.AppStatus
		action        app.AppAction
		wantErr       bool
		wantInter     app.AppStatus
		wantFinal     app.AppStatus
	}{
		{
			name:          "Undeployed to Deploy",
			currentStatus: app.AppStatusUndeployed,
			action:        app.AppActionDeploy,
			wantErr:       false,
			wantInter:     app.AppStatusStarting,
			wantFinal:     app.AppStatusRunning,
		},
		{
			name:          "Running to Stop",
			currentStatus: app.AppStatusRunning,
			action:        app.AppActionStop,
			wantErr:       false,
			wantInter:     app.AppStatusStopping,
			wantFinal:     app.AppStatusStopped,
		},
		{
			name:          "Stopped to Start",
			currentStatus: app.AppStatusStopped,
			action:        app.AppActionStart,
			wantErr:       false,
			wantInter:     app.AppStatusStarting,
			wantFinal:     app.AppStatusRunning,
		},
		{
			name:          "Undeployed to Stop (Invalid)",
			currentStatus: app.AppStatusUndeployed,
			action:        app.AppActionStop,
			wantErr:       true,
		},
		{
			name:          "Running to Debug",
			currentStatus: app.AppStatusRunning,
			action:        app.AppActionDebug,
			wantErr:       false,
			wantInter:     app.AppStatusDebugging,
			wantFinal:     app.AppStatusDebugging,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateStateTransition(tt.currentStatus, tt.action)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateStateTransition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.IntermediateStatus != tt.wantInter {
					t.Errorf("ValidateStateTransition() IntermediateStatus = %v, want %v", got.IntermediateStatus, tt.wantInter)
				}
				if got.FinalStatus != tt.wantFinal {
					t.Errorf("ValidateStateTransition() FinalStatus = %v, want %v", got.FinalStatus, tt.wantFinal)
				}
			}
		})
	}
}
