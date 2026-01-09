package client

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

type MeshTenantV4 struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshTenantV4Metadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshTenantV4Spec     `json:"spec" tfsdk:"spec"`
	Status     MeshTenantV4Status   `json:"status" tfsdk:"status"`
}

type MeshTenantV4Metadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	OwnedByProject      string  `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace    string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	DeletedOn           *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshTenantV4Spec struct {
	PlatformIdentifier    string             `json:"platformIdentifier" tfsdk:"platform_identifier"`
	PlatformTenantId      *string            `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	LandingZoneIdentifier *string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                *[]MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantV4Status struct {
	TenantName                  string              `json:"tenantName" tfsdk:"tenant_name"`
	PlatformTypeIdentifier      string              `json:"platformTypeIdentifier" tfsdk:"platform_type_identifier"`
	PlatformWorkspaceIdentifier *string             `json:"platformWorkspaceIdentifier" tfsdk:"platform_workspace_identifier"`
	Tags                        map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshTenantV4Create struct {
	Metadata MeshTenantV4CreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantV4CreateSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTenantV4CreateMetadata struct {
	OwnedByProject   string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshTenantV4CreateSpec struct {
	PlatformIdentifier    string             `json:"platformIdentifier" tfsdk:"platform_identifier"`
	LandingZoneIdentifier *string            `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	PlatformTenantId      *string            `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	Quotas                *[]MeshTenantQuota `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantV4Client struct {
	meshObjectClient[MeshTenantV4]
}

func (c MeshTenantV4Client) Read(uuid string) (*MeshTenantV4, error) {
	return c.get(uuid)
}

func (c MeshTenantV4Client) Create(tenant *MeshTenantV4Create) (*MeshTenantV4, error) {
	return c.post(tenant)
}

func (c MeshTenantV4Client) Delete(uuid string) error {
	return c.delete(uuid)
}

// PollUntilCreation polls a tenant until creation completes (platformTenantId is set)
// Returns the final tenant state or an error if polling fails or times out.
func (c MeshTenantV4Client) PollUntilCreation(ctx context.Context, uuid string) (*MeshTenantV4, error) {
	var result *MeshTenantV4

	err := retry.RetryContext(ctx, 30*time.Minute, c.waitForCreationFunc(uuid, &result))
	return result, err
}

// waitForCreationFunc returns a RetryFunc that checks tenant creation status.
func (c MeshTenantV4Client) waitForCreationFunc(uuid string, result **MeshTenantV4) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.Read(uuid)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("could not read tenant status while waiting for creation: %w", err))
		}

		if current == nil {
			return retry.NonRetryableError(fmt.Errorf("tenant was not found while waiting for creation"))
		}

		// Check if creation is complete (platformTenantId is set)
		if current.Spec.PlatformTenantId != nil && *current.Spec.PlatformTenantId != "" {
			*result = current
			return nil // Success, stop retrying
		}

		// Not done yet, continue polling
		return retry.RetryableError(fmt.Errorf("waiting for tenant %s creation to complete: platformTenantId not yet set", uuid))
	}
}

// PollUntilDeletion polls a tenant until it is deleted (not found)
// Returns nil on successful deletion or an error if polling fails or times out.
func (c MeshTenantV4Client) PollUntilDeletion(ctx context.Context, uuid string) error {
	return retry.RetryContext(ctx, 30*time.Minute, c.waitForDeletionFunc(uuid))
}

// waitForDeletionFunc returns a RetryFunc that checks tenant deletion status.
func (c MeshTenantV4Client) waitForDeletionFunc(uuid string) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.Read(uuid)
		if err != nil {
			return retry.NonRetryableError(fmt.Errorf("could not read tenant status while waiting for deletion: %w", err))
		}

		// If tenant is not found, deletion is complete
		if current == nil {
			return nil // Success, stop retrying
		}

		// Not done yet, continue polling
		return retry.RetryableError(fmt.Errorf("waiting for tenant %s to be deleted: still present", uuid))
	}
}
