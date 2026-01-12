package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

const (
	// Building Block Status Constants.
	BUILDING_BLOCK_STATUS_WAITING_FOR_DEPENDENT_INPUT = "WAITING_FOR_DEPENDENT_INPUT"
	BUILDING_BLOCK_STATUS_WAITING_FOR_OPERATOR_INPUT  = "WAITING_FOR_OPERATOR_INPUT"
	BUILDING_BLOCK_STATUS_PENDING                     = "PENDING"
	BUILDING_BLOCK_STATUS_IN_PROGRESS                 = "IN_PROGRESS"
	BUILDING_BLOCK_STATUS_SUCCEEDED                   = "SUCCEEDED"
	BUILDING_BLOCK_STATUS_FAILED                      = "FAILED"
)

type MeshBuildingBlockV2 struct {
	ApiVersion string                      `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                      `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockV2Metadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockV2Spec     `json:"spec" tfsdk:"spec"`
	Status     MeshBuildingBlockV2Status   `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockV2Metadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	OwnedByWorkspace    string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy *string `json:"markedForDeletionBy" tfsdk:"marked_for_deletion_by"`
}

type MeshBuildingBlockV2Spec struct {
	BuildingBlockDefinitionVersionRef MeshBuildingBlockV2DefinitionVersionRef `json:"buildingBlockDefinitionVersionRef" tfsdk:"building_block_definition_version_ref"`
	TargetRef                         MeshBuildingBlockV2TargetRef            `json:"targetRef" tfsdk:"target_ref"`
	DisplayName                       string                                  `json:"displayName" tfsdk:"display_name"`

	Inputs               []MeshBuildingBlockIO     `json:"inputs" tfsdk:"inputs"`
	ParentBuildingBlocks []MeshBuildingBlockParent `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockV2DefinitionVersionRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshBuildingBlockV2TargetRef struct {
	Kind       string  `json:"kind" tfsdk:"kind"`
	Uuid       *string `json:"uuid" tfsdk:"uuid"`
	Identifier *string `json:"identifier" tfsdk:"identifier"`
}

type MeshBuildingBlockV2Create struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                  `json:"kind" tfsdk:"kind"`
	Spec       MeshBuildingBlockV2Spec `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockV2Status struct {
	Status     string                `json:"status" tfsdk:"status"`
	Outputs    []MeshBuildingBlockIO `json:"outputs" tfsdk:"outputs"`
	ForcePurge bool                  `json:"forcePurge" tfsdk:"force_purge"`
}

type MeshBuildingBlockV2Client struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockV2]
}

func newBuildingBlockV2Client(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockV2Client {
	return MeshBuildingBlockV2Client{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockV2](ctx, httpClient, "v2-preview"),
	}
}

func (c MeshBuildingBlockV2Client) Read(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c MeshBuildingBlockV2Client) ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV2, error) {
	return func(ctx context.Context) (*MeshBuildingBlockV2, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c MeshBuildingBlockV2Client) Create(ctx context.Context, bb *MeshBuildingBlockV2Create) (*MeshBuildingBlockV2, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c MeshBuildingBlockV2Client) Delete(ctx context.Context, uuid string) error {
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
		done = true
	case bb.Status.Status == BUILDING_BLOCK_STATUS_FAILED:
		err = fmt.Errorf("building block %s reached FAILED state during deletion. For more details, check the building block run logs in meshStack", bb.Metadata.Uuid)
	}
	return
}
