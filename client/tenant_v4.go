package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
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

// UnmarshalJSON fills the identifier-shaped spec from a ref-shaped meshTenant v4 payload.
//
// meshStack replaced spec.platformIdentifier and spec.landingZoneIdentifier with platformRef and
// landingZoneRef, which this deprecated identifier-based resource has no attribute for. Both fields
// would otherwise read back empty, and because both force replacement, a refresh makes Terraform
// plan the destruction and recreation of a live tenant. meshstack_tenant is the ref-based
// replacement; this keeps meshstack_tenant_v4 readable until a configuration has migrated.
//
// The landing zone comes straight off landingZoneRef.name. The platform identifier is not on the
// wire at all, so it is recovered from status.tenantName, which meshStack composes as
// "<workspace>.<project>.<platformIdentifier>".
func (t *MeshTenantV4) UnmarshalJSON(data []byte) error {
	type wire MeshTenantV4
	var target wire
	if err := json.Unmarshal(data, &target); err != nil {
		return err
	}
	*t = MeshTenantV4(target)

	var refs struct {
		Spec struct {
			PlatformRef    *UuidRef  `json:"platformRef"`
			LandingZoneRef *NamedRef `json:"landingZoneRef"`
		} `json:"spec"`
	}
	if err := json.Unmarshal(data, &refs); err != nil {
		return err
	}

	if t.Spec.LandingZoneIdentifier == nil && refs.Spec.LandingZoneRef != nil {
		t.Spec.LandingZoneIdentifier = &refs.Spec.LandingZoneRef.Name
	}
	if t.Spec.PlatformIdentifier == "" {
		t.Spec.PlatformIdentifier = platformIdentifierFromTenantName(
			t.Status.TenantName, t.Metadata.OwnedByWorkspace, t.Metadata.OwnedByProject)
	}

	return nil
}

// platformIdentifierFromTenantName strips the "<workspace>.<project>." prefix off a tenant name.
// It returns "" when the name does not carry that prefix, which leaves the caller with the same
// empty value it would have had without this recovery rather than a wrong platform identifier.
func platformIdentifierFromTenantName(tenantName, workspace, project string) string {
	prefix := workspace + "." + project + "."
	if !strings.HasPrefix(tenantName, prefix) {
		return ""
	}
	return strings.TrimPrefix(tenantName, prefix)
}

type MeshTenantV4Status struct {
	TenantName                  string              `json:"tenantName" tfsdk:"tenant_name"`
	PlatformTypeIdentifier      string              `json:"platformTypeIdentifier" tfsdk:"platform_type_identifier"`
	PlatformWorkspaceIdentifier *string             `json:"platformWorkspaceIdentifier" tfsdk:"platform_workspace_identifier"`
	Tags                        map[string][]string `json:"tags" tfsdk:"tags"`
	// AppliedQuotas are the effective quotas meshStack applied to the tenant as a key->value map,
	// distinct from the create-only spec.quotas which carries only the requested values. Each value is a
	// structured object (e.g. `{"limits.cpu": {"value": 4}}`) so the preview API can grow per-quota
	// fields without a breaking change to the map shape.
	AppliedQuotas map[string]AppliedQuotaValue `json:"appliedQuotas" tfsdk:"applied_quotas"`
	Lifecycle     MeshTenantLifecycle          `json:"lifecycle" tfsdk:"-"`
}

type TenantLifecycleState string

var (
	TenantLifecycleStates                 = enum.Enum[TenantLifecycleState]{}
	TenantLifecycleStateActive            = TenantLifecycleStates.Entry("ACTIVE")
	TenantLifecycleStateMarkedForDeletion = TenantLifecycleStates.Entry("MARKED_FOR_DELETION")
	TenantLifecycleStateDeleted           = TenantLifecycleStates.Entry("DELETED")
)

type MeshTenantLifecycle struct {
	State             enum.Entry[TenantLifecycleState] `json:"state" tfsdk:"-"`
	MarkedForDeletion *MeshTenantLifecycleAction       `json:"markedForDeletion" tfsdk:"-"`
}

type MeshTenantLifecycleAction struct {
	Timestamp string `json:"timestamp" tfsdk:"-"`
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
	return tenant == nil || tenant.Status.Lifecycle.State == TenantLifecycleStateDeleted, nil
}

