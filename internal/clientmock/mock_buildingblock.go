package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockClient struct {
	Store Store[client.MeshBuildingBlock]
}

func (m MeshBuildingBlockClient) Read(_ context.Context, id string) (*client.MeshBuildingBlock, error) {
	return m.Store[id], nil
}

func (m MeshBuildingBlockClient) Create(_ context.Context, bb *client.MeshBuildingBlockCreate) (*client.MeshBuildingBlock, error) {
	bbUuid := uuid.NewString()

	created := &client.MeshBuildingBlock{
		Metadata: client.MeshBuildingBlockMetadata{
			Uuid:              bbUuid,
			DefinitionUuid:    bb.Metadata.DefinitionUuid,
			DefinitionVersion: bb.Metadata.DefinitionVersion,
			TenantIdentifier:  bb.Metadata.TenantIdentifier,
			CreatedOn:         time.Now().UTC().Format(time.RFC3339),
		},
		Spec: bb.Spec,
		Status: client.MeshBuildingBlockStatus{
			Status:  "SUCCEEDED",
			Outputs: []client.MeshBuildingBlockIO{},
		},
	}

	m.Store[bbUuid] = created
	return created, nil
}

func (m MeshBuildingBlockClient) Delete(_ context.Context, id string) error {
	delete(m.Store, id)
	return nil
}
