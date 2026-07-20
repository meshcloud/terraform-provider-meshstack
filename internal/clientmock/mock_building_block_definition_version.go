package clientmock

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type meshBuildingBlockDefinitionVersionClient struct {
	Store *Store[client.MeshBuildingBlockDefinitionVersion]
}

func (m meshBuildingBlockDefinitionVersionClient) List(_ context.Context, buildingBlockDefinitionUuid string) ([]client.MeshBuildingBlockDefinitionVersion, error) {
	var result []client.MeshBuildingBlockDefinitionVersion
	for _, version := range m.Store.Values() {
		if version.Spec.BuildingBlockDefinitionRef.Uuid == buildingBlockDefinitionUuid {
			result = append(result, *version)
		}
	}
	return result, nil
}

func (m meshBuildingBlockDefinitionVersionClient) Create(_ context.Context, ownedByWorkspace string, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	nextNum := m.getNextVersionNumber()
	versionUuid := uuid.NewString()
	// Compute hashes for all secrets in the spec
	backendSecretBehavior(true, &versionSpec, nil)
	applyManualOutputBehavior(&versionSpec)

	// Set version number if not already set
	if versionSpec.VersionNumber == nil {
		versionSpec.VersionNumber = new(int64(nextNum))
	}

	created := &client.MeshBuildingBlockDefinitionVersion{
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: ownedByWorkspace,
			CreatedOn:        "2024-01-01T00:00:00Z",
		},
		Spec: versionSpec,
	}

	m.Store.Set(versionUuid, created)
	return created, nil
}

func (m meshBuildingBlockDefinitionVersionClient) Update(_ context.Context, uuid string, ownedByWorkspace string, versionSpec client.MeshBuildingBlockDefinitionVersionSpec) (*client.MeshBuildingBlockDefinitionVersion, error) {
	if existing, ok := m.Store.Get(uuid); ok {
		// Compute hashes for all secrets in the spec
		backendSecretBehavior(false, &versionSpec, &existing.Spec)
		applyManualOutputBehavior(&versionSpec)
		if existing.Metadata.OwnedByWorkspace != ownedByWorkspace {
			return nil, fmt.Errorf("mismatching workspace ownership: %s (existing) != %s (expected)", existing.Metadata.OwnedByWorkspace, ownedByWorkspace)
		}
		existing.Spec = versionSpec
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition version not found: %s", uuid)
}

// applyManualOutputBehavior mirrors the real backend's ManualBuildingBlockCreationModule: for manual
// building blocks the outputs are derived from the inputs (one output per input, assignment type NONE,
// with SINGLE_SELECT/MULTI_SELECT/LIST input types translated to output-compatible types). Any output the
// caller marked with a non-NONE assignment_type (PLATFORM_TENANT_ID, SIGN_IN_URL, RESOURCE_URL, SUMMARY)
// keeps that assignment on the matching input's derived output; all other supplied outputs are ignored,
// matching the backend.
func applyManualOutputBehavior(versionSpec *client.MeshBuildingBlockDefinitionVersionSpec) {
	if versionSpec.Implementation.Manual == nil {
		return
	}
	specialOutputAssignmentTypes := map[string]client.MeshBuildingBlockDefinitionOutputAssignmentType{}
	for key, output := range versionSpec.Outputs {
		if output.AssignmentType != client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.Unwrap() {
			specialOutputAssignmentTypes[key] = output.AssignmentType
		}
	}
	// Mirror the backend (ManualDefinitionVersionService.mapInputsToOutputUpdateModels): derived outputs
	// take their display_order from the input's position in (display_order, key)-sorted order, NOT the
	// input's own display_order. So two inputs both at display_order 0 yield outputs 0 and 1.
	keys := make([]string, 0, len(versionSpec.Inputs))
	for key := range versionSpec.Inputs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if di, dj := versionSpec.Inputs[keys[i]].DisplayOrder, versionSpec.Inputs[keys[j]].DisplayOrder; di != dj {
			return di < dj
		}
		return keys[i] < keys[j]
	})

	outputs := make(map[string]client.MeshBuildingBlockDefinitionOutput, len(versionSpec.Inputs))
	for index, key := range keys {
		input := versionSpec.Inputs[key]
		assignmentType := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.Unwrap()
		if definedSpecialType, ok := specialOutputAssignmentTypes[key]; ok {
			assignmentType = definedSpecialType
		}
		outputs[key] = client.MeshBuildingBlockDefinitionOutput{
			DisplayName:    input.DisplayName,
			Type:           translateManualInputTypeToOutput(input.Type),
			AssignmentType: assignmentType,
			DisplayOrder:   int64(index),
		}
	}
	versionSpec.Outputs = outputs
}

// translateManualInputTypeToOutput mirrors backend ManualIOTypeTranslation: SINGLE_SELECT, MULTI_SELECT
// and LIST cannot be output types and are translated; all other types are kept as-is.
func translateManualInputTypeToOutput(inputType client.MeshBuildingBlockIOType) client.MeshBuildingBlockIOType {
	switch inputType {
	case client.MeshBuildingBlockIOTypeSingleSelect.Unwrap():
		return client.MeshBuildingBlockIOTypeString.Unwrap()
	case client.MeshBuildingBlockIOTypeMultiSelect.Unwrap(), client.MeshBuildingBlockIOTypeList.Unwrap():
		return client.MeshBuildingBlockIOTypeCode.Unwrap()
	default:
		return inputType
	}
}

func (m meshBuildingBlockDefinitionVersionClient) getNextVersionNumber() int {
	maxNum := 0
	for _, v := range m.Store.Values() {
		if v.Spec.VersionNumber != nil {
			num := int(*v.Spec.VersionNumber)
			if num > maxNum {
				maxNum = num
			}
		}
	}
	return maxNum + 1
}
