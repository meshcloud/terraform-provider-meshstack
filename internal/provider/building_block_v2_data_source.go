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
	_ datasource.DataSource              = &buildingBlockV2DataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockV2DataSource{}
)

func NewBuildingBlockV2DataSource() datasource.DataSource {
	return &buildingBlockV2DataSource{}
}

type buildingBlockV2DataSource struct {
	client *client.MeshStackProviderClient
}

func (d *buildingBlockV2DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_v2"
}

func (d *buildingBlockV2DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	mkIoMap := func() schema.MapNestedAttribute {
		return schema.MapNestedAttribute{
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"value_string": schema.StringAttribute{
						Computed: true,
						Validators: []validator.String{stringvalidator.ExactlyOneOf(
							path.MatchRelative().AtParent().AtName("value_string"),
							path.MatchRelative().AtParent().AtName("value_single_select"),
							path.MatchRelative().AtParent().AtName("value_file"),
							path.MatchRelative().AtParent().AtName("value_int"),
							path.MatchRelative().AtParent().AtName("value_bool"),
							path.MatchRelative().AtParent().AtName("value_list"),
							path.MatchRelative().AtParent().AtName("value_code"),
						)},
					},
					"value_single_select": schema.StringAttribute{Computed: true},
					"value_file":          schema.StringAttribute{Computed: true},
					"value_int":           schema.Int64Attribute{Computed: true},
					"value_bool":          schema.BoolAttribute{Computed: true},
					"value_list": schema.StringAttribute{
						MarkdownDescription: "JSON encoded list of objects.",
						Computed:            true,
					},
					"value_code": schema.StringAttribute{
						MarkdownDescription: "Code value.",
						Computed:            true,
					},
				},
			},
		}
	}

	inputs := mkIoMap()
	inputs.MarkdownDescription = "Contains all building block inputs. Each input has exactly one value attribute set according to its' type."

	outputs := mkIoMap()
	outputs.MarkdownDescription = "Building block outputs. Each output has exactly one value attribute set."

	resp.Schema = schema.Schema{
		MarkdownDescription: "Single building block by UUID.\n\n~> **Note:** This resource is in preview. It's incomplete and will change in the near future.",

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
				MarkdownDescription: "Building block metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID which uniquely identifies the building block.",
						Required:            true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The workspace containing this building block.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp of building block creation.",
						Computed:            true,
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "For deleted building blocks: timestamp of deletion.",
						Computed:            true,
					},
					"marked_for_deletion_by": schema.StringAttribute{
						MarkdownDescription: "For deleted building blocks: user who requested deletion.",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building block specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name for the building block as shown in meshPanel.",
						Computed:            true,
					},

					"building_block_definition_version_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block definition this building block is based on.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the building block definition.",
								Computed:            true,
							},
						},
					},

					"target_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "References the building block target. Depending on the building block definition this will be a workspace or a tenant",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Target kind for this building block, depends on building block definition type. One of `meshTenant`, `meshWorkspace`.",
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"meshTenant", "meshWorkspace"}...),
								},
							},
							"uuid": schema.StringAttribute{
								MarkdownDescription: "UUID of the target tenant.",
								Computed:            true,
								Validators: []validator.String{stringvalidator.ExactlyOneOf(
									path.MatchRelative().AtParent().AtName("uuid"),
									path.MatchRelative().AtParent().AtName("identifier"),
								)},
							},
							"identifier": schema.StringAttribute{
								MarkdownDescription: "Identifier of the target workspace.",
								Computed:            true,
							},
						},
					},

					"inputs": inputs,

					"parent_building_blocks": schema.ListNestedAttribute{
						MarkdownDescription: "List of parent building blocks.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"buildingblock_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent building block.",
									Computed:            true,
								},
								"definition_uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the parent building block definition.",
									Computed:            true,
								},
							},
						},
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Current building block status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"status": schema.StringAttribute{
						MarkdownDescription: "Execution status. One of `WAITING_FOR_DEPENDENT_INPUT`, `WAITING_FOR_OPERATOR_INPUT`, `PENDING`, `IN_PROGRESS`, `SUCCEEDED`, `FAILED`.",
						Computed:            true,
						Validators: []validator.String{
							stringvalidator.OneOf([]string{"WAITING_FOR_DEPENDENT_INPUT", "WAITING_FOR_OPERATOR_INPUT", "PENDING", "IN_PROGRESS", "SUCCEEDED", "FAILED"}...),
						},
					},
					"force_purge": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether an operator has requested purging of this Building Block.",
						Computed:            true,
					},
					"outputs": outputs,
				},
			},
		},
	}
}

func (d *buildingBlockV2DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *buildingBlockV2DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	bb, err := d.client.ReadBuildingBlockV2(uuid)
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

	// Set all spec values except for inputs
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("building_block_definition_version_ref"), bb.Spec.BuildingBlockDefinitionVersionRef)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("target_ref"), bb.Spec.TargetRef)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)...)

	// Read inputs
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

	// Set all status values except for outputs
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("status"), bb.Status.Status)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("force_purge"), bb.Status.ForcePurge)...)

	// Read outputs
	outputs := make(map[string]buildingBlockIoModel)
	for _, output := range bb.Status.Outputs {
		value, err := toResourceModel(&output)

		if err != nil {
			resp.Diagnostics.AddError("Error processing output", err.Error())
			return
		}

		outputs[output.Key] = *value
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("status").AtName("outputs"), outputs)...)
}
