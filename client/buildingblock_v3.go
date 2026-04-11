package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

type MeshBuildingBlockV3 struct {
	Metadata MeshBuildingBlockV3Metadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshBuildingBlockV3Spec     `json:"spec" tfsdk:"spec"`
	Status   MeshBuildingBlockV3Status   `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockV3Metadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	OwnedByWorkspace    string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy *string `json:"markedForDeletionBy" tfsdk:"marked_for_deletion_by"`
}

type MeshBuildingBlockV3Spec struct {
	BuildingBlockDefinitionVersionRef MeshBuildingBlockV2DefinitionVersionRef  `json:"buildingBlockDefinitionVersionRef" tfsdk:"building_block_definition_version_ref"`
	TargetRef                         MeshBuildingBlockV2TargetRef             `json:"targetRef" tfsdk:"target_ref"`
	DisplayName                       string                                   `json:"displayName" tfsdk:"display_name"`
	Inputs                            map[string]MeshBuildingBlockV3InputValue `json:"inputs" tfsdk:"inputs"`
	InputsPlatformOperator            map[string]MeshBuildingBlockV3InputValue `json:"inputsPlatformOperator" tfsdk:"inputs_platform_operator"`
	InputsStatic                      map[string]MeshBuildingBlockV3InputValue `json:"inputsStatic" tfsdk:"inputs_static"`
	ParentBuildingBlocks              []MeshBuildingBlockParent                `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockV3InputValue struct {
	Value     any                 `json:"value,omitempty" tfsdk:"value"`
	Sensitive *clientTypes.Secret `json:"sensitive,omitempty" tfsdk:"sensitive"`
}

type MeshBuildingBlockV3Status struct {
	Status     string                  `json:"status" tfsdk:"status"`
	Outputs    []MeshBuildingBlockIO   `json:"outputs" tfsdk:"outputs"`
	ForcePurge bool                    `json:"forcePurge" tfsdk:"force_purge"`
	LatestRun  *MeshBuildingBlockV3Run `json:"latestRun" tfsdk:"latest_run"`
}

type MeshBuildingBlockV3Run struct {
	Uuid      string `json:"uuid" tfsdk:"uuid"`
	RunNumber int64  `json:"runNumber" tfsdk:"run_number"`
	Status    string `json:"status" tfsdk:"status"`
	Behavior  string `json:"behavior" tfsdk:"behavior"`
}

type MeshBuildingBlockV3Create struct {
	Spec MeshBuildingBlockV3Spec `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockV3Client interface {
	Read(ctx context.Context, uuid string) (*MeshBuildingBlockV3, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV3, error)
	Create(ctx context.Context, bb *MeshBuildingBlockV3Create) (*MeshBuildingBlockV3, error)
	Update(ctx context.Context, uuid string, bb *MeshBuildingBlockV3Create) (*MeshBuildingBlockV3, error)
	RetriggerRun(ctx context.Context, uuid string) (*MeshBuildingBlockV3, error)
	Delete(ctx context.Context, uuid string, purge bool) error
}

type meshBuildingBlockV3Client struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockV3]
}

func newBuildingBlockV3Client(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockV3Client {
	return meshBuildingBlockV3Client{internal.NewMeshObjectClient[MeshBuildingBlockV3](ctx, httpClient, "v3-preview")}
}

func (c meshBuildingBlockV3Client) Read(ctx context.Context, uuid string) (*MeshBuildingBlockV3, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c meshBuildingBlockV3Client) ReadFunc(uuid string) func(ctx context.Context) (*MeshBuildingBlockV3, error) {
	return func(ctx context.Context) (*MeshBuildingBlockV3, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c meshBuildingBlockV3Client) Create(ctx context.Context, bb *MeshBuildingBlockV3Create) (*MeshBuildingBlockV3, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c meshBuildingBlockV3Client) Update(ctx context.Context, uuid string, bb *MeshBuildingBlockV3Create) (*MeshBuildingBlockV3, error) {
	return c.meshObject.Put(ctx, uuid, bb)
}

func (c meshBuildingBlockV3Client) RetriggerRun(ctx context.Context, uuid string) (*MeshBuildingBlockV3, error) {
	return c.meshObject.Post(ctx, struct{}{}, internal.WithPathElems(uuid, "actions", "retrigger-run"))
}

func (c meshBuildingBlockV3Client) Delete(ctx context.Context, uuid string, purge bool) error {
	mode := "DELETE"
	if purge {
		mode = "PURGE"
	}
	return c.meshObject.Delete(ctx, uuid, internal.WithUrlQuery("mode", mode))
}

func (bb *MeshBuildingBlockV3) CreateSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		err = fmt.Errorf("building block not found after creation")
	case bb.Status.Status == BUILDING_BLOCK_STATUS_FAILED:
		err = fmt.Errorf("building block %s reached FAILED state, check the building block run logs in meshStack", bb.Metadata.Uuid)
	case bb.Status.Status == BUILDING_BLOCK_STATUS_SUCCEEDED:
		done = true
	}
	return
}

func (bb *MeshBuildingBlockV3) DeletionSuccessful() (done bool, err error) {
	switch {
	case bb == nil:
		done = true
	case bb.Status.Status == BUILDING_BLOCK_STATUS_FAILED:
		err = fmt.Errorf("building block %s reached FAILED state during deletion. For more details, check the building block run logs in meshStack", bb.Metadata.Uuid)
	}
	return
}
