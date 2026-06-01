package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

// mockBuildingBlockCreatedOn is a fixed creation timestamp so the v1 building block's computed
// created_on is stable across Create and a subsequent refresh-Read (the v2 store does not persist
// it, so it is reconstructed deterministically). Mirrors the fixed timestamp used by the mock BBD
// version client.
const mockBuildingBlockCreatedOn = "2024-01-01T00:00:00Z"

// meshBuildingBlockClient is the mock for the legacy v1 building block resource/data source
// (meshstack_buildingblock). On the real backend a building block is a single entity exposed by
// both the v1 and the v2/v3 APIs, so a block created via v1 is readable via v2/v3 (same uuid). The
// mock models this by backing the v1 client with the SAME store the v2/v3 clients use, mapping the
// v1 representation to/from the v2 one. This is what lets the v1->v3 `moved` migration refresh-Read
// find the live block (and plan an in-place Update instead of a destroy+recreate / Create).
//
// The v1 DTO carries references the v2 DTO does not (definition uuid + version number,
// tenant_identifier) and omits the resolved refs the v2 DTO needs (version uuid, target_ref uuid),
// so Create/Read resolve between the two using the BBD-version store and the tenant store — exactly
// the lookups the backend performs.
type meshBuildingBlockClient struct {
	// Store is the shared v2 building block store (also used by the v2/v3 clients).
	Store *Store[client.MeshBuildingBlockV2]
	// BbdVersionStore resolves definition uuid + version number <-> definition-version uuid.
	BbdVersionStore *Store[client.MeshBuildingBlockDefinitionVersion]
	// TenantStore resolves tenant_identifier <-> tenant target_ref uuid.
	TenantStore *Store[client.MeshTenantV4]
}

func (m meshBuildingBlockClient) Read(_ context.Context, id string) (*client.MeshBuildingBlock, error) {
	stored, ok := m.Store.Get(id)
	if !ok {
		return nil, nil
	}
	return m.toV1(deepCopyBB(stored))
}

func (m meshBuildingBlockClient) Create(_ context.Context, bb *client.MeshBuildingBlockCreate) (*client.MeshBuildingBlock, error) {
	versionUuid, err := m.resolveVersionUuid(bb.Metadata.DefinitionUuid, bb.Metadata.DefinitionVersion)
	if err != nil {
		return nil, err
	}
	tenantUuid, ownedByWorkspace, err := m.resolveTenant(bb.Metadata.TenantIdentifier)
	if err != nil {
		return nil, err
	}

	id := uuid.NewString()
	runUuid := uuid.NewString()

	inputs := make(map[string]*client.MeshBuildingBlockInput, len(bb.Spec.Inputs))
	for _, io := range bb.Spec.Inputs {
		var value clientTypes.SecretOrAny
		value.Y = io.Value
		valueType := enum.Entry[client.MeshBuildingBlockIOType](io.ValueType)
		inputs[io.Key] = &client.MeshBuildingBlockInput{
			Value:          value,
			ValueType:      &valueType,
			AssignmentType: client.MeshBuildingBlockInputAssignmentTypeUserInput,
		}
	}
	// Mirror the v2 Create path so a v1-created block looks identical to a v2-created one in the
	// shared store (the real backend returns the same canonical representation either way).
	materializeNullRows(inputs, m.BbdVersionStore, versionUuid)

	parents := clientTypes.Set[client.MeshBuildingBlockParent](bb.Spec.ParentBuildingBlocks)

	stored := &client.MeshBuildingBlockV2{
		Metadata: client.MeshBuildingBlockV2Metadata{
			Uuid:             &id,
			OwnedByWorkspace: ownedByWorkspace,
		},
		Spec: client.MeshBuildingBlockV2Spec{
			BuildingBlockDefinitionVersionRef: client.MeshBuildingBlockV2DefinitionVersionRef{Uuid: versionUuid},
			TargetRef: client.MeshBuildingBlockV2TargetRef{
				Kind: client.MeshObjectKind.Tenant,
				Uuid: &tenantUuid,
			},
			DisplayName:          bb.Spec.DisplayName,
			Inputs:               inputs,
			ParentBuildingBlocks: parents,
		},
		Status: &client.MeshBuildingBlockV2Status{
			Status:        client.BuildingBlockStatusSucceeded,
			LatestRunUuid: &runUuid,
			Lifecycle: client.MeshBuildingBlockV2Lifecycle{
				State: client.BuildingBlockLifecycleStateActive,
			},
		},
	}

	m.Store.Set(id, stored)
	return m.toV1(deepCopyBB(stored))
}

func (m meshBuildingBlockClient) Delete(_ context.Context, id string) error {
	m.Store.Delete(id)
	return nil
}

