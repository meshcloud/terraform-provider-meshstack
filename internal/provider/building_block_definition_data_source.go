package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &buildingBlockDefinitionDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockDefinitionDataSource{}
)

func NewBuildingBlockDefinitionDataSource() datasource.DataSource {
	return &buildingBlockDefinitionDataSource{}
}

type buildingBlockDefinitionDataSource struct {
	meshBuildingBlockDefinitionClient        client.MeshBuildingBlockDefinitionClient
	meshBuildingBlockDefinitionVersionClient client.MeshBuildingBlockDefinitionVersionClient
}

func (d *buildingBlockDefinitionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_definition"
}

func (d *buildingBlockDefinitionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshBuildingBlockDefinitionClient = client.BuildingBlockDefinition
		d.meshBuildingBlockDefinitionVersionClient = client.BuildingBlockDefinitionVersion
	})...)
}

func (d *buildingBlockDefinitionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := buildingBlockDefinitionDataSourceAttributes()
	attributes["metadata"] = schema.SingleNestedAttribute{
		MarkdownDescription: "Building block definition metadata.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the building block definition.",
				Required:            true,
			},
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Workspace identifier owning this definition.",
				Computed:            true,
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single building block definition by UUID, with its versions and a `ref` for dependency refs. " +
			"This data source performs an additional API call to load all versions of the definition. " +
			"Use `meshstack_building_block_definitions` to look definitions up by workspace instead of by UUID." +
			"\n\n" +
			"**Cross-Workspace Access**: When accessing building block definitions from workspaces other than your own " +
			"using a workspace-scoped API key, the `content_hash` attribute will be null as it requires detailed version " +
			"information that is not accessible without workspace permissions. The `versions`, `version_latest`, and " +
			"`version_latest_release` attributes will still be populated with uuid, number, state, and workload_identity_federation." +
			"\n\n" + previewDisclaimer(),
		Attributes: attributes,
	}
}

func (d *buildingBlockDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	definition, err := d.meshBuildingBlockDefinitionClient.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block definition", err.Error())
		return
	}
	if definition == nil {
		resp.Diagnostics.AddError(
			"Building block definition not found",
			fmt.Sprintf("Building block definition with UUID '%s' was not found", uuid),
		)
		return
	}

	model := buildingBlockDefinitionDataSourceModelFrom(ctx, d.meshBuildingBlockDefinitionVersionClient, *definition, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
