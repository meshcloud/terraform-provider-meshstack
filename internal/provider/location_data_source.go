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
	_ datasource.DataSource              = &locationDataSource{}
	_ datasource.DataSourceWithConfigure = &locationDataSource{}
)

func NewLocationDataSource() datasource.DataSource {
	return &locationDataSource{}
}

type locationDataSource struct {
	meshLocationClient client.MeshLocationClient
}

func (d *locationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_location"
}

func (d *locationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single location by name.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Location identifier.",
						Required:            true,
					},
					"owned_by_workspace": computedString("Identifier of the workspace that owns this location."),
					"uuid":               computedString("Unique identifier of the location (server-generated)."),
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"display_name": computedString("The human-readable display name of the location."),
					"description":  computedString("The description of the location."),
				},
			},

			"status": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"is_public": schema.BoolAttribute{
						MarkdownDescription: "Indicates whether the location has any public platform instances associated with it.",
						Computed:            true,
					},
				},
			},

			"ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.Location, Description: "Reference to this location, can be used as input for `location_ref` in platform resources.", Output: true}),
		},
	}
}

func (d *locationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshLocationClient = client.Location
	})...)
}

func (d *locationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	location, err := d.meshLocationClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not read location '%s'", name), err.Error())
		return
	}

	if location == nil {
		resp.Diagnostics.AddError("Location not found", fmt.Sprintf("The requested location '%s' was not found.", name))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newLocationModel(location))...)
}
