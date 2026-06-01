package clientmock

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockRunClient struct {
	Store    *Store[client.MeshBuildingBlockRun]
	LogStore *Store[client.MeshBuildingBlockRunLogs]
}

func (m MeshBuildingBlockRunClient) GetLogs(_ context.Context, runUuid string) (client.MeshBuildingBlockRunLogs, error) {
	if m.LogStore != nil {
		if logs, ok := m.LogStore.Get(runUuid); ok && logs != nil {
			return *logs, nil
		}
	}
	return client.MeshBuildingBlockRunLogs{}, nil
}