// toV1 maps a stored v2 building block back to the v1 representation, reconstructing the
// v1-only metadata (definition uuid + version number, tenant_identifier) from the BBD-version and
// tenant stores.
func (m meshBuildingBlockClient) toV1(v2 *client.MeshBuildingBlockV2) (*client.MeshBuildingBlock, error) {
	definitionUuid, definitionVersion, err := m.resolveDefinitionRef(v2.Spec.BuildingBlockDefinitionVersionRef.Uuid)
	if err != nil {
		return nil, err
	}
	tenantIdentifier, err := m.resolveTenantIdentifier(v2.Spec.TargetRef)
	if err != nil {
		return nil, err
	}

	id := ""
	if v2.Metadata.Uuid != nil {
		id = *v2.Metadata.Uuid
	}

	inputs := make([]client.MeshBuildingBlockIO, 0, len(v2.Spec.Inputs))
	for key, in := range v2.Spec.Inputs {
		valueType := ""
		if in.ValueType != nil {
			valueType = string(*in.ValueType)
		}
		var value any
		if in.Value.HasY() {
			value = in.Value.Y
		}
		inputs = append(inputs, client.MeshBuildingBlockIO{
			Key:       key,
			Value:     value,
			ValueType: valueType,
		})
	}

	status := string(client.BuildingBlockStatusSucceeded.Unwrap())
	if v2.Status != nil {
		status = string(v2.Status.Status.Unwrap())
	}

	return &client.MeshBuildingBlock{
		Metadata: client.MeshBuildingBlockMetadata{
			Uuid:              id,
			DefinitionUuid:    definitionUuid,
			DefinitionVersion: definitionVersion,
			TenantIdentifier:  tenantIdentifier,
			CreatedOn:         mockBuildingBlockCreatedOn,
		},
		Spec: client.MeshBuildingBlockSpec{
			DisplayName:          v2.Spec.DisplayName,
			Inputs:               inputs,
			ParentBuildingBlocks: []client.MeshBuildingBlockParent(v2.Spec.ParentBuildingBlocks),
		},
		Status: client.MeshBuildingBlockStatus{
			Status:  status,
			Outputs: []client.MeshBuildingBlockIO{},
		},
	}, nil
}

// resolveVersionUuid finds the definition-version uuid for a (definition uuid, version number) pair,
// the lookup the backend does when a v1 create references a definition by uuid + version number.
func (m meshBuildingBlockClient) resolveVersionUuid(definitionUuid string, versionNumber int64) (string, error) {
	for _, v := range m.BbdVersionStore.Values() {
		if v.Spec.BuildingBlockDefinitionRef.Uuid == definitionUuid &&
			v.Spec.VersionNumber != nil && *v.Spec.VersionNumber == versionNumber {
			return v.Metadata.Uuid, nil
		}
	}
	return "", fmt.Errorf("mock: no building block definition version found for definition %q version %d", definitionUuid, versionNumber)
}

// resolveDefinitionRef is the inverse of resolveVersionUuid: definition-version uuid ->
// (definition uuid, version number).
func (m meshBuildingBlockClient) resolveDefinitionRef(versionUuid string) (definitionUuid string, versionNumber int64, err error) {
	v, ok := m.BbdVersionStore.Get(versionUuid)
	if !ok {
		return "", 0, fmt.Errorf("mock: building block definition version %q not found", versionUuid)
	}
	if v.Spec.VersionNumber == nil {
		return "", 0, fmt.Errorf("mock: building block definition version %q has no version number", versionUuid)
	}
	return v.Spec.BuildingBlockDefinitionRef.Uuid, *v.Spec.VersionNumber, nil
}

// resolveTenant maps a tenant_identifier (workspace.project.platformIdentifier) to the tenant's
// uuid and owning workspace by matching the tenant's computed name, mirroring how the backend
// resolves the target tenant of a v1 building block.
func (m meshBuildingBlockClient) resolveTenant(tenantIdentifier string) (tenantUuid string, ownedByWorkspace string, err error) {
	for _, t := range m.TenantStore.Values() {
		if t.Status.TenantName == tenantIdentifier {
			return t.Metadata.Uuid, t.Metadata.OwnedByWorkspace, nil
		}
	}
	return "", "", fmt.Errorf("mock: no tenant found for identifier %q", tenantIdentifier)
}

// resolveTenantIdentifier is the inverse of resolveTenant: tenant target_ref -> tenant_identifier.
func (m meshBuildingBlockClient) resolveTenantIdentifier(targetRef client.MeshBuildingBlockV2TargetRef) (string, error) {
	if targetRef.Uuid == nil {
		return "", fmt.Errorf("mock: tenant target_ref has no uuid")
	}
	t, ok := m.TenantStore.Get(*targetRef.Uuid)
	if !ok {
		return "", fmt.Errorf("mock: tenant %q not found", *targetRef.Uuid)
	}
	return t.Status.TenantName, nil
}
