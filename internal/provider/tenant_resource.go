package provider

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/poll"
)

var (
	_ resource.Resource                 = &tenantResource{}
	_ resource.ResourceWithConfigure    = &tenantResource{}
	_ resource.ResourceWithImportState  = &tenantResource{}
	_ resource.ResourceWithUpgradeState = &tenantResource{}
)

func NewTenantResource() resource.Resource {
	return &tenantResource{}
}

// tenantResource is the unsuffixed, stable meshTenant resource. It runs on the ref-based meshTenant
// (v4) body, migrating existing v3 state via an UpgradeState.
type tenantResource struct {
	meshTenantClient client.MeshTenantClient
}

func (r *tenantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *tenantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshTenantClient = client.Tenant
	})...)
}

func (r *tenantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:             1,
		MarkdownDescription: "Manages a `meshTenant`.",
		Attributes:          tenantBodyAttributes(),
	}
}

func (r *tenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[tenantResourceModel](ctx, req.Plan, &resp.Diagnostics, tenantConverterOptions().Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := client.MeshTenantCreate{
		Metadata: client.MeshTenantCreateMetadata{
			OwnedByProject:   plan.Metadata.OwnedByProject,
			OwnedByWorkspace: plan.Metadata.OwnedByWorkspace,
		},
		Spec: client.MeshTenantCreateSpec{
			PlatformRef:      plan.Spec.PlatformRef,
			PlatformTenantId: plan.Spec.PlatformTenantId,
			LandingZoneRef:   plan.Spec.LandingZoneRef,
			Quotas:           plan.Spec.Quotas,
		},
	}

	tenant, err := r.meshTenantClient.Create(ctx, &createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating tenant", fmt.Sprintf("Could not create tenant, unexpected error: %s", err.Error()))
		return
	}

	if plan.WaitForCompletion {
		err := poll.AtMostFor(30*time.Minute, r.meshTenantClient.ReadFunc(tenant.Metadata.Uuid), poll.WithLastResultTo(&tenant)).
			Until(ctx, (*client.MeshTenant).CreationSuccessful)
		if err != nil {
			resp.Diagnostics.AddError("Failed to await tenant creation", err.Error())
			return
		}
	}

	model := tenantResourceModelFromDto(tenant, plan.Spec.Quotas, plan.WaitForCompletion)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, model, tenantConverterOptions()...)...)
}

func (r *tenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state := generic.Get[tenantResourceModel](ctx, req.State, &resp.Diagnostics, tenantConverterOptions().Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := r.meshTenantClient.Read(ctx, state.Metadata.Uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading tenant", fmt.Sprintf("Could not read tenant with uuid %s, unexpected error: %s", state.Metadata.Uuid, err.Error()))
		return
	}

	if tenant == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// spec.quotas is Optional (not computed), so preserve the configured value from state rather than
	// the backend's (possibly landing-zone-defaulted) spec.quotas.
	model := tenantResourceModelFromDto(tenant, state.Spec.Quotas, state.WaitForCompletion)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, model, tenantConverterOptions()...)...)
}

func (r *tenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	opts := tenantConverterOptions().Append(generic.WithSetUnknownValueToZero())
	plan := generic.Get[tenantResourceModel](ctx, req.Plan, &resp.Diagnostics, opts...)
	state := generic.Get[tenantResourceModel](ctx, req.State, &resp.Diagnostics, opts...)
	if resp.Diagnostics.HasError() {
		return
	}

	// wait_for_completion is a provider-only toggle (no API call), so a change to just it is allowed
	// and simply written back to state. Every other tenant attribute is either immutable
	// (RequiresReplace) or computed (UseStateForUnknown, so it equals state in the plan), so any
	// remaining diff is an unsupported in-place update.
	normalized := state
	normalized.WaitForCompletion = plan.WaitForCompletion
	if !reflect.DeepEqual(plan, normalized) {
		resp.Diagnostics.AddError(
			"Tenants can't be updated",
			"Unsupported operation: a tenant can't be updated in place; only wait_for_completion may be changed. "+
				"The meshTenant API is create/delete only. In particular, quotas can only be set at creation — "+
				"change a live tenant's quotas via a quota request in the meshStack panel (Tenant > Settings > "+
				"Quotas), which is subject to platform-operator approval, not through Terraform.",
		)
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, tenantConverterOptions()...)...)
}

