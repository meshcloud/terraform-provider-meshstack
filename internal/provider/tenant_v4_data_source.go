package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &tenantV4DataSource{}
	_ datasource.DataSourceWithConfigure = &tenantV4DataSource{}
)

func NewTenantV4DataSource() datasource.DataSource {
	return &tenantV4DataSource{}
}

type tenantV4DataSource struct {
	client *client.MeshStackProviderClient
}

func (d *tenantV4DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_v4"
}

func (d *tenantV4DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = client
}

func (d *tenantV4DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of a single tenant by UUID (v4).",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the tenant.",
				Required:            true,
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant metadata.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace the tenant belongs to.",
						Computed:            true,
					},
					"owned_by_project": schema.StringAttribute{
						MarkdownDescription: "Identifier of the project the tenant belongs to.",
						Computed:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "If the tenant has been submitted for deletion by a workspace manager, the date is shown here.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "The date the tenant was created.",
						Computed:            true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"platform_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the target platform.",
						Computed:            true,
					},
					"local_id": schema.StringAttribute{
						MarkdownDescription: "Tenant ID local to the platform.",
						Computed:            true,
					},
					"landing_zone_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of landing zone to assign to this tenant.",
						Computed:            true,
					},
					"quotas": schema.ListNestedAttribute{
						MarkdownDescription: "Set of applied tenant quotas.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key":   schema.StringAttribute{Computed: true},
								"value": schema.Int64Attribute{Computed: true},
							},
						},
					},
				},
			},
			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant status.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags assigned to this tenant.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
					"last_replicated": schema.StringAttribute{
						MarkdownDescription: "The last time the tenant was replicated.",
						Computed:            true,
					},
					"current_replication_status": schema.StringAttribute{
						MarkdownDescription: "The current replication status of the tenant.",
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *tenantV4DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("uuid"), &uuid)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := d.client.ReadTenantV4(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading tenant",
			fmt.Sprintf("Could not read tenant, unexpected error: %s", err.Error()),
		)
		return
	}

	if tenant == nil {
		resp.Diagnostics.AddError(
			"Error reading tenant",
			"Tenant not found",
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tenant)...)
}