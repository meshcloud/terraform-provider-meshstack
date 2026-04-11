package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshBuildingBlockRun struct {
	Metadata MeshBuildingBlockRunMetadata `json:"metadata"`
	Spec     MeshBuildingBlockRunSpec     `json:"spec"`
	Status   string                       `json:"status"`
}

type MeshBuildingBlockRunMetadata struct {
	Uuid      string `json:"uuid"`
	CreatedOn string `json:"createdOn"`
}

type MeshBuildingBlockRunSpec struct {
	RunNumber     int64                             `json:"runNumber"`
	Behavior      string                            `json:"behavior"`
	BuildingBlock MeshBuildingBlockRunBuildingBlock `json:"buildingBlock"`
}

type MeshBuildingBlockRunBuildingBlock struct {
	Uuid string `json:"uuid"`
}

type MeshBuildingBlockRunClient interface {
	ListByBuildingBlockUUID(ctx context.Context, buildingBlockUUID string) ([]MeshBuildingBlockRun, error)
}

type meshBuildingBlockRunClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockRun]
}

func newBuildingBlockRunClient(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockRunClient {
	return meshBuildingBlockRunClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockRun](ctx, httpClient, "v1"),
	}
}

func (c meshBuildingBlockRunClient) ListByBuildingBlockUUID(ctx context.Context, buildingBlockUUID string) ([]MeshBuildingBlockRun, error) {
	return c.meshObject.List(ctx, internal.WithUrlQuery("buildingBlockUuid", buildingBlockUUID))
}
