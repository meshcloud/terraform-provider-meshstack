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

// MeshBuildingBlockRunLogs is the response from the download-logs actions endpoint.
type MeshBuildingBlockRunLogs struct {
	Steps []MeshBuildingBlockRunStepLog `json:"steps"`
}

// MeshBuildingBlockRunStepLog represents a single step's log data.
type MeshBuildingBlockRunStepLog struct {
	DisplayName   string  `json:"displayName"`
	Status        string  `json:"status"`
	UserMessage   *string `json:"userMessage"`
	SystemMessage *string `json:"systemMessage"`
}

type MeshBuildingBlockRunClient interface {
	ListByBuildingBlockUUID(ctx context.Context, buildingBlockUUID string) ([]MeshBuildingBlockRun, error)
	DownloadLogs(ctx context.Context, runUUID string) (*MeshBuildingBlockRunLogs, error)
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

func (c meshBuildingBlockRunClient) DownloadLogs(ctx context.Context, runUUID string) (*MeshBuildingBlockRunLogs, error) {
	return internal.GetActionAs[MeshBuildingBlockRun, MeshBuildingBlockRunLogs](ctx, c.meshObject, runUUID, "download-logs")
}
