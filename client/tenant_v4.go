package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshTenantV4 struct {
	Metadata MeshTenantV4Metadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantV4Spec     `json:"spec" tfsdk:"spec"`
	Status   MeshTenantV4Status   `json:"status" tfsdk:"status"`
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

type MeshTenantV4Query struct {
	Workspace      string
	Project        *string
	Platform       *string
	PlatformType   *string
	LandingZone    *string
	PlatformTenant *string
}

type MeshTenantV4Client interface {
	Read(ctx context.Context, uuid string) (*MeshTenantV4, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshTenantV4, error)
	List(ctx context.Context, query *MeshTenantV4Query) ([]MeshTenantV4, error)
	Create(ctx context.Context, tenant *MeshTenantV4Create) (*MeshTenantV4, error)
	Delete(ctx context.Context, uuid string) error
}

type meshTenantV4Client struct {
	meshObject internal.MeshObjectClient[MeshTenantV4]
}

func newTenantV4Client(ctx context.Context, httpClient *internal.HttpClient) MeshTenantV4Client {
	return meshTenantV4Client{internal.NewMeshObjectClient[MeshTenantV4](ctx, httpClient, "v4-preview")}
}

func (c meshTenantV4Client) Read(ctx context.Context, uuid string) (*MeshTenantV4, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c meshTenantV4Client) ReadFunc(uuid string) func(ctx context.Context) (*MeshTenantV4, error) {
	return func(ctx context.Context) (*MeshTenantV4, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c meshTenantV4Client) Create(ctx context.Context, tenant *MeshTenantV4Create) (*MeshTenantV4, error) {
	return c.meshObject.Post(ctx, tenant)
}

func (c meshTenantV4Client) List(ctx context.Context, query *MeshTenantV4Query) ([]MeshTenantV4, error) {
	options := []internal.RequestOption{
		internal.WithUrlQuery("workspaceIdentifier", query.Workspace),
	}
	if query.Project != nil {
		options = append(options, internal.WithUrlQuery("projectIdentifier", *query.Project))
	}
	if query.Platform != nil {
		options = append(options, internal.WithUrlQuery("platformIdentifier", *query.Platform))
	}
	if query.PlatformType != nil {
		options = append(options, internal.WithUrlQuery("platformTypeIdentifier", *query.PlatformType))
	}
	if query.LandingZone != nil {
		options = append(options, internal.WithUrlQuery("landingZoneIdentifier", *query.LandingZone))
	}
	if query.PlatformTenant != nil {
		options = append(options, internal.WithUrlQuery("platformTenantId", *query.PlatformTenant))
	}
	return c.meshObject.List(ctx, options...)
}

func (c meshTenantV4Client) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}

func (tenant *MeshTenantV4) CreationSuccessful() (done bool, err error) {
	switch {
	case tenant == nil:
		err = fmt.Errorf("tenant not found after creation")
	case tenant.Spec.PlatformTenantId != nil && *tenant.Spec.PlatformTenantId != "":
		// Creation is complete (platformTenantId is set and not empty)
		done = true
	}
	return
}

func (tenant *MeshTenantV4) DeletionSuccessful() (done bool, err error) {
	return tenant == nil, nil
}
