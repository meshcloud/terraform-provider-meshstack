package clientmock

import (
	"context"
	"fmt"

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
// caller marked PLATFORM_TENANT_ID keeps that assignment for the matching input; all other supplied
// outputs are ignored, matching the backend.
func applyManualOutputBehavior(versionSpec *client.MeshBuildingBlockDefinitionVersionSpec) {
	if versionSpec.Implementation.Manual == nil {
		return
	}
	platformTenantIdKeys := map[string]bool{}
	for key, output := range versionSpec.Outputs {
		if output.AssignmentType == client.MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID.Unwrap() {
			platformTenantIdKeys[key] = true
		}
	}
	outputs := make(map[string]client.MeshBuildingBlockDefinitionOutput, len(versionSpec.Inputs))
	for key, input := range versionSpec.Inputs {
		assignmentType := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.Unwrap()
		if platformTenantIdKeys[key] {
			assignmentType = client.MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID.Unwrap()
		}
		outputs[key] = client.MeshBuildingBlockDefinitionOutput{
			DisplayName:    input.DisplayName,
			Type:           translateManualInputTypeToOutput(input.Type),
			AssignmentType: assignmentType,
			DisplayOrder:   input.DisplayOrder,
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
