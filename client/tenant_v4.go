package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types"
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
	Workspace      string  `json:"workspaceIdentifier"`
	Project        *string `json:"projectIdentifier"`
	Platform       *string `json:"platformIdentifier"`
	PlatformType   *string `json:"platformTypeIdentifier"`
	LandingZone    *string `json:"landingZoneIdentifier"`
	PlatformTenant *string `json:"platformTenantId"`
}

type MeshTenantV4Client interface {
	Read(ctx context.Context, uuid string) (*MeshTenantV4, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshTenantV4, error)
	List(ctx context.Context, query MeshTenantV4Query) ([]MeshTenantV4, error)
	Create(ctx context.Context, tenant *MeshTenantV4Create) (*MeshTenantV4, error)
	Delete(ctx context.Context, uuid string) error
}

type meshTenantV4Client struct {
	meshObject internal.MeshObjectClient[MeshTenantV4]
}

func newTenantV4Client(ctx context.Context, httpClient internal.HttpClient) MeshTenantV4Client {
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

func (c meshTenantV4Client) List(ctx context.Context, query MeshTenantV4Query) ([]MeshTenantV4, error) {
	return c.meshObject.List(ctx, internal.WithUrlQuery(query))
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

type MeshTenant struct {
	Metadata MeshTenantMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantSpec     `json:"spec" tfsdk:"spec"`
	Status   MeshTenantStatus   `json:"status" tfsdk:"status"`
}

type MeshTenantMetadata struct {
	Uuid             string `json:"uuid" tfsdk:"uuid"`
	OwnedByProject   string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshTenantSpec struct {
	PlatformRef      UuidRef                    `json:"platformRef" tfsdk:"platform_ref"`
	PlatformTenantId *string                    `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	LandingZoneRef   *NamedRef                  `json:"landingZoneRef" tfsdk:"landing_zone_ref"`
	Quotas           types.Set[MeshTenantQuota] `json:"quotas" tfsdk:"quotas"`
}

// MeshTenantStatus has no quotas field; quotas are part of the tenant spec, not its status.
type MeshTenantStatus struct {
	TenantIdentifier       string              `json:"tenantIdentifier" tfsdk:"tenant_identifier"`
	PlatformTypeIdentifier string              `json:"platformTypeIdentifier" tfsdk:"platform_type_identifier"`
	PlatformWorkspaceId    *string             `json:"platformWorkspaceId" tfsdk:"platform_workspace_id"`
	Tags                   map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshTenantQuota struct {
	Key   string `json:"key" tfsdk:"key"`
	Value int64  `json:"value" tfsdk:"value"`
}

type MeshTenantCreate struct {
	Metadata MeshTenantCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshTenantCreateSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTenantCreateMetadata struct {
	OwnedByProject   string `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshTenantCreateSpec struct {
	PlatformRef      UuidRef                    `json:"platformRef" tfsdk:"platform_ref"`
	LandingZoneRef   *NamedRef                  `json:"landingZoneRef" tfsdk:"landing_zone_ref"`
	PlatformTenantId *string                    `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	Quotas           types.Set[MeshTenantQuota] `json:"quotas" tfsdk:"quotas"`
}

type MeshTenantQuery struct {
	Workspace      string  `json:"workspaceIdentifier"`
	Project        *string `json:"projectIdentifier"`
	Platform       *string `json:"platformIdentifier"`
	PlatformType   *string `json:"platformTypeIdentifier"`
	LandingZone    *string `json:"landingZoneIdentifier"`
	PlatformTenant *string `json:"platformTenantId"`
}

type MeshTenantClient interface {
	Read(ctx context.Context, uuid string) (*MeshTenant, error)
	ReadFunc(uuid string) func(ctx context.Context) (*MeshTenant, error)
	List(ctx context.Context, query MeshTenantQuery) ([]MeshTenant, error)
	Create(ctx context.Context, tenant *MeshTenantCreate) (*MeshTenant, error)
	Delete(ctx context.Context, uuid string) error
}

type meshTenantClient struct {
	meshObject internal.MeshObjectClient[MeshTenant]
}

func newTenantClient(ctx context.Context, httpClient internal.HttpClient) MeshTenantClient {
	return meshTenantClient{internal.NewMeshObjectClient[MeshTenant](ctx, httpClient, "v4-preview")}
}

func (c meshTenantClient) Read(ctx context.Context, uuid string) (*MeshTenant, error) {
	return c.ReadFunc(uuid)(ctx)
}

func (c meshTenantClient) ReadFunc(uuid string) func(ctx context.Context) (*MeshTenant, error) {
	return func(ctx context.Context) (*MeshTenant, error) {
		return c.meshObject.Get(ctx, uuid)
	}
}

func (c meshTenantClient) Create(ctx context.Context, tenant *MeshTenantCreate) (*MeshTenant, error) {
	return c.meshObject.Post(ctx, tenant)
}

func (c meshTenantClient) List(ctx context.Context, query MeshTenantQuery) ([]MeshTenant, error) {
	return c.meshObject.List(ctx, internal.WithUrlQuery(query))
}

func (c meshTenantClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}

func (tenant *MeshTenant) CreationSuccessful() (done bool, err error) {
	switch {
	case tenant == nil:
		err = fmt.Errorf("tenant not found after creation")
	case tenant.Spec.PlatformTenantId != nil && *tenant.Spec.PlatformTenantId != "":
		// Creation is complete (platformTenantId is set and not empty)
		done = true
	}
	return
}

func (tenant *MeshTenant) DeletionSuccessful() (done bool, err error) {
	return tenant == nil, nil
}
