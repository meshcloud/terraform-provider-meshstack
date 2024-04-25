package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
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

type buildingBlockDataSourceModel struct {
	Metadata buildingBlockMetadataModel `tfsdk:"metadata"`
}

type buildingBlockMetadataModel struct {
	Uuid           types.String `tfsdk:"uuid"`
	DefinitionUuid types.String `tfsdk:"definition_uuid"`
}

func (d *buildingBlockDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_buildingblock"
}

func (d *buildingBlockDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "BuildingBlock data source",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Description: "Metadata",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						Required: true,
					},
					"definition_uuid": schema.StringAttribute{
						Computed: true,
					},
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
	var state buildingBlockDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Metadata.Uuid.ValueString()
	bb, err := d.client.ReadBuildingBlock(uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read buildingblock", err.Error())
	}

	state.Metadata.DefinitionUuid = types.StringValue(bb.Metadata.DefinitionUuid)

	diags := resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}