func (tenant *MeshTenantV4) DeletionState() string {
	if tenant == nil {
		return tenantNotObserved
	}
	return tenantDeletionState(tenant.Status.Lifecycle)
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
	PlatformRef      UuidRef   `json:"platformRef" tfsdk:"platform_ref"`
	PlatformTenantId *string   `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	LandingZoneRef   *NamedRef `json:"landingZoneRef" tfsdk:"landing_zone_ref"`
	// RequestedQuotas is the preferred key->value form for requesting quotas at creation, e.g.
	// {"limits.cpu": {"value": 4}}. The backend does not return it on read (it is a create-time input),
	// so the resource echoes the configured value from state.
	RequestedQuotas map[string]RequestQuotaValue `json:"requestedQuotas" tfsdk:"requested_quotas"`
}

type MeshTenantStatus struct {
	TenantName             string              `json:"tenantName" tfsdk:"tenant_name"`
	PlatformTypeIdentifier string              `json:"platformTypeIdentifier" tfsdk:"platform_type_identifier"`
	PlatformWorkspaceId    *string             `json:"platformWorkspaceId" tfsdk:"platform_workspace_id"`
	Tags                   map[string][]string `json:"tags" tfsdk:"tags"`
	// AppliedQuotas are the effective quotas meshStack applied to the tenant as a key->value map, each
	// value a structured object (e.g. `{"limits.cpu": {"value": 4}}`). spec.requested_quotas carries
	// only the values requested at create (create-only); the effective quotas here can differ once
	// landing-zone defaults are merged in or an operator adjusts them, so drift is tracked against these.
	AppliedQuotas map[string]AppliedQuotaValue `json:"appliedQuotas" tfsdk:"applied_quotas"`
	Lifecycle     MeshTenantLifecycle          `json:"lifecycle" tfsdk:"-"`
}

// MeshTenantQuota is the {key, value} element of the deprecated list-form spec.quotas, superseded by
// the requested_quotas / applied_quotas maps. It is still the quota shape of the deprecated
// meshstack_tenant_v4 resource, so it carries no godoc deprecation marker.
type MeshTenantQuota struct {
	Key   string `json:"key" tfsdk:"key"`
	Value int64  `json:"value" tfsdk:"value"`
}

// RequestQuotaValue is a tenant quota value as requested at create time. The scalar is wrapped in an
// object (rather than a bare number) so the v4 preview API can grow per-quota fields — e.g. a unit —
// without a breaking change to the requested_quotas map shape.
//
// Its shape is identical to AppliedQuotaValue, deliberately so: the resource must echo the configured
// request in spec while reading effective values from status, and separate types turn mixing the two
// into a compile error rather than the requested-vs-applied conflation this map form fixes.
type RequestQuotaValue struct {
	Value int64 `json:"value" tfsdk:"value"`
}

// AppliedQuotaValue is a tenant quota value as actually applied by the backend. See RequestQuotaValue
// for why the two are not a single type.
type AppliedQuotaValue struct {
	Value int64 `json:"value" tfsdk:"value"`
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
	PlatformRef      UuidRef                      `json:"platformRef" tfsdk:"platform_ref"`
	LandingZoneRef   *NamedRef                    `json:"landingZoneRef" tfsdk:"landing_zone_ref"`
	PlatformTenantId *string                      `json:"platformTenantId" tfsdk:"platform_tenant_id"`
	RequestedQuotas  map[string]RequestQuotaValue `json:"requestedQuotas,omitempty" tfsdk:"requested_quotas"`
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
	return tenant == nil || tenant.Status.Lifecycle.State == TenantLifecycleStateDeleted, nil
}

func (tenant *MeshTenant) DeletionState() string {
	if tenant == nil {
		return tenantNotObserved
	}
	return tenantDeletionState(tenant.Status.Lifecycle)
}

const tenantNotObserved = "no successful read after the delete request"

func tenantDeletionState(lifecycle MeshTenantLifecycle) string {
	switch {
	case lifecycle.State == TenantLifecycleStateDeleted:
		return "DELETED"
	case lifecycle.State == TenantLifecycleStateMarkedForDeletion && lifecycle.MarkedForDeletion != nil:
		return fmt.Sprintf(
			"MARKED_FOR_DELETION since %s, awaiting deletion approval, cleanup of the tenant's resources, or the platform deletion the replicator confirms",
			lifecycle.MarkedForDeletion.Timestamp,
		)
	case lifecycle.State == TenantLifecycleStateMarkedForDeletion:
		return "MARKED_FOR_DELETION, awaiting deletion approval, cleanup of the tenant's resources, or the platform deletion the replicator confirms"
	default:
		return fmt.Sprintf("%s, meshStack accepted the delete request but has not acted on it", lifecycle.State)
	}
}
