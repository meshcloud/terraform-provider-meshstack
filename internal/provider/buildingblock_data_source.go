package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	mkIoList := func(desc string) schema.ListNestedAttribute {
		return schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: desc,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"key":        schema.StringAttribute{Computed: true},
					"value":      schema.StringAttribute{Computed: true},
					"value_type": schema.StringAttribute{Computed: true},
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
	// get UUID for BB we want to query from the request
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	bb, err := d.client.ReadBuildingBlock(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read buildingblock", err.Error())
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, bb)...)
}
