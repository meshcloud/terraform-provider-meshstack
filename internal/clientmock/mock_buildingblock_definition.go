package clientmock

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
)

type MeshBuildingBlockDefinitionClient struct {
	Store        Store[client.MeshBuildingBlockDefinition]
	StoreVersion Store[client.MeshBuildingBlockDefinitionVersion]
}

func (m MeshBuildingBlockDefinitionClient) List(_ context.Context, workspaceIdentifier *string) ([]client.MeshBuildingBlockDefinition, error) {
	var result []client.MeshBuildingBlockDefinition
	for _, def := range m.Store {
		if workspaceIdentifier == nil || def.Metadata.OwnedByWorkspace == *workspaceIdentifier {
			result = append(result, *def)
		}
	}
	return result, nil
}

func (m MeshBuildingBlockDefinitionClient) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockDefinition, error) {
	if def, ok := m.Store[uuid]; ok {
		return def, nil
	}
	return nil, nil
}

func (m MeshBuildingBlockDefinitionClient) Create(_ context.Context, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	definitionUuid := acctest.RandString(32)
	definition.Metadata.Uuid = ptr.To(definitionUuid)
	if definition.Spec.Symbol == nil {
		definition.Spec.Symbol = ptr.To("mock-default-symbol")
	}
	m.Store[definitionUuid] = &definition

	// Create initial empty version (as the backend does)
	versionUuid := acctest.RandString(32)
	m.StoreVersion[versionUuid] = &client.MeshBuildingBlockDefinitionVersion{
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
			VersionNumber: ptr.To(int64(1)),
			State:         client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
		},
	}
	return &definition, nil
}

func (m MeshBuildingBlockDefinitionClient) Update(_ context.Context, uuid string, definition client.MeshBuildingBlockDefinition) (*client.MeshBuildingBlockDefinition, error) {
	if existing, ok := m.Store[uuid]; ok {
		existing.Spec = definition.Spec
		existing.Metadata.Tags = definition.Metadata.Tags
		return existing, nil
	}
	return nil, fmt.Errorf("building block definition not found: %s", uuid)
}

func (m MeshBuildingBlockDefinitionClient) Delete(_ context.Context, uuid string) error {
	delete(m.Store, uuid)
	return nil
}
