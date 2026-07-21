package provider

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Test_manualTrackedOutputs pins the diff rule the read-back prune relies on: an output is a tracked override
// iff its assignment_type != NONE OR its display_name differs from the matching input's. This is what lets the
// provider reconstruct exactly the declared subset from any backend response (which is always the full
// one-per-input set) - the losslessness ValidateConfig guarantees, and the basis of stable content hashes.
// Non-manual specs are returned unchanged (their outputs are configured explicitly, not derived).
func Test_manualTrackedOutputs(t *testing.T) {
	str := client.MeshBuildingBlockIOTypeString.Unwrap()
	boolean := client.MeshBuildingBlockIOTypeBoolean.Unwrap()
	none := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.Unwrap()
	summary := client.MeshBuildingBlockDefinitionOutputAssignmentTypeSummary.Unwrap()

	inputs := map[string]*client.MeshBuildingBlockDefinitionInput{
		"approval": {DisplayName: "Approval", Type: boolean},
		"region":   {DisplayName: "Region", Type: str},
		"ticket":   {DisplayName: "Ticket", Type: str},
	}
	// A full backend response: approval overrides display_name, region overrides assignment_type, ticket is a
	// pure derived output (assignment NONE, display_name equal to the input's).
	fullResponse := map[string]client.MeshBuildingBlockDefinitionOutput{
		"approval": {DisplayName: "Approval Output", Type: boolean, AssignmentType: none, DisplayOrder: 0},
		"region":   {DisplayName: "Region", Type: str, AssignmentType: summary, DisplayOrder: 1},
		"ticket":   {DisplayName: "Ticket", Type: str, AssignmentType: none, DisplayOrder: 2},
	}

	manualSpec := client.MeshBuildingBlockDefinitionVersionSpec{
		Implementation: client.MeshBuildingBlockDefinitionImplementation{
			Type:   client.MeshBuildingBlockImplementationTypeManual,
			Manual: &client.MeshBuildingBlockDefinitionManualImplementation{},
		},
		Inputs:  inputs,
		Outputs: fullResponse,
	}

	tracked := manualTrackedOutputs(manualSpec)
	gotKeys := make([]string, 0, len(tracked))
	for key := range tracked {
		gotKeys = append(gotKeys, key)
	}
	slices.Sort(gotKeys)
	assert.Equal(t, []string{"approval", "region"}, gotKeys, "ticket (derived, no override) must be pruned")
	assert.Equal(t, "Approval Output", tracked["approval"].DisplayName)
	assert.Equal(t, summary, tracked["region"].AssignmentType)

	// Non-manual specs are returned unchanged.
	nonManual := manualSpec
	nonManual.Implementation = client.MeshBuildingBlockDefinitionImplementation{}
	assert.Len(t, manualTrackedOutputs(nonManual), 3)
}
