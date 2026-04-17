package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockRunClient struct {
	Store    *Store[client.MeshBuildingBlockRun]
	LogStore map[string]*client.MeshBuildingBlockRunLogs
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

func (m MeshBuildingBlockRunClient) DownloadLogs(_ context.Context, runUUID string) (*client.MeshBuildingBlockRunLogs, error) {
	if m.LogStore != nil {
		if logs, ok := m.LogStore[runUUID]; ok {
			return logs, nil
		}
	}
	return &client.MeshBuildingBlockRunLogs{}, nil
}
