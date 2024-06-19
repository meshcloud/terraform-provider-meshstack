package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &tenantResource{}
	_ resource.ResourceWithConfigure   = &tenantResource{}
	_ resource.ResourceWithImportState = &tenantResource{}
)

func NewTenantResource() resource.Resource {
	return &tenantResource{}
}

type tenantResource struct {
	client *client.MeshStackProviderClient
}

func (r *tenantResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant"
}

func (r *tenantResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *tenantResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single tenant by workspace, project, and platform.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Tenant datatype version",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshTenant`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshTenant"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant metadata. Workspace, project and platform of the target tenant must be set here.",
				Required:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace the tenant belongs to.",
						Required:            true,
					},
					"owned_by_project": schema.StringAttribute{
						MarkdownDescription: "Identifier of the project the tenant belongs to.",
						Required:            true,
					},
					"platform_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the target platform.",
						Required:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "If the tenant has been submitted for deletion by a workspace manager, the date is shown here (e.g. 2020-12-22T09:37:43Z).",
						Computed:            true,
					},
					"assigned_tags": schema.MapAttribute{
						MarkdownDescription: "Tags assigned to this tenant originating from workspace, payment method and project.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},

			// Making this optional would be nicer
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant specification.",
				Required:            true,
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.RequiresReplace()},
				Attributes: map[string]schema.Attribute{
					"local_id": schema.StringAttribute{
						MarkdownDescription: "Tenant ID local to the platform (e.g. GCP project ID, Azure subscription ID). Setting the local ID means that a tenant with this ID should be imported into meshStack. Not setting a local ID means that a new tenant should be created. Field will be empty until a successful replication has run.",
						Optional:            true,
						Computed:            true,
					},
					"landing_zone_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of landing zone to assign to this tenant.",
						Optional:            true,
						Computed:            true,
					},
					"quotas": schema.ListNestedAttribute{
						MarkdownDescription: "Set of applied tenant quotas. By default the landing zone quotas are applied to new tenants.",
						Optional:            true,
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

type tenantCreateMetadata struct {
	OwnedByProject     types.String `json:"ownedByProject" tfsdk:"owned_by_project"`
	OwnedByWorkspace   types.String `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	PlatformIdentifier types.String `json:"platformIdentifier" tfsdk:"platform_identifier"`
	AssignedTags       types.Map    `json:"assignedTags" tfsdk:"assigned_tags"`
	DeletedOn          types.String `json:"deletedOn" tfsdk:"deleted_on"`
}

type tenantCreateSpec struct {
	LocalId               types.String `json:"localId" tfsdk:"local_id"`
	LandingZoneIdentifier types.String `json:"landingZoneIdentifier" tfsdk:"landing_zone_identifier"`
	Quotas                types.List   `json:"quotas" tfsdk:"quotas"`
}

func (r *tenantResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var metadata tenantCreateMetadata
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)

	var spec tenantCreateSpec
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)...)

	var local_id *string
	if !spec.LocalId.IsUnknown() {
		local_id = spec.LocalId.ValueStringPointer()
	}

	var landing_zone_identifier *string
	if !spec.LandingZoneIdentifier.IsUnknown() {
		landing_zone_identifier = spec.LandingZoneIdentifier.ValueStringPointer()
	}

	var quotas []client.MeshTenantQuota
	if !spec.Quotas.IsNull() && !spec.Quotas.IsUnknown() {
		resp.Diagnostics.Append(spec.Quotas.ElementsAs(ctx, &quotas, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	create := client.MeshTenantCreate{
		Metadata: client.MeshTenantCreateMetadata{
			OwnedByProject:     metadata.OwnedByProject.ValueString(),
			OwnedByWorkspace:   metadata.OwnedByWorkspace.ValueString(),
			PlatformIdentifier: metadata.PlatformIdentifier.ValueString(),
		},
		Spec: client.MeshTenantCreateSpec{
			LocalId:               local_id,
			LandingZoneIdentifier: landing_zone_identifier,
			Quotas:                &quotas,
		},
	}

	tenant, err := r.client.CreateTenant(&create)

	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tenant",
			"Could not create tenant, unexpected error: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}

func (r *tenantResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// get workspace, project and platform to query for tenant
	var workspace, project, platform string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_project"), &project)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("platform_identifier"), &platform)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := r.client.ReadTenant(workspace, project, platform)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tenant", err.Error())
		return
	}

	if tenant == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}

func (r *tenantResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Tenants can't be updated", "Unsupported operation: tenants can't be updated.")
}

func (r *tenantResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state client.MeshTenant

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTenant(state.Metadata.OwnedByWorkspace, state.Metadata.OwnedByProject, state.Metadata.PlatformIdentifier)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tenant",
			"Could not delete tenant, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *tenantResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	identifier := strings.Split(req.ID, ".")

	for _, s := range identifier {
		if s == "" {
			resp.Diagnostics.AddError(
				"Incomplete Import Identifier",
				fmt.Sprintf("Encountered empty import identifier field. Got: %q", req.ID),
			)
			return
		}
	}

	if len(identifier) != 4 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace.project.location.platform Got: %q", req.ID),
		)
		return
	}

	platform := identifier[2] + "." + identifier[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), identifier[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("owned_by_project"), identifier[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("platform_identifier"), platform)...)
}
