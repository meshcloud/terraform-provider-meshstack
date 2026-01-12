package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &tenantV4DataSource{}
	_ datasource.DataSourceWithConfigure = &tenantV4DataSource{}
)

func NewTenantV4DataSource() datasource.DataSource {
	return &tenantV4DataSource{}
}

type tenantV4DataSource struct {
	MeshTenantV4 client.MeshTenantV4Client
}

func (d *tenantV4DataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tenant_v4"
}

func (d *tenantV4DataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.MeshTenantV4 = client.TenantV4
	})...)
}

func (d *tenantV4DataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches details of a single tenant by UUID.\n\n~> **Note:** This resource is in preview and may change in the near future.",
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Tenant datatype version",
				Computed:            true,
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshTenant`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshTenant"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the tenant.",
						Required:            true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace the tenant belongs to.",
						Computed:            true,
					},
					"owned_by_project": schema.StringAttribute{
						MarkdownDescription: "Identifier of the project the tenant belongs to.",
						Computed:            true,
					},
					"marked_for_deletion_on": schema.StringAttribute{
						MarkdownDescription: "Date when the tenant was marked for deletion.",
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
					"platform_tenant_id": schema.StringAttribute{
						MarkdownDescription: "Platform-specific tenant ID.",
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
					"tenant_name": schema.StringAttribute{
						MarkdownDescription: "Name of the tenant.",
						Computed:            true,
					},
					"platform_type_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the platform type.",
						Computed:            true,
					},
					"platform_workspace_identifier": schema.StringAttribute{
						MarkdownDescription: "Identifier of the platform workspace.",
						Computed:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags assigned to this tenant.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *tenantV4DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tenant, err := d.MeshTenantV4.Read(uuid)
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
