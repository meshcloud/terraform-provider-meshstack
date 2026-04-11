package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

func TestParseStringInputToClientValue(t *testing.T) {
	t.Run("keeps plain string value", func(t *testing.T) {
		result := parseStringInputToClientValue("my-secret", client.MESH_BUILDING_BLOCK_IO_TYPE_STRING)

		require.Nil(t, result.Sensitive)
		require.Equal(t, "my-secret", result.Value)
		require.Equal(t, client.MESH_BUILDING_BLOCK_IO_TYPE_STRING, result.ValueType)
	})

	t.Run("decodes json number", func(t *testing.T) {
		result := parseStringInputToClientValue(`16`, client.MESH_BUILDING_BLOCK_IO_TYPE_INTEGER)

		require.Nil(t, result.Sensitive)
		typed, ok := result.Value.(float64)
		require.True(t, ok)
		require.InDelta(t, 16.0, typed, 0.0)
		require.Equal(t, client.MESH_BUILDING_BLOCK_IO_TYPE_INTEGER, result.ValueType)
	})

	t.Run("keeps object-like values as non-sensitive value", func(t *testing.T) {
		result := parseStringInputToClientValue(`{"plaintext":"my-secret"}`, client.MESH_BUILDING_BLOCK_IO_TYPE_CODE)

		require.Nil(t, result.Sensitive)
		typed, ok := result.Value.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "my-secret", typed["plaintext"])
		require.Equal(t, client.MESH_BUILDING_BLOCK_IO_TYPE_CODE, result.ValueType)
	})
}

func TestSelectLatestBuildingBlockRun(t *testing.T) {
	t.Run("prefers highest run number", func(t *testing.T) {
		runs := []client.MeshBuildingBlockRun{
			{
				Metadata: client.MeshBuildingBlockRunMetadata{Uuid: "run-2", CreatedOn: "2024-05-01T12:00:00Z"},
				Spec:     client.MeshBuildingBlockRunSpec{RunNumber: 2, Behavior: "APPLY"},
				Status:   "SUCCEEDED",
			},
			{
				Metadata: client.MeshBuildingBlockRunMetadata{Uuid: "run-7", CreatedOn: "2024-01-01T12:00:00Z"},
				Spec:     client.MeshBuildingBlockRunSpec{RunNumber: 7, Behavior: "APPLY"},
				Status:   "SUCCEEDED",
			},
			{
				Metadata: client.MeshBuildingBlockRunMetadata{Uuid: "run-5", CreatedOn: "2024-06-01T12:00:00Z"},
				Spec:     client.MeshBuildingBlockRunSpec{RunNumber: 5, Behavior: "APPLY"},
				Status:   "SUCCEEDED",
			},
		}

		latest := selectLatestBuildingBlockRun(runs)
		require.NotNil(t, latest)
		require.Equal(t, "run-7", latest.Metadata.Uuid)
	})

	t.Run("uses createdOn as tie-breaker", func(t *testing.T) {
		runs := []client.MeshBuildingBlockRun{
			{
				Metadata: client.MeshBuildingBlockRunMetadata{Uuid: "old", CreatedOn: "2024-05-01T12:00:00Z"},
				Spec:     client.MeshBuildingBlockRunSpec{RunNumber: 7, Behavior: "APPLY"},
				Status:   "SUCCEEDED",
			},
			{
				Metadata: client.MeshBuildingBlockRunMetadata{Uuid: "new", CreatedOn: "2024-05-01T13:00:00Z"},
				Spec:     client.MeshBuildingBlockRunSpec{RunNumber: 7, Behavior: "APPLY"},
				Status:   "SUCCEEDED",
			},
		}

		latest := selectLatestBuildingBlockRun(runs)
		require.NotNil(t, latest)
		require.Equal(t, "new", latest.Metadata.Uuid)
	})
}

func TestOperatorInputWarnings(t *testing.T) {
	assignments := map[string]buildingBlockV3InputAssignment{
		"approval_ticket": {
			AssignmentType: client.MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput.Unwrap(),
			Bucket:         buildingBlockV3InputBucketPlatformOperator,
		},
		"name": {
			AssignmentType: client.MeshBuildingBlockInputAssignmentTypeUserInput.Unwrap(),
			Bucket:         buildingBlockV3InputBucketUser,
		},
	}

	t.Run("warns about missing platform operator inputs", func(t *testing.T) {
		var diags diag.Diagnostics
		addMissingPlatformOperatorInputWarning(&diags, assignments, nil)

		require.Equal(t, 1, diags.WarningsCount())
		warning := diags.Warnings()[0]
		require.Equal(t, "Platform operator inputs missing", warning.Summary())
		require.Contains(t, warning.Detail(), "approval_ticket")
	})
}
