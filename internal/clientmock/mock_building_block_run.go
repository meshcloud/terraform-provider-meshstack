package clientmock

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/client/http"
	"github.com/meshcloud/meshstack-cli/client/types/hal"
)

type MeshBuildingBlockRunClient struct {
	client.MeshBuildingBlockRunClient

	Store    *Store[client.MeshBuildingBlockRun]
	LogStore *Store[client.MeshBuildingBlockRunLogs]
}

func (m MeshBuildingBlockRunClient) Read(_ context.Context, runUuid string) (*client.MeshBuildingBlockRun, error) {
	stored, ok := m.Store.Get(runUuid)
	if !ok {
		return nil, nil
	}
	run := *stored
	if _, hasLogs := m.LogStore.Get(runUuid); hasLogs {
		run.Links = hal.Links{"downloadLogs": hal.NewLink(runUuid, m.downloadLogs)}
	}
	return &run, nil
}

func (m MeshBuildingBlockRunClient) downloadLogs(_ context.Context, _, runUuid string, _ ...http.RequestOption) (io.ReadCloser, error) {
	logs, _ := m.LogStore.Get(runUuid)
	data, err := json.Marshal(logs)
	return io.NopCloser(bytes.NewReader(data)), err
}
