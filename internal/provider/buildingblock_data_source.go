package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	client *MeshStackProviderClient
}

func (d *buildingBlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (d *buildingBlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	// Dynamic attributes are not supported as nested attributes, we use mutually exclusive fields for each possible value type instead.
	mkIoList := func(desc string) schema.ListNestedAttribute {
		return schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: desc,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"key":          schema.StringAttribute{Computed: true},
					"value_string": schema.StringAttribute{Computed: true},
					"value_int":    schema.Int64Attribute{Computed: true},
					"value_bool":   schema.BoolAttribute{Computed: true},
					"value_type":   schema.StringAttribute{Computed: true},
				},
			},
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Query a single Building Block by UUID.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Building Block datatype version",
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
				MarkdownDescription: "Building Block metadata. UUID of the target Building Block must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid":               schema.StringAttribute{Required: true},
					"definition_uuid":    schema.StringAttribute{Computed: true},
					"definition_version": schema.Int64Attribute{Computed: true},
					"tenant_identifier":  schema.StringAttribute{Computed: true},
					"force_purge":        schema.BoolAttribute{Computed: true},
					"created_on":         schema.StringAttribute{Computed: true},
					"marked_for_deletion_on": schema.StringAttribute{
						Computed: true,
					},
					"marked_for_deletion_by": schema.StringAttribute{
						Computed: true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Building Block specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{

					"display_name": schema.StringAttribute{Computed: true},
					"inputs":       mkIoList("List of Building Block inputs."),
					"parent_building_blocks": schema.ListNestedAttribute{
						Computed: true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"buildingblock_uuid": schema.StringAttribute{Computed: true},
								"definition_uuid":    schema.StringAttribute{Computed: true},
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
					"outputs": mkIoList("List of building block outputs."),
				},
			},
		},
	}
}

func (d *buildingBlockDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*MeshStackProviderClient)

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
	type io struct {
		Key         types.String `tfsdk:"key"`
		ValueString types.String `tfsdk:"value_string"`
		ValueInt    types.Int64  `tfsdk:"value_int"`
		ValueBool   types.Bool   `tfsdk:"value_bool"`
		ValueType   types.String `tfsdk:"value_type"`
	}

	mkIoList := func(ios *[]MeshBuildingBlockIO) *[]io {
		result := make([]io, 0)
		for _, input := range *ios {
			var valueString *string
			var valueInt *int64
			var valueBool *bool

			if input.ValueType == "STRING" {
				val := input.Value.(string)
				valueString = &val
			} else if input.ValueType == "INTEGER" {
				val := int64(input.Value.(float64))
				valueInt = &val
			} else if input.ValueType == "BOOLEAN" {
				val := input.Value.(bool)
				valueBool = &val
			}

			result = append(result, io{
				Key:         types.StringValue(input.Key),
				ValueString: types.StringPointerValue(valueString),
				ValueInt:    types.Int64PointerValue(valueInt),
				ValueBool:   types.BoolPointerValue(valueBool),
				ValueType:   types.StringValue(input.ValueType),
			})
		}
		return &result
	}

	// get UUID for BB we want to query from the request
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	bb, err := d.client.ReadBuildingBlock(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read buildingblock", err.Error())
	}

	// must set attributes individually to handle dynamic input/output types
	resp.State.SetAttribute(ctx, path.Root("api_version"), bb.ApiVersion)
	resp.State.SetAttribute(ctx, path.Root("kind"), bb.Kind)
	resp.State.SetAttribute(ctx, path.Root("metadata"), bb.Metadata)

	resp.State.SetAttribute(ctx, path.Root("spec").AtName("display_name"), bb.Spec.DisplayName)
	resp.State.SetAttribute(ctx, path.Root("spec").AtName("parent_building_blocks"), bb.Spec.ParentBuildingBlocks)
	if bb.Spec.Inputs != nil {
		resp.State.SetAttribute(ctx, path.Root("spec").AtName("inputs"), mkIoList(&bb.Spec.Inputs))
	}

	resp.State.SetAttribute(ctx, path.Root("status").AtName("status"), bb.Status.Status)
	if bb.Status.Outputs != nil {
		resp.State.SetAttribute(ctx, path.Root("status").AtName("outputs"), &bb.Status.Outputs)
	}
}
