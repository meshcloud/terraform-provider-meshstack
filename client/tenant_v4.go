package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const CONTENT_TYPE_TENANT_V4 = "application/vnd.meshcloud.api.meshtenant.v4-preview.hal+json"

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

func (c *MeshStackProviderClient) urlForTenantV4(uuid string) *url.URL {
	return c.endpoints.Tenants.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadTenantV4(uuid string) (*MeshTenantV4, error) {
	targetUrl := c.urlForTenantV4(uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_TENANT_V4)

	body, err := c.doAuthenticatedRequest(req)
	if errors.Is(err, errNotFound) {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	var tenant MeshTenantV4
	err = json.Unmarshal(body, &tenant)
	if err != nil {
		return nil, err
	}

	return &tenant, nil
}

func (c *MeshStackProviderClient) CreateTenantV4(tenant *MeshTenantV4Create) (*MeshTenantV4, error) {
	payload, err := json.Marshal(tenant)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Tenants.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_TENANT_V4)
	req.Header.Set("Accept", CONTENT_TYPE_TENANT_V4)

	body, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	var createdTenant MeshTenantV4
	err = json.Unmarshal(body, &createdTenant)
	if err != nil {
		return nil, err
	}

	return &createdTenant, nil
}

func (c *MeshStackProviderClient) DeleteTenantV4(uuid string) error {
	targetUrl := c.urlForTenantV4(uuid)
	return c.deleteMeshObject(*targetUrl, 202)
}

// PollTenantV4UntilCreation polls a tenant until creation completes (platformTenantId is set)
// Returns the final tenant state or an error if polling fails or times out.
func (c *MeshStackProviderClient) PollTenantV4UntilCreation(ctx context.Context, uuid string) (*MeshTenantV4, error) {
	var result *MeshTenantV4

	err := retry.RetryContext(ctx, 30*time.Minute, c.waitForTenantV4CreationFunc(uuid, &result))
	return result, err
}

// waitForTenantV4CreationFunc returns a RetryFunc that checks tenant creation status.
func (c *MeshStackProviderClient) waitForTenantV4CreationFunc(uuid string, result **MeshTenantV4) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.ReadTenantV4(uuid)
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

// PollTenantV4UntilDeletion polls a tenant until it is deleted (not found)
// Returns nil on successful deletion or an error if polling fails or times out.
func (c *MeshStackProviderClient) PollTenantV4UntilDeletion(ctx context.Context, uuid string) error {
	return retry.RetryContext(ctx, 30*time.Minute, c.waitForTenantV4DeletionFunc(uuid))
}

// waitForTenantV4DeletionFunc returns a RetryFunc that checks tenant deletion status.
func (c *MeshStackProviderClient) waitForTenantV4DeletionFunc(uuid string) retry.RetryFunc {
	return func() *retry.RetryError {
		current, err := c.ReadTenantV4(uuid)
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
