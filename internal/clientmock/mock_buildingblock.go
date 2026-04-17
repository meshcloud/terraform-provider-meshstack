package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type meshBuildingBlockClient struct {
	Store *Store[client.MeshBuildingBlock]
}

func (m meshBuildingBlockClient) Read(_ context.Context, id string) (*client.MeshBuildingBlock, error) {
	v, _ := m.Store.Get(id)
	return v, nil
}

func (m meshBuildingBlockClient) Create(_ context.Context, bb *client.MeshBuildingBlockCreate) (*client.MeshBuildingBlock, error) {
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

	m.Store.Set(bbUuid, created)
	return created, nil
}

func (m meshBuildingBlockClient) Delete(_ context.Context, id string) error {
	m.Store.Delete(id)
	return nil
}
