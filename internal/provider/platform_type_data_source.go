package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &platformTypeDataSource{}
	_ datasource.DataSourceWithConfigure = &platformTypeDataSource{}
)

func NewPlatformTypeDataSource() datasource.DataSource {
	return &platformTypeDataSource{}
}

type platformTypeDataSource struct {
	meshPlatformTypeClient client.MeshPlatformTypeClient
}

func (d *platformTypeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform_type"
}

func (d *platformTypeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single platform type by name.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "API version of meshPlatformType datatype.",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "Kind of meshObject. This is always meshPlatformType for this endpoint.",
				Computed:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the platform type",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the platform type",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the platform type",
						Computed:            true,
					},
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the platform type",
						Computed:            true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Specifications of the platform type",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the platform type",
						Computed:            true,
					},
					"category": schema.StringAttribute{
						MarkdownDescription: "Category of the platform type",
						Computed:            true,
					},
					"default_endpoint": schema.StringAttribute{
						MarkdownDescription: "Default endpoint for the platform type",
						Computed:            true,
					},
					"icon": schema.StringAttribute{
						MarkdownDescription: "Icon of the platform type",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *platformTypeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshPlatformTypeClient = client.PlatformType
	})...)
}

func (d *platformTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name string

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platformType, err := d.meshPlatformTypeClient.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform type '%s'", name),
			err.Error(),
		)
		return
	}

	if platformType == nil {
		resp.Diagnostics.AddError(
			"Platform type not found",
			fmt.Sprintf("The requested platform type '%s' was not found.", name),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, platformType)...)
}
