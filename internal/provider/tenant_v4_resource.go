package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &tenantResourceV4{}
	_ resource.ResourceWithConfigure   = &tenantResourceV4{}
	_ resource.ResourceWithImportState = &tenantResourceV4{}
)

func NewTenantResourceV4() resource.Resource {
	return &tenantResourceV4{}
}

type tenantResourceV4 struct {
	client *client.MeshStackProviderClient
}

func (r *tenantResourceV4) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_v4"
}

func (r *tenantResourceV4) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tenantResourceV4) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single tenant by workspace, project, and platform (v4).",

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
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the tenant.",
						Required:            true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace the tenant belongs to.",
						Required:            true,
					},
					"owned_by_project": schema.StringAttribute{
						MarkdownDescription: "Identifier of the project the tenant belongs to.",
						Required:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "If the tenant has been submitted for deletion by a workspace manager, the date is shown here (e.g. 2020-12-22T09:37:43Z).",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "The date the tenant was created (e.g. 2020-12-22T09:37:43Z).",
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

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags assigned to this tenant.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
					"last_replicated": schema.StringAttribute{
						MarkdownDescription: "The last time the tenant was replicated (e.g. 2020-12-22T09:37:43Z).",
						Computed:            true,
					},
					"current_replication_status": schema.StringAttribute{
						MarkdownDescription: "The current replication status of the tenant.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (r *tenantResourceV4) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var metadata client.MeshTenantCreateMetadataV4
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)

	var spec client.MeshTenantCreateSpecV4
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)...)

	if resp.Diagnostics.HasError() {
		return
	}

	create := client.MeshTenantCreateV4{
		Metadata: metadata,
		Spec:     spec,
	}

	tenant, err := r.client.CreateTenantV4(&create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tenant",
			fmt.Sprintf("Could not create tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}

func (r *tenantResourceV4) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := r.client.ReadTenantV4(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tenant",
			fmt.Sprintf("Could not read tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	if tenant == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}

func (r *tenantResourceV4) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var metadata client.MeshTenantCreateMetadataV4
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)

	var spec client.MeshTenantCreateSpecV4
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)...)

	if resp.Diagnostics.HasError() {
		return
	}

	update := client.MeshTenantCreateV4{
		Metadata: metadata,
		Spec:     spec,
	}

	tenant, err := r.client.CreateTenantV4(&update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating tenant",
			fmt.Sprintf("Could not update tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}

func (r *tenantResourceV4) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTenantV4(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tenant",
			fmt.Sprintf("Could not delete tenant, unexpected error: %s", err.Error()),
		)
		return
	}
}

func (r *tenantResourceV4) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	uuid := req.ID

	tenant, err := r.client.ReadTenantV4(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing tenant",
			fmt.Sprintf("Could not import tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	if tenant == nil {
		resp.Diagnostics.AddError(
			"Error importing tenant",
			"Tenant not found",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}
