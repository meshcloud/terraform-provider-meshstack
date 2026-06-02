package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

const (
	// Building Block Status Constants.
	BUILDING_BLOCK_STATUS_WAITING_FOR_DEPENDENT_INPUT  = "WAITING_FOR_DEPENDENT_INPUT"
	BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT   = "WAITING_FOR_OPERATOR_INPUT"
	BUILDING_BLOCK_STATUS_PENDING                      = "PENDING"
	BUILDING_BLOCK_STATUS_IN_PROGRESS                  = "IN_PROGRESS"
	BUILDING_BLOCK_STATUS_SUCCEEDED                    = "SUCCEEDED"
	BUILDING_BLOCK_STATUS_FAILED                       = "FAILED"
	BUILDING_BLOCK_LIFECYCLE_STATE_ACTIVE              = "ACTIVE"
	BUILDING_BLOCK_LIFECYCLE_STATE_MARKED_FOR_DELETION = "MARKED_FOR_DELETION"
	BUILDING_BLOCK_LIFECYCLE_STATE_DELETED             = "DELETED"
)

type MeshBuildingBlockV2 struct {
	Metadata MeshBuildingBlockV2Metadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshBuildingBlockV2Spec     `json:"spec" tfsdk:"spec"`
	Status   MeshBuildingBlockV2Status   `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockV2Metadata struct {
	Uuid             string `json:"uuid" tfsdk:"uuid"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshBuildingBlockV2Spec struct {
	BuildingBlockDefinitionVersionRef MeshBuildingBlockV2DefinitionVersionRef `json:"buildingBlockDefinitionVersionRef" tfsdk:"building_block_definition_version_ref"`
	TargetRef                         MeshBuildingBlockV2TargetRef            `json:"targetRef" tfsdk:"target_ref"`
	DisplayName                       string                                  `json:"displayName" tfsdk:"display_name"`

	Inputs               map[string]MeshBuildingBlockV2Input `json:"inputs" tfsdk:"-"`
	ParentBuildingBlocks []MeshBuildingBlockParent           `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockV2Input struct {
	Value                any     `json:"value"`
	ValueType            string  `json:"valueType"`
	IsSensitive          bool    `json:"isSensitive"`
	AssignmentType       *string `json:"assignmentType"`
	UpdateableByConsumer bool    `json:"updateableByConsumer"`
}

type MeshBuildingBlockV2Output struct {
	Value          any     `json:"value"`
	ValueType      string  `json:"valueType"`
	AssignmentType *string `json:"assignmentType"`
}

type MeshBuildingBlockV2DefinitionVersionRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshBuildingBlockV2TargetRef struct {
	Kind string  `json:"kind" tfsdk:"kind"`
	Uuid *string `json:"uuid" tfsdk:"uuid"`
	Name *string `json:"name" tfsdk:"name"`
}

type MeshBuildingBlockV2Create struct {
	Spec MeshBuildingBlockV2Spec `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockV2Lifecycle struct {
	State string `json:"state" tfsdk:"state"`
}

type MeshBuildingBlockV2Status struct {
	Status     string                               `json:"status" tfsdk:"status"`
	Outputs    map[string]MeshBuildingBlockV2Output `json:"outputs" tfsdk:"-"`
	ForcePurge bool                                 `json:"forcePurge" tfsdk:"force_purge"`
	Lifecycle  MeshBuildingBlockV2Lifecycle         `json:"lifecycle" tfsdk:"lifecycle"`
}

type MeshBuildingBlockV2Client interface {
	Read(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV2, error)
	Create(ctx context.Context, bb *MeshBuildingBlockV2Create) (*MeshBuildingBlockV2, error)
	Delete(ctx context.Context, uuid string, purge bool) error
}

type meshBuildingBlockV2Client struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockV2]
}

func newBuildingBlockV2Client(ctx context.Context, httpClient internal.HttpClient) MeshBuildingBlockV2Client {
	return meshBuildingBlockV2Client{internal.NewMeshObjectClient[MeshBuildingBlockV2](ctx, httpClient, "v2-preview")}
}

func (c meshBuildingBlockV2Client) Read(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c meshBuildingBlockV2Client) ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*MeshBuildingBlockV2, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c meshBuildingBlockV2Client) Create(ctx context.Context, bb *MeshBuildingBlockV2Create) (*MeshBuildingBlockV2, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c meshBuildingBlockV2Client) Delete(ctx context.Context, uuid string, purge bool) error {
	if purge {
		return c.meshObject.Purge(ctx, uuid)
	}
	return c.meshObject.Delete(ctx, uuid)
}

func (bb *MeshBuildingBlockV2) CreateSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		err = fmt.Errorf("building block not found after creation")
	case bb.Status.Status == BUILDING_BLOCK_STATUS_FAILED:
		err = fmt.Errorf("building block %s reached FAILED state during creation, check the building block run logs in meshStack", bb.Metadata.Uuid)
	case bb.Status.Status == BUILDING_BLOCK_STATUS_SUCCEEDED:
		done = true
	}
	return
}

func (bb *MeshBuildingBlockV2) DeletionSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		// Expected when receiving a 404 (hard deletion), default behavior until meshStack v2026.20.0.
		// For versions higher than that, we get a building block back with a lifecycle state to inspect.
		done = true
	case bb.Status.Lifecycle.State == BUILDING_BLOCK_LIFECYCLE_STATE_DELETED:
		done = true
	case bb.Status.Status == BUILDING_BLOCK_STATUS_FAILED:
		err = fmt.Errorf("building block %s reached FAILED state during deletion. For more details, check the building block run logs in meshStack", bb.Metadata.Uuid)
	}
	return
}
