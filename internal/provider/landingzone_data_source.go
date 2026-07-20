package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &landingZoneDataSource{}
	_ datasource.DataSourceWithConfigure = &landingZoneDataSource{}
)

func NewLandingZoneDataSource() datasource.DataSource {
	return &landingZoneDataSource{}
}

type landingZoneDataSource struct {
	meshLandingZoneClient client.MeshLandingZoneClient
}

func (d *landingZoneDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_landingzone"
}

// Schema defines the schema for the data source.
func (d *landingZoneDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single landing zone by identifier.",

		Attributes: map[string]schema.Attribute{
			"ref": meshRefByName(meshRefOptions{
				Kind:        client.MeshObjectKind.LandingZone,
				Description: "Reference to this landing zone, can be used as `landing_zone_ref` in tenant resources. The landing zone name is only unique together with its platform, so a `meshstack_tenant` references both `platform_ref` and `landing_zone_ref`.",
				Output:      true,
			}),
			"metadata": landingZoneMetadataDataSourceSchema(false),
			"spec":     landingZoneSpecDataSourceSchema(),
			"status":   landingZoneStatusDataSourceSchema(),
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *landingZoneDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshLandingZoneClient = client.LandingZone
	})...)
}

// Read refreshes the Terraform state with the latest data.
func (d *landingZoneDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	landingZone, err := d.meshLandingZoneClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read landing zone '%s'", name),
			err.Error(),
		)
		return
	}

	if landingZone == nil {
		resp.Diagnostics.AddError(
			"Landing zone not found",
			fmt.Sprintf("The requested landingZone '%s' was not found.", name),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, landingZoneModelFrom(landingZone))...)
}
