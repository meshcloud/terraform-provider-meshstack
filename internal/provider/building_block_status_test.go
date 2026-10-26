package provider

import (
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/client/types/enum"
	"github.com/stretchr/testify/assert"
)

func TestNewBuildingBlockStatusModel_Succeeded(t *testing.T) {
	tests := []struct {
		health enum.Entry[client.BuildingBlockHealthStatus]
		action enum.Entry[client.BuildingBlockAction]
		want   bool
	}{
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionNoAction, true},
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionUpgradeAvailable, true},
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionReviewDrift, true},
		// health_status keeps the previous outcome while a new run waits or runs.
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionApproveRun, false},
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionPending, false},
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionReviewErrors, false},
		{client.BuildingBlockHealthStatusSucceeded, client.BuildingBlockActionReviewAbortedRun, false},
		{client.BuildingBlockHealthStatusNone, client.BuildingBlockActionNoAction, false},
		{client.BuildingBlockHealthStatusFailed, client.BuildingBlockActionReviewErrors, false},
	}
	for _, tt := range tests {
		t.Run(tt.health.String()+"_"+tt.action.String(), func(t *testing.T) {
			got := newBuildingBlockStatusModel(client.MeshBuildingBlockV2Status{HealthStatus: tt.health, Action: tt.action})
			assert.Equal(t, tt.want, got.Succeeded)
		})
	}
}
