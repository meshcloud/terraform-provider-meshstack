package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var platformTypeCategories = []string{
	"OPENSTACK", "CLOUDFOUNDRY", "SERVICEREGISTRY", "AWS",
	"OPENSHIFT", "KUBERNETES", "AZURE", "GCP",
	"AZURE_KUBERNETES_SERVICE", "AZURE_RESOURCE_GROUP",
	"CUSTOM", "GITHUB",
}

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
		MarkdownDescription: "Platform types available in meshStack.",

		Attributes: map[string]schema.Attribute{
			"category": schema.StringAttribute{
				MarkdownDescription: "Filter platform types by category. Possible values: " + strings.Join(platformTypeCategories, ", ") + ".",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(platformTypeCategories...),
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
						"metadata": platformTypeMetadataSchema(true),
						"spec":     platformTypeSpecSchema(),
						"status":   platformTypeStatusSchema(),
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
