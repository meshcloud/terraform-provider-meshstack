package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &tenantV4Resource{}
	_ resource.ResourceWithConfigure   = &tenantV4Resource{}
	_ resource.ResourceWithImportState = &tenantV4Resource{}
)

type tenantV4ResourceModel struct {
	ApiVersion types.String                  `tfsdk:"api_version"`
	Kind       types.String                  `tfsdk:"kind"`
	Metadata   tenantV4ResourceMetadataModel `tfsdk:"metadata"`
	Spec       tenantV4ResourceSpecModel     `tfsdk:"spec"`
	Status     types.Object                  `tfsdk:"status"`
}

type tenantV4ResourceMetadataModel struct {
	Uuid                types.String `tfsdk:"uuid"`
	OwnedByWorkspace    types.String `tfsdk:"owned_by_workspace"`
	OwnedByProject      types.String `tfsdk:"owned_by_project"`
	CreatedOn           types.String `tfsdk:"created_on"`
	DeletedOn           types.String `tfsdk:"deleted_on"`
	MarkedForDeletionOn types.String `tfsdk:"marked_for_deletion_on"`
}

type tenantV4ResourceSpecModel struct {
	PlatformIdentifier    types.String `tfsdk:"platform_identifier"`
	PlatformTenantId      types.String `tfsdk:"platform_tenant_id"`
	LandingZoneIdentifier types.String `tfsdk:"landing_zone_identifier"`
	Quotas                types.Set    `tfsdk:"quotas"`
}

type tenantV4ResourceStatusModel struct {
	TenantName                  types.String `tfsdk:"tenant_name"`
	PlatformTypeIdentifier      types.String `tfsdk:"platform_type_identifier"`
	PlatformWorkspaceIdentifier types.String `tfsdk:"platform_workspace_identifier"`
	Tags                        types.Map    `tfsdk:"tags"`
	Quotas                      types.Set    `tfsdk:"quotas"`
}

func NewTenantV4Resource() resource.Resource {
	return &tenantV4Resource{}
}

type tenantV4Resource struct {
	client *client.MeshStackProviderClient
}

func (r *tenantV4Resource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_v4"
}

func (r *tenantV4Resource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *tenantV4Resource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a `meshTenant` with API version 4.\n\n~> **Note:** This resource is in preview and may change in the near future.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "API version of the tenant resource.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "The kind of the meshObject, always `meshTenant`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshTenant"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the tenant. The `owned_by_workspace` and `owned_by_project` attributes must be set here.",
				Required:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "The unique identifier (UUID) of the tenant.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The identifier of the workspace that the tenant belongs to.",
						Required:            true,
					},
					"owned_by_project": schema.StringAttribute{
						MarkdownDescription: "The identifier of the project that the tenant belongs to.",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "The creation timestamp of the meshTenant (e.g. `2020-12-22T09:37:43Z`).",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "The deletion timestamp of the tenant (e.g. `2020-12-22T09:37:43Z`).",
						Computed:            true,
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "The timestamp when the tenant was marked for deletion (e.g. `2020-12-22T09:37:43Z`).",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant specification.",
				Required:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"platform_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the target platform.",
						Required:            true,
					},
					"platform_tenant_id": schema.StringAttribute{
						MarkdownDescription: "The identifier of the tenant on the platform (e.g. GCP project ID or Azure subscription ID). If this is not set, a new tenant will be created. If this is set, an existing tenant will be imported. Otherwise, this field will be empty until a successful replication has run.",
						Optional:            true,
						Computed:            true,
					},
					"landing_zone_identifier": schema.StringAttribute{
						MarkdownDescription: "The identifier of the landing zone to assign to this tenant.",
						Optional:            true,
					},
					"quotas": schema.SetNestedAttribute{
						MarkdownDescription: "Landing zone quota settings will be applied by default but can be changed here.",
						Optional:            true,
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
				Attributes: map[string]schema.Attribute{
					"tenant_name": schema.StringAttribute{
						MarkdownDescription: "The full tenant name, a concatenation of the workspace identifier, project identifier and platform identifier.",
						Computed:            true,
					},
					"platform_type_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the platform type.",
						Computed:            true,
					},
					"platform_workspace_identifier": schema.StringAttribute{
						MarkdownDescription: "Some platforms create representations of workspaces, in such cases this will contain the identifier of the workspace on the platform.",
						Computed:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags assigned to this tenant.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
					"quotas": schema.SetNestedAttribute{
						MarkdownDescription: "The effective quotas applied to the tenant.",
						Computed:            true,
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

func (r *tenantV4Resource) setStateFromResponse(ctx context.Context, tenant *client.MeshTenantV4, knownQuotas types.Set, state *tfsdk.State, diags *diag.Diagnostics) {
	diags.Append(state.SetAttribute(ctx, path.Root("api_version"), tenant.ApiVersion)...)
	diags.Append(state.SetAttribute(ctx, path.Root("kind"), tenant.Kind)...)

	diags.Append(state.SetAttribute(ctx, path.Root("metadata"), tenant.Metadata)...)

	spec := tenantV4ResourceSpecModel{
		PlatformIdentifier:    types.StringValue(tenant.Spec.PlatformIdentifier),
		PlatformTenantId:      types.StringPointerValue(tenant.Spec.PlatformTenantId),
		LandingZoneIdentifier: types.StringPointerValue(tenant.Spec.LandingZoneIdentifier),
		Quotas:                knownQuotas,
	}
	diags.Append(state.SetAttribute(ctx, path.Root("spec"), spec)...)

	quotaAttributeTypes := map[string]attr.Type{
		"key":   types.StringType,
		"value": types.Int64Type,
	}
	quotasStatus, d := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: quotaAttributeTypes}, tenant.Spec.Quotas)
	diags.Append(d...)

	tagsValue, d := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, tenant.Status.Tags)
	diags.Append(d...)

	status := tenantV4ResourceStatusModel{
		TenantName:                  types.StringValue(tenant.Status.TenantName),
		PlatformTypeIdentifier:      types.StringValue(tenant.Status.PlatformTypeIdentifier),
		PlatformWorkspaceIdentifier: types.StringPointerValue(tenant.Status.PlatformWorkspaceIdentifier),
		Tags:                        tagsValue,
		Quotas:                      quotasStatus,
	}
	diags.Append(state.SetAttribute(ctx, path.Root("status"), status)...)
}

