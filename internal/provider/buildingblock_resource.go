package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BuildingBlockResource{}
var _ resource.ResourceWithImportState = &BuildingBlockResource{}

func NewBuildingBlockResource() resource.Resource {
	return &BuildingBlockResource{}
}

type BuildingBlockResource struct {
	client *MeshStackProviderClient
}

type BuildingBlockResourceModel struct {
	Uuid              types.String `tfsdk:"uuid"`
	DefinitionUUid    types.String `tfsdk:"definition_uuid"`
	DefinitionVersion types.Int64  `tfsdk:"definition_version"`
	TenantIdentifier  types.String `tfsdk:"tenant_identifier"`
	DisplayName       types.String `tfsdk:"display_name"`
	Inputs            types.Map    `tfsdk:"inputs"`
	Parents           types.Set    `tfsdk:"parents"`
}

func (r *BuildingBlockResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (r *BuildingBlockResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Building Block",

		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the Building Block (generated)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"definition_uuid": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The UUID of the Building Block definition, that is used for this Building Block.",
			},
			"definition_version": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Version number of the Building Block definition, that is used for this Building Block.",
			},
			"tenant_identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The identifier of the tenant, this Building Block belongs to.",
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name of the Building Block.",
			},
			"inputs": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Assigned input values for the Building Block.",
			},
			"parents": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "The Building Blocks, that are parents of this Building Block.",
			},
		},
	}
}

func (r *BuildingBlockResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *BuildingBlockResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BuildingBlockResourceModel

	// read from PLAN
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO use client to create resource

	// save to STATE
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildingBlockResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BuildingBlockResourceModel

	// read from STATE
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := r.client.ReadBuildingBlock(data.Uuid.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("An error occured while contacting the meshObjects API.", err.Error())
		return
	}

	data.Uuid = basetypes.NewStringValue(bb.Metadata.Uuid)
	data.DefinitionUUid = basetypes.NewStringValue(bb.Metadata.DefinitionUuid)
	data.DefinitionVersion = basetypes.NewInt64Value(bb.Metadata.DefinitionVersion)
	data.TenantIdentifier = basetypes.NewStringValue(bb.Metadata.TenantIdentifier)

	data.DisplayName = basetypes.NewStringValue(bb.Spec.DisplayName)

	// TODO set data.Inputs
	// TODO set data.Parents

	// save to STATE
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildingBlockResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data BuildingBlockResourceModel

	// read from PLAN
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO use client to update resource

	// save to STATE
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BuildingBlockResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BuildingBlockResourceModel

	// read from STATE
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// TODO use client to delete resource
}

func (r *BuildingBlockResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
