package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockRunClient struct {
	Store *Store[client.MeshBuildingBlockRun]
}

func (m MeshBuildingBlockRunClient) ListByBuildingBlockUUID(_ context.Context, buildingBlockUUID string) ([]client.MeshBuildingBlockRun, error) {
	var result []client.MeshBuildingBlockRun
	for _, run := range m.Store.Values() {
		if run.Spec.BuildingBlock.Uuid == buildingBlockUUID {
			result = append(result, *run)
		}
	}
	return result, nil
}
