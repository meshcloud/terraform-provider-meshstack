package provider

import (
	"errors"
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	clientTypes "github.com/meshcloud/meshstack-cli/client/types"
	"github.com/meshcloud/meshstack-cli/client/types/enum"
	"github.com/stretchr/testify/require"
)

func Test_changedLockedInputs(t *testing.T) {
	planned := func(value any) *client.MeshBuildingBlockInput {
		return &client.MeshBuildingBlockInput{Value: clientTypes.SecretOrAny{Y: value}}
	}
	plannedSecret := &client.MeshBuildingBlockInput{IsSensitive: true}
	current := func(assignmentType enum.Entry[client.MeshBuildingBlockInputAssignmentType], updateableByConsumer *bool, value *string) buildingBlockAllInput {
		return buildingBlockAllInput{Value: value, AssignmentType: assignmentType, UpdateableByConsumer: updateableByConsumer}
	}
	userInput := client.MeshBuildingBlockInputAssignmentTypeUserInput
	locked, updateable := new(false), new(true)

	tests := []struct {
		name    string
		planned map[string]*client.MeshBuildingBlockInput
		current map[string]buildingBlockAllInput
		rotated bool
		want    []string
	}{
		{
			name:    "a changed locked input is reported",
			planned: map[string]*client.MeshBuildingBlockInput{"env": planned("prod")},
			current: map[string]buildingBlockAllInput{"env": current(userInput, locked, new(`"dev"`))},
			want:    []string{"env"},
		},
		{
			name:    "an unchanged locked input is not reported",
			planned: map[string]*client.MeshBuildingBlockInput{"size": planned(float64(16)), "tags": planned(map[string]any{"a": "b"})},
			current: map[string]buildingBlockAllInput{"size": current(userInput, locked, new(`16`)), "tags": current(userInput, locked, new(`{"a":"b"}`))},
		},
		{
			name:    "a locked input that gets its first value is reported",
			planned: map[string]*client.MeshBuildingBlockInput{"env": planned("prod")},
			current: map[string]buildingBlockAllInput{"env": current(userInput, locked, nil)},
			want:    []string{"env"},
		},
		{
			name:    "a changed updateable input is not reported",
			planned: map[string]*client.MeshBuildingBlockInput{"env": planned("prod")},
			current: map[string]buildingBlockAllInput{"env": current(userInput, updateable, new(`"dev"`))},
		},
		{
			name:    "an input without the flag from meshStack is not reported",
			planned: map[string]*client.MeshBuildingBlockInput{"env": planned("prod")},
			current: map[string]buildingBlockAllInput{"env": current(userInput, nil, new(`"dev"`))},
		},
		{
			name:    "an operator input is not reported, a replacement does not let a consumer set it",
			planned: map[string]*client.MeshBuildingBlockInput{"quota": planned("large")},
			current: map[string]buildingBlockAllInput{"quota": current(client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput, locked, new(`"small"`))},
		},
		{
			name:    "an input the building block does not have yet is not reported",
			planned: map[string]*client.MeshBuildingBlockInput{"new": planned("value")},
			current: map[string]buildingBlockAllInput{},
		},
		{
			name:    "a rotated locked secret is reported",
			planned: map[string]*client.MeshBuildingBlockInput{"token": plannedSecret},
			current: map[string]buildingBlockAllInput{"token": current(userInput, locked, nil)},
			rotated: true,
			want:    []string{"token"},
		},
		{
			name:    "a locked secret that is not rotated is not reported",
			planned: map[string]*client.MeshBuildingBlockInput{"token": plannedSecret},
			current: map[string]buildingBlockAllInput{"token": current(userInput, locked, nil)},
		},
		{
			name:    "changed locked inputs are reported in a stable order",
			planned: map[string]*client.MeshBuildingBlockInput{"b": planned("x"), "a": planned("x"), "c": planned("x")},
			current: map[string]buildingBlockAllInput{
				"a": current(userInput, locked, new(`"y"`)),
				"b": current(userInput, locked, new(`"y"`)),
				"c": current(userInput, locked, new(`"y"`)),
			},
			want: []string{"a", "b", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := changedLockedInputs(tt.planned, tt.current, func(string) bool { return tt.rotated })
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Test_lockedInputUpdateRejected(t *testing.T) {
	summary, detail := lockedInputUpdateRejected([]string{"environment", "region"}, errors.New("403 Forbidden"))

	require.Equal(t, "Building block input is not updateable by consumers", summary)
	require.Contains(t, detail, `"environment", "region"`)
	require.Contains(t, detail, "tofu taint <resource address>")
	require.Contains(t, detail, lockedInputChangeDocsSection)
	require.Contains(t, detail, "Underlying error: 403 Forbidden")
}
