package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

func NewPlatformTypesDataSource() datasource.DataSource {
	return &platformTypesDataSource{}
}

type platformTypesDataSource struct {
	meshPlatformTypeClient client.MeshPlatformTypeClient
}

type platformTypesDataSourceModel struct {
	PlatformTypes   []client.MeshPlatformType `tfsdk:"platform_types"`
	Category        *string                   `tfsdk:"category"`
	LifecycleStatus *string                   `tfsdk:"lifecycle_status"`
}

func (d *platformTypesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform_types"
}

func (d *platformTypesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Platform types available in the meshStack installation.",

		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				MarkdownDescription: "Filter platform types by category",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"OPENSTACK", "CLOUDFOUNDRY", "SERVICEREGISTRY", "AWS",
						"OPENSHIFT", "KUBERNETES", "AZURE", "GCP",
						"AZURE_KUBERNETES_SERVICE", "AZURE_RESOURCE_GROUP",
						"CUSTOM", "GITHUB",
					),
				},
			},
			"lifecycle_status": schema.StringAttribute{
				MarkdownDescription: "Filter platform types by lifecycle status",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("ACTIVE", "DEACTIVATED"),
				},
			},
			"platform_types": schema.ListNestedAttribute{
				MarkdownDescription: "List of platform types",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
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
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "Name of the platform type",
									Computed:            true,
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
				},
			},
		},
	}
}

func (d *platformTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshPlatformTypeClient = client.PlatformType
	})...)
}

func (d *platformTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data platformTypesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	platformTypes, err := d.meshPlatformTypeClient.List(ctx, data.Category, data.LifecycleStatus)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read meshPlatformTypes", err.Error())
		return
	}

	data.PlatformTypes = platformTypes
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
