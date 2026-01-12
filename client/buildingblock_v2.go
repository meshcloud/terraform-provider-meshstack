package client

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"

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
	return c.meshObject.Get(ctx, uuid)
}

func (c MeshBuildingBlockV2Client) Create(ctx context.Context, bb *MeshBuildingBlockV2Create) (*MeshBuildingBlockV2, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c MeshBuildingBlockV2Client) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}

// PollUntilCompletion polls a building block until it reaches a terminal state (SUCCEEDED or FAILED)
// Returns the final building block state or an error if polling fails or times out.
func (c MeshBuildingBlockV2Client) PollUntilCompletion(ctx context.Context, uuid string) (*MeshBuildingBlockV2, error) {
	var result *MeshBuildingBlockV2

	err := retry.RetryContext(ctx, 30*time.Minute, c.waitForCompletionFunc(ctx, uuid, &result))
	return result, err
}

// waitForCompletionFunc returns a RetryFunc that checks building block completion status.
func (c MeshBuildingBlockV2Client) waitForCompletionFunc(ctx context.Context, uuid string, result **MeshBuildingBlockV2) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.Read(ctx, uuid)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("could not read building block status while waiting for completion: %w", err))
		}

		if current == nil {
			return retry.NonRetryableError(fmt.Errorf("building block was not found while waiting for completion"))
		}
		*result = current

		// Check if we've reached a terminal state
		status := current.Status.Status
		switch status {
		case BUILDING_BLOCK_STATUS_SUCCEEDED:
			return nil // Success, stop retrying
		case BUILDING_BLOCK_STATUS_FAILED:
			return retry.NonRetryableError(fmt.Errorf("building block %s reached FAILED state", uuid))
		}

		// Not done yet, continue polling
		return retry.RetryableError(fmt.Errorf("waiting for building block %s to complete: currently in %s state", uuid, status))
	}
}

// PollUntilDeletion polls a building block until it is deleted (not found)
// Returns nil on successful deletion or an error if polling fails or times out.
func (c MeshBuildingBlockV2Client) PollUntilDeletion(ctx context.Context, uuid string) error {
	return retry.RetryContext(ctx, 30*time.Minute, c.waitForDeletionFunc(ctx, uuid))
}

// waitForDeletionFunc returns a RetryFunc that checks building block deletion status.
func (c MeshBuildingBlockV2Client) waitForDeletionFunc(ctx context.Context, uuid string) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.Read(ctx, uuid)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("could not read building block status while waiting for deletion: %w", err))
		}

		// If building block is not found, deletion is complete
		if current == nil {
			return nil // Success, stop retrying
		}

		// If building block is in FAILED state during deletion, consider it a terminal state
		if current.Status.Status == BUILDING_BLOCK_STATUS_FAILED {
			return retry.NonRetryableError(fmt.Errorf("building block %s reached FAILED state during deletion. For more details, check the building block run logs in meshStack", uuid))
		}

		// Not done yet, continue polling
		return retry.RetryableError(fmt.Errorf("waiting for building block %s to be deleted: currently in %s state", uuid, current.Status.Status))
	}
}
