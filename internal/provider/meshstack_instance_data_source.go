package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &meshStackInstanceDataSource{}
	_ datasource.DataSourceWithConfigure = &meshStackInstanceDataSource{}
)

func NewMeshStackInstanceDataSource() datasource.DataSource {
	return &meshStackInstanceDataSource{}
}

type meshStackInstanceDataSource struct {
	meshInfoClient client.MeshInfoClient
}

func (d *meshStackInstanceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (d *meshStackInstanceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Information about the meshStack instance this provider is configured against.",

		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "URL of the meshStack API this provider is configured against.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Version of the meshStack instance.",
				Computed:            true,
			},
			"enabled_feature_flags": schema.SetAttribute{
				MarkdownDescription: "Feature flags enabled on this meshStack instance. Currently the only possible entry is `four_eyes_role_approval` (the four-eyes principle / role approval).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"metadata": schema.MapAttribute{
				MarkdownDescription: "Git commit SHA of each meshStack subsystem making up this instance, keyed by subsystem name.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"admin_workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "Identifier of the admin (partner) workspace, the only valid owner for admin-only resources such as `meshstack_integration` Entra ID integrations.",
				Computed:            true,
			},
		},
	}
}

func (d *meshStackInstanceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshInfoClient = client.MeshInfo
	})...)
}

func (d *meshStackInstanceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	info, err := d.meshInfoClient.Read(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read meshStack instance information", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, info)...)
}
