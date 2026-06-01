package clientmock

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type meshBuildingBlockDefinitionClient struct {
	Store        *Store[client.MeshBuildingBlockDefinition]
	StoreVersion *Store[client.MeshBuildingBlockDefinitionVersion]
}

func (m meshBuildingBlockDefinitionClient) List(_ context.Context, workspaceIdentifier *string) ([]client.MeshBuildingBlockDefinition, error) {
	var result []client.MeshBuildingBlockDefinition
	for _, def := range m.Store.Values() {
		if workspaceIdentifier == nil || def.Metadata.OwnedByWorkspace == *workspaceIdentifier {
			result = append(result, *def)
		}
	}
	return result, nil
}

func (m meshBuildingBlockDefinitionClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockDefinition, error) {
	if def, ok := m.Store.Get(uuid); ok {
		return def, nil
	}
	return nil, nil
}

func (m meshBuildingBlockDefinitionClient) Create(_ context.Context, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	definitionUuid := uuid.NewString()
	definition.Metadata.Uuid = new(definitionUuid)
	if definition.Spec.Symbol == nil {
		definition.Spec.Symbol = new("mock-default-symbol")
	}
	m.Store.Set(definitionUuid, &definition)

	// Create initial empty version (as the backend does)
	versionUuid := uuid.NewString()
	m.StoreVersion.Set(versionUuid, &client.MeshBuildingBlockDefinitionVersion{
		Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{
			Uuid:             versionUuid,
			OwnedByWorkspace: definition.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshBuildingBlockDefinitionVersionSpec{
			BuildingBlockDefinitionRef: &client.BuildingBlockDefinitionRef{
				Uuid: definitionUuid,
				Kind: "meshBuildingBlockDefinition",
			},
			DeletionMode:  client.BuildingBlockDeletionModeDelete.Unwrap(),
			VersionNumber: new(int64(1)),
			State:         client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
		},
	})
	return &definition, nil
}

func (m meshBuildingBlockDefinitionClient) Update(_ context.Context, uuid string, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	if existing, ok := m.Store.Get(uuid); ok {
		existing.Spec = definition.Spec
		existing.Metadata.Tags = definition.Metadata.Tags
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition not found: %s", uuid)
}

func (m meshBuildingBlockDefinitionClient) Delete(_ context.Context, uuid string) error {
	m.Store.Delete(uuid)
	return nil
}
