package clientmock

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

type MeshBuildingBlockV3Client struct {
	Store    *Store[client.MeshBuildingBlockV3]
	RunStore *Store[client.MeshBuildingBlockRun]
}

func (m MeshBuildingBlockV3Client) Read(_ context.Context, uuid string) (*client.MeshBuildingBlockV3, error) {
	if bb, ok := m.Store.Get(uuid); ok {
		return bb, nil
	}
	return nil, nil
}

func (m MeshBuildingBlockV3Client) ReadFunc(uuid string) func(ctx context.Context) (*client.MeshBuildingBlockV3, error) {
	return func(ctx context.Context) (*client.MeshBuildingBlockV3, error) {
		return m.Read(ctx, uuid)
	}
}

func (m MeshBuildingBlockV3Client) Create(_ context.Context, bb *client.MeshBuildingBlockV3Create) (*client.MeshBuildingBlockV3, error) {
	id := uuid.NewString()

	ownedByWorkspace := ""
	if bb.Spec.TargetRef.Identifier != nil {
		ownedByWorkspace = *bb.Spec.TargetRef.Identifier
	}

	created := &client.MeshBuildingBlockV3{
		Metadata: client.MeshBuildingBlockV3Metadata{
			Uuid:             id,
			OwnedByWorkspace: ownedByWorkspace,
			CreatedOn:        time.Now().UTC().Format(time.RFC3339),
		},
		Spec: bb.Spec,
		Status: client.MeshBuildingBlockV3Status{
			Status:     client.BUILDING_BLOCK_STATUS_SUCCEEDED,
			Outputs:    make([]client.MeshBuildingBlockIO, 0),
			ForcePurge: false,
		},
	}
	m.recordRun(created, "APPLY")

	m.Store.Set(id, created)
	return created, nil
}

func (m MeshBuildingBlockV3Client) Update(_ context.Context, uuid string, bb *client.MeshBuildingBlockV3Create) (*client.MeshBuildingBlockV3, error) {
	existing, ok := m.Store.Get(uuid)
	if !ok {
		return nil, nil
	}
	existing.Spec.DisplayName = bb.Spec.DisplayName
	existing.Spec.BuildingBlockDefinitionVersionRef = bb.Spec.BuildingBlockDefinitionVersionRef
	existing.Spec.TargetRef = bb.Spec.TargetRef
	existing.Spec.ParentBuildingBlocks = bb.Spec.ParentBuildingBlocks
	existing.Spec.Inputs = bb.Spec.Inputs
	existing.Spec.InputsPlatformOperator = bb.Spec.InputsPlatformOperator
	existing.Status.Status = client.BUILDING_BLOCK_STATUS_SUCCEEDED
	m.recordRun(existing, "APPLY")
	return existing, nil
}

func (m MeshBuildingBlockV3Client) RetriggerRun(_ context.Context, uuid string) (*client.MeshBuildingBlockV3, error) {
	existing, ok := m.Store.Get(uuid)
	if !ok {
		return nil, nil
	}
	existing.Status.Status = client.BUILDING_BLOCK_STATUS_SUCCEEDED
	m.recordRun(existing, "APPLY")
	return existing, nil
}

func (m MeshBuildingBlockV3Client) Delete(_ context.Context, uuid string, _ bool) error {
	m.Store.Delete(uuid)
	return nil
}

func (m MeshBuildingBlockV3Client) recordRun(buildingBlock *client.MeshBuildingBlockV3, behavior string) {
	if buildingBlock == nil {
		return
	}

	nextRunNumber := int64(1)
	for _, run := range m.RunStore.Values() {
		if run.Spec.BuildingBlock.Uuid != buildingBlock.Metadata.Uuid {
			continue
		}
		if run.Spec.RunNumber >= nextRunNumber {
			nextRunNumber = run.Spec.RunNumber + 1
		}
	}

	runID := uuid.NewString()
	run := &client.MeshBuildingBlockRun{
		Metadata: client.MeshBuildingBlockRunMetadata{
			Uuid:      runID,
			CreatedOn: time.Now().UTC().Format(time.RFC3339Nano),
		},
		Spec: client.MeshBuildingBlockRunSpec{
			RunNumber: nextRunNumber,
			Behavior:  behavior,
			BuildingBlock: client.MeshBuildingBlockRunBuildingBlock{
				Uuid: buildingBlock.Metadata.Uuid,
			},
		},
		Status: "SUCCEEDED",
	}
	m.RunStore.Set(runID, run)
	buildingBlock.Status.LatestRun = &client.MeshBuildingBlockV3Run{
		Uuid:      run.Metadata.Uuid,
		RunNumber: run.Spec.RunNumber,
		Status:    run.Status,
		Behavior:  run.Spec.Behavior,
	}
}
