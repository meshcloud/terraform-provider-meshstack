package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource              = &buildingBlockRunnerDataSource{}
	_ datasource.DataSourceWithConfigure = &buildingBlockRunnerDataSource{}
)

func NewBuildingBlockRunnerDataSource() datasource.DataSource {
	return &buildingBlockRunnerDataSource{}
}

type buildingBlockRunnerDataSource struct {
	client client.MeshBuildingBlockRunnerClient
}

func (d *buildingBlockRunnerDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_building_block_runner"
}

func (d *buildingBlockRunnerDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		d.client = providerClient.BuildingBlockRunner
	})...)
}

func (d *buildingBlockRunnerDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	var runnerResource resource.SchemaResponse
	NewBuildingBlockRunnerResource().Schema(ctx, resource.SchemaRequest{}, &runnerResource)
	resourceAttributes := runnerResource.Schema.Attributes

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single building block runner by UUID. It is read-only and mirrors the `meshstack_building_block_runner` resource." +
			previewDisclaimer(),
		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the building block runner.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the runner.",
						Required:            true,
					},
					"owned_by_workspace": computedString("Identifier of the workspace that owns this runner."),
					"created_on":         computedString("Timestamp when the runner was created (ISO 8601)."),
				},
			},
			"spec":   computedDataSourceAttribute(resourceAttributes["spec"]),
			"status": computedDataSourceAttribute(resourceAttributes["status"]),
			"ref":    computedDataSourceAttribute(resourceAttributes["ref"]),
		},
	}
}

func (d *buildingBlockRunnerDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	runner, err := d.client.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read building block runner", fmt.Sprintf("Reading runner '%s' failed: %s", uuid, err.Error()))
		return
	}
	if runner == nil {
		resp.Diagnostics.AddError("Building block runner not found", fmt.Sprintf("Building block runner with UUID '%s' was not found", uuid))
		return
	}

	var state buildingBlockRunnerModel
	state.setFromClientDto(runner)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state)...)
}
