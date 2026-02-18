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
			"metadata": platformTypeMetadataSchema(false),
			"spec":     platformTypeSpecSchema(),
			"status":   platformTypeStatusSchema(),
			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to this platform type, can be used as input for `platform_type_ref` in platform resources.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"kind": schema.StringAttribute{
						MarkdownDescription: "The kind of the object. Always `meshPlatformType`.",
						Computed:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Identifier of the platform type.",
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

	resp.Diagnostics.Append(resp.State.Set(ctx, newPlatformTypeModel(platformType))...)
}
