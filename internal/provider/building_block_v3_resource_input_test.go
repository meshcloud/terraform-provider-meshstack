package provider

import (
	"fmt"
	"strings"
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

func TestFormatRunFailureDiagnostics(t *testing.T) {
	ptr := func(s string) *string { return &s }

	t.Run("fallback when no steps available", func(t *testing.T) {
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Creation failed", fmt.Errorf("building block reached FAILED state"), nil)

		require.Equal(t, 1, diags.ErrorsCount())
		require.Contains(t, diags.Errors()[0].Detail(), "Run logs could not be retrieved")
		require.Equal(t, 0, diags.WarningsCount())
	})

	t.Run("fallback when steps are empty", func(t *testing.T) {
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Creation failed", fmt.Errorf("building block reached FAILED state"), &buildingBlockV3LatestRunModel{
			Steps: []buildingBlockV3RunStepModel{},
		})

		require.Equal(t, 1, diags.ErrorsCount())
		require.Contains(t, diags.Errors()[0].Detail(), "Run logs could not be retrieved")
	})

	t.Run("formats step summary and per-step warnings", func(t *testing.T) {
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Creation failed", fmt.Errorf("building block reached FAILED state"), &buildingBlockV3LatestRunModel{
			Uuid:      "run-abc",
			RunNumber: 3,
			Status:    "FAILED",
			Steps: []buildingBlockV3RunStepModel{
				{DisplayName: "Init", Status: "SUCCEEDED"},
				{DisplayName: "Plan", Status: "SUCCEEDED"},
				{DisplayName: "Apply", Status: "FAILED", UserMessage: ptr("terraform apply failed"), SystemMessage: ptr("Error: resource not found\n\ndetails here")},
			},
		})

		require.Equal(t, 1, diags.ErrorsCount())
		errDetail := diags.Errors()[0].Detail()
		require.Contains(t, errDetail, "Run #3 (run-abc)")
		require.Contains(t, errDetail, "Init ✓ SUCCEEDED")
		require.Contains(t, errDetail, "Apply ✗ FAILED")

		require.Equal(t, 2, diags.WarningsCount())
		require.Equal(t, `Step "Apply" — user message`, diags.Warnings()[0].Summary())
		require.Equal(t, "terraform apply failed", diags.Warnings()[0].Detail())
		require.Equal(t, `Step "Apply" — system message`, diags.Warnings()[1].Summary())
		require.Contains(t, diags.Warnings()[1].Detail(), "resource not found")
	})

	t.Run("skips warnings for succeeded steps", func(t *testing.T) {
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Update failed", fmt.Errorf("timeout"), &buildingBlockV3LatestRunModel{
			Uuid:      "run-xyz",
			RunNumber: 1,
			Status:    "FAILED",
			Steps: []buildingBlockV3RunStepModel{
				{DisplayName: "Init", Status: "SUCCEEDED", UserMessage: ptr("all good"), SystemMessage: ptr("detailed logs")},
				{DisplayName: "Apply", Status: "FAILED", UserMessage: ptr("broke")},
			},
		})

		require.Equal(t, 1, diags.ErrorsCount())
		require.Equal(t, 1, diags.WarningsCount())
		require.Equal(t, `Step "Apply" — user message`, diags.Warnings()[0].Summary())
	})

	t.Run("truncates long messages", func(t *testing.T) {
		longMsg := strings.Repeat("x", 5000)
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Failed", fmt.Errorf("err"), &buildingBlockV3LatestRunModel{
			Uuid:      "run-1",
			RunNumber: 1,
			Status:    "FAILED",
			Steps: []buildingBlockV3RunStepModel{
				{DisplayName: "Apply", Status: "FAILED", SystemMessage: &longMsg},
			},
		})

		require.Equal(t, 1, diags.WarningsCount())
		detail := diags.Warnings()[0].Detail()
		require.Contains(t, detail, "... truncated")
		require.LessOrEqual(t, len(detail), maxStepMessageLength+100)
	})

	t.Run("includes warning steps", func(t *testing.T) {
		var diags diag.Diagnostics
		formatRunFailureDiagnostics(&diags, "Failed", fmt.Errorf("err"), &buildingBlockV3LatestRunModel{
			Uuid:      "run-w",
			RunNumber: 2,
			Status:    "FAILED",
			Steps: []buildingBlockV3RunStepModel{
				{DisplayName: "Validate", Status: "WARNING", UserMessage: ptr("drift detected")},
				{DisplayName: "Apply", Status: "FAILED", UserMessage: ptr("error")},
			},
		})

		require.Equal(t, 1, diags.ErrorsCount())
		errDetail := diags.Errors()[0].Detail()
		require.Contains(t, errDetail, "Validate ⚠ WARNING")
		require.Contains(t, errDetail, "Apply ✗ FAILED")

		require.Equal(t, 2, diags.WarningsCount())
		require.Equal(t, `Step "Validate" — user message`, diags.Warnings()[0].Summary())
		require.Equal(t, `Step "Apply" — user message`, diags.Warnings()[1].Summary())
	})
}
