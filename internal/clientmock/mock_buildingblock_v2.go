package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockV2Client struct {
	Store Store[client.MeshBuildingBlockV2]
}

func (m MeshBuildingBlockV2Client) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockV2, error) {
	if bb, ok := m.Store[uuid]; ok {
		return bb, nil
	}
	return nil, nil
}

func (m MeshBuildingBlockV2Client) ReadFunc(uuid string) func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*client.MeshBuildingBlockV2, error) {
		return m.Read(ctx, uuid)
	}
}

func (m MeshBuildingBlockV2Client) Create(_ context.Context, bb *client.MeshBuildingBlockV2Create) (*client.MeshBuildingBlockV2, error) {
	id := uuid.NewString()

	ownedByWorkspace := ""
	if bb.Spec.TargetRef.Identifier != nil {
		ownedByWorkspace = *bb.Spec.TargetRef.Identifier
	}

	created := &client.MeshBuildingBlockV2{
		Metadata: client.MeshBuildingBlockV2Metadata{
			Uuid:             id,
			OwnedByWorkspace: ownedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: bb.Spec,
		Status: client.MeshBuildingBlockV2Status{
			Status:  client.BUILDING_BLOCK_STATUS_SUCCEEDED,
			Outputs: make([]client.MeshBuildingBlockIO, 0),
		},
	}

	m.Store[id] = created
	return created, nil
}

func (m MeshBuildingBlockV2Client) Delete(_ context.Context, uuid string) error {
	delete(m.Store, uuid)
	return nil
}