func (r *tenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	state := generic.Get[tenantResourceModel](ctx, req.State, &resp.Diagnostics, tenantConverterOptions().Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Metadata.Uuid
	err := r.meshTenantClient.Delete(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting tenant", fmt.Sprintf("Could not delete tenant with uuid %s, unexpected error: %s", uuid, err.Error()))
		return
	}

	if state.WaitForCompletion {
		if err := poll.AtMostFor(30*time.Minute, r.meshTenantClient.ReadFunc(uuid)).
			Until(ctx, (*client.MeshTenant).DeletionSuccessful); err != nil {
			resp.Diagnostics.AddError("Failed to await tenant deletion", err.Error())
			return
		}
	}
}

// ImportState accepts either a tenant UUID or the legacy `workspace.project.platform.location`
// composite identifier (the shape the v3 meshstack_tenant used, where `platform.location` is the full
// platform identifier), resolving the latter to a uuid via the list endpoint.
func (r *tenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ".")

	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("uuid"), parts[0])...)
	case 4:
		if parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
			resp.Diagnostics.AddError("Incomplete Import Identifier", fmt.Sprintf("Encountered empty import identifier field. Got: %q", req.ID))
			return
		}
		workspace, project := parts[0], parts[1]
		platform := parts[2] + "." + parts[3]
		tenant, err := r.listSingleTenant(ctx, workspace, project, platform)
		if err != nil {
			resp.Diagnostics.AddError("Failed to import tenant", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("uuid"), tenant.Metadata.Uuid)...)
	default:
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected either a tenant UUID or an identifier with format workspace.project.platform.location. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_completion"), types.BoolValue(true))...)
}

func (r *tenantResource) UpgradeState(_ context.Context) map[int64]resource.StateUpgrader {
	priorSchema := tenantV0Schema()
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &priorSchema,
			StateUpgrader: r.upgradeFromV0,
		},
	}
}

// upgradeFromV0 migrates the legacy v3 meshstack_tenant state (identifier-based, schema version 0) to
// the ref-based v4 body. The v4 body drops platform_identifier, so we resolve the tenant via the list
// endpoint (using the old workspace/project/platform_identifier) and populate uuid + refs from the
// API. Configure runs before UpgradeResourceState, so the client is available here.
func (r *tenantResource) upgradeFromV0(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior tenantV0Model
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := r.listSingleTenant(ctx,
		prior.Metadata.OwnedByWorkspace.ValueString(),
		prior.Metadata.OwnedByProject.ValueString(),
		prior.Metadata.PlatformIdentifier.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upgrade meshstack_tenant state from v3 to v4", err.Error())
		return
	}

	// spec.quotas is Optional (not computed) and echoes the configured value; a migrated config that
	// omits quotas plans null, so carry null here (not the backend's effective quotas, often an empty
	// set) to avoid a spurious spec.quotas diff that would route to the unsupported tenant Update.
	model := tenantResourceModelFromDto(tenant, nil, true)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, model, tenantConverterOptions()...)...)
}

// listSingleTenant resolves the single active (not soft-deleted) tenant for a workspace/project/platform
// composite, erroring if zero or more than one match.
func (r *tenantResource) listSingleTenant(ctx context.Context, workspace, project, platform string) (*client.MeshTenant, error) {
	tenants, err := r.meshTenantClient.List(ctx, &client.MeshTenantQuery{
		Workspace: workspace,
		Project:   &project,
		Platform:  &platform,
	})
	if err != nil {
		return nil, err
	}

	// The backend list returns only active tenants by default (soft-deleted and marked-for-deletion
	// are excluded), so no client-side lifecycle filter is needed.
	if len(tenants) != 1 {
		return nil, fmt.Errorf("expected exactly one active tenant for %s.%s.%s, found %d", workspace, project, platform, len(tenants))
	}
	return &tenants[0], nil
}

// --- legacy v0 (v3) schema, kept only for the state upgrader ---

type tenantV0Model struct {
	Metadata tenantV0Metadata `tfsdk:"metadata"`
	Spec     tenantV0Spec     `tfsdk:"spec"`
}

type tenantV0Metadata struct {
	OwnedByWorkspace   types.String `tfsdk:"owned_by_workspace"`
	OwnedByProject     types.String `tfsdk:"owned_by_project"`
	PlatformIdentifier types.String `tfsdk:"platform_identifier"`
	DeletedOn          types.String `tfsdk:"deleted_on"`
	AssignedTags       types.Map    `tfsdk:"assigned_tags"`
}

type tenantV0Spec struct {
	LocalId               types.String `tfsdk:"local_id"`
	LandingZoneIdentifier types.String `tfsdk:"landing_zone_identifier"`
	Quotas                types.List   `tfsdk:"quotas"`
}