func (r *tenantV4Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tenantV4ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var platformTenantId *string
	if !plan.Spec.PlatformTenantId.IsNull() && !plan.Spec.PlatformTenantId.IsUnknown() {
		platformTenantId = plan.Spec.PlatformTenantId.ValueStringPointer()
	}

	var landingZoneIdentifier *string
	if !plan.Spec.LandingZoneIdentifier.IsNull() && !plan.Spec.LandingZoneIdentifier.IsUnknown() {
		landingZoneIdentifier = plan.Spec.LandingZoneIdentifier.ValueStringPointer()
	}

	var quotas []client.MeshTenantQuota
	if !plan.Spec.Quotas.IsNull() && !plan.Spec.Quotas.IsUnknown() {
		resp.Diagnostics.Append(plan.Spec.Quotas.ElementsAs(ctx, &quotas, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	createRequest := client.MeshTenantV4Create{
		Metadata: client.MeshTenantV4CreateMetadata{
			OwnedByProject:   plan.Metadata.OwnedByProject.ValueString(),
			OwnedByWorkspace: plan.Metadata.OwnedByWorkspace.ValueString(),
		},
		Spec: client.MeshTenantV4CreateSpec{
			PlatformIdentifier:    plan.Spec.PlatformIdentifier.ValueString(),
			PlatformTenantId:      platformTenantId,
			LandingZoneIdentifier: landingZoneIdentifier,
			Quotas:                &quotas,
		},
	}

	tenant, err := r.client.CreateTenantV4(&createRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tenant",
			fmt.Sprintf("Could not create tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	r.setStateFromResponse(ctx, tenant, plan.Spec.Quotas, &resp.State, &resp.Diagnostics)
}

func (r *tenantV4Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Tenants can't be updated", "Unsupported operation: tenant can't be updated.")
}

func (r *tenantV4Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	var quotas types.Set
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("quotas"), &quotas)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := r.client.ReadTenantV4(uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tenant",
			fmt.Sprintf("Could not read tenant with uuid %s, unexpected error: %s", uuid.ValueString(), err.Error()),
		)
		return
	}

	if tenant == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.setStateFromResponse(ctx, tenant, quotas, &resp.State, &resp.Diagnostics)
}

func (r *tenantV4Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tenantV4ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Metadata.Uuid.ValueString()

	err := r.client.DeleteTenantV4(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tenant",
			fmt.Sprintf("Could not delete tenant with uuid %s, unexpected error: %s", uuid, err.Error()),
		)
		return
	}
}

func (r *tenantV4Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
