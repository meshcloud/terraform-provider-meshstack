package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &buildingBlockDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDataSource{}
)

func NewBuildingBlockDataSource() datasource.DataSource {
	return &buildingBlockDataSource{}
}

type buildingBlockDataSource struct {
	client *client.MeshStackProviderClient
}

func (d *buildingBlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (d *buildingBlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single Building Block by UUID.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Building block datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshBuildingBlock`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshBuildingBlock"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Building Block metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the Building Block.",
						Required:            true,
					},
					"definition_uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the Building Block Definition this Building Block is based on.",
						Computed:            true,
					},
					"definition_version": schema.Int64Attribute{
						MarkdownDescription: "Version number of the Building Block Definition this Building Block is based on",
						Computed:            true,
					},
					"tenant_identifier": schema.StringAttribute{
						MarkdownDescription: "Full tenant identifier of the tenant this Building Block is assigned to.",
						Computed:            true,
					},
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp of Building Block creation.",
						Computed:            true,
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "For deleted Building Blocks: timestamp of deletion.",
						Computed:            true,
					},
					"marked_for_deletion_by": schema.StringAttribute{
						MarkdownDescription: "For deleted Building Blocks: user who requested deletion.",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building Block specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the Building Block as shown in meshPanel.",
						Computed:            true,
					},

					"inputs": buildingBlockCombinedInputs(),

					"parent_building_blocks": schema.ListNestedAttribute{
						MarkdownDescription: "List of parent Building Blocks.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"buildingblock_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent Building Block.",
									Computed:            true,
								},
								"definition_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent Building Block definition.",
									Computed:            true,
								},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current Building Block status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						MarkdownDescription: "Execution status. One of `WAITING_FOR_DEPENDENT_INPUT`, `WAITING_FOR_OPERATOR_INPUT`, `PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`.",
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"WAITING_FOR_DEPENDENT_INPUT", "WAITING_FOR_OPERATOR_INPUT", "PENDING", "IN_PROGRESS", "SUCCEEDED", "FAILED"}...),
						},
					},
					"outputs": buildingBlockOutputs(),
				},
			},
		},
	}
}

func (d *buildingBlockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *buildingBlockDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := d.client.ReadBuildingBlock(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block", err.Error())
	}

	if bb == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("api_version"), bb.ApiVersion)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("kind"), bb.Kind)...)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata"), bb.Metadata)...)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)...)

	inputs := make(map[string]buildingBlockIoModel)
	for _, input := range bb.Spec.Inputs {
		value, err := toResourceModel(&input)

		if err != nil {
			resp.Diagnostics.AddError("Error processing input", err.Error())
			return
		}

		inputs[input.Key] = *value
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), inputs)...)

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("status"), bb.Status.Status)...)

	outputs := make(map[string]buildingBlockOutputModel)
	for _, output := range bb.Status.Outputs {
		value, err := toResourceModel(&output)

		if err != nil {
			resp.Diagnostics.AddError("Error processing output", err.Error())
			return
		}

		outputs[output.Key] = value.toOutputModel()
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("outputs"), outputs)...)

}