func tenantV0Schema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace":  schema.StringAttribute{Required: true},
					"owned_by_project":    schema.StringAttribute{Required: true},
					"platform_identifier": schema.StringAttribute{Required: true},
					"deleted_on":          schema.StringAttribute{Computed: true},
					"assigned_tags": schema.MapAttribute{
						ElementType: types.ListType{ElemType: types.StringType},
						Computed:    true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"local_id":                schema.StringAttribute{Optional: true, Computed: true},
					"landing_zone_identifier": schema.StringAttribute{Optional: true, Computed: true},
					"quotas": schema.ListNestedAttribute{
						Optional: true,
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key":   schema.StringAttribute{Computed: true},
								"value": schema.Int64Attribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

// tenantBodyAttributes returns the ref-based meshTenant (v4) body schema attributes for the unsuffixed
// meshstack_tenant resource.
func tenantBodyAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ref": meshRefByUuid(meshRefOptions{
			Kind:        client.MeshObjectKind.Tenant,
			Description: "Reference to this tenant, can be used as `target_ref` in building block resources.",
			Output:      true,
		}),

		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Metadata of the tenant. The `owned_by_workspace` and `owned_by_project` attributes must be set here.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"uuid": schema.StringAttribute{
					MarkdownDescription: "The unique identifier (UUID) of the tenant.",
					Computed:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"owned_by_workspace": schema.StringAttribute{
					MarkdownDescription: "The identifier of the workspace that the tenant belongs to.",
					Required:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				},
				"owned_by_project": schema.StringAttribute{
					MarkdownDescription: "The identifier of the project that the tenant belongs to.",
					Required:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				},
			},
		},

		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Tenant specification.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"platform_ref": meshRefByUuid(meshRefOptions{
					Kind:            client.MeshObjectKind.Platform,
					Description:     "Reference to the platform this tenant belongs to, identified by its uuid.",
					RequiresReplace: true,
				}),
				"platform_tenant_id": schema.StringAttribute{
					MarkdownDescription: "The identifier of the tenant on the platform (e.g. GCP project ID or Azure subscription ID). If this is not set, a new tenant will be created. If this is set, an existing tenant will be imported. Otherwise, this field will be empty until a successful replication has run.",
					Optional:            true,
					Computed:            true,
					PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				},
				"landing_zone_ref": meshRefByName(meshRefOptions{
					Kind:             client.MeshObjectKind.LandingZone,
					Description:      "Reference to the landing zone to assign to this tenant, identified by its name (the landing zone identifier).",
					OptionalComputed: true,
					RequiresReplace:  true,
				}),
				"quotas": schema.SetNestedAttribute{
					MarkdownDescription: "Quotas to apply to the tenant at creation. If omitted, the landing zone's " +
						"default quotas apply. Set only at creation: " +
						"the meshTenant API cannot update a tenant, so changing this on an existing tenant is rejected. " +
						"To change a live tenant's quotas, file a quota request in the meshStack panel " +
						"(Tenant > Settings > Quotas), which is subject to platform-operator approval.",
					Optional: true,
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"key":   schema.StringAttribute{Required: true},
							"value": schema.Int64Attribute{Required: true},
						},
					},
				},
			},
		},

		"status": schema.SingleNestedAttribute{
			MarkdownDescription: "Tenant status.",
			Computed:            true,
			PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
			Attributes: map[string]schema.Attribute{
				"tenant_identifier": schema.StringAttribute{
					MarkdownDescription: "Fully-qualified identifier of the tenant: the owning workspace, project and platform (instance) identifiers joined by dots (`<workspace>.<project>.<platform>.<location>`).",
					Computed:            true,
				},
				"platform_type_identifier": schema.StringAttribute{
					MarkdownDescription: "Identifier of the tenant's platform type — the kind of platform (e.g. `aws`, `azure`), not the specific platform instance the tenant lives on.",
					Computed:            true,
				},
				"platform_workspace_id": schema.StringAttribute{
					MarkdownDescription: "For platforms that represent a workspace as a platform-side container (e.g. a Cloud Foundry Organization or an OpenStack Domain), the platform's own id of that container (an id assigned by the external platform, not a meshWorkspace identifier). Null for platforms with no such concept or until the tenant has been replicated.",
					Computed:            true,
				},
				"tags": schema.MapAttribute{
					MarkdownDescription: "Tags assigned to this tenant.",
					ElementType:         types.ListType{ElemType: types.StringType},
					Computed:            true,
				},
			},
		},

		"wait_for_completion": schema.BoolAttribute{
			MarkdownDescription: "Wait for tenant creation/deletion to complete. Note that tenant creation is considered complete when `spec.platformTenantId` is set and not necessarily when replication is finished. Defaults to `true`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
	}
}
