package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ datasource.DataSource              = &platformDataSource{}
	_ datasource.DataSourceWithConfigure = &platformDataSource{}
)

func NewPlatformDataSource() datasource.DataSource {
	return &platformDataSource{}
}

type platformDataSource struct {
	client *client.MeshStackProviderClient
}

func (d *platformDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Schema defines the schema for the data source.
func (d *platformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single platform by identifier.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform API version.",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
			},
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Platform metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Platform identifier.",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the platform.",
						Computed:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the platform.",
						Computed:            true,
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Platform specification.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the platform.",
						Computed:            true,
					},
					"platform_type": schema.StringAttribute{
						MarkdownDescription: "Type of the platform.",
						Computed:            true,
					},
					"config": schema.MapAttribute{
						MarkdownDescription: "Platform configuration.",
						ElementType:         types.StringType,
						Computed:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the platform.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *platformDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

type platformDataSourceModel struct {
	ApiVersion types.String               `tfsdk:"api_version"`
	Kind       types.String               `tfsdk:"kind"`
	Metadata   platformDataSourceMetadata `tfsdk:"metadata"`
	Spec       platformDataSourceSpec     `tfsdk:"spec"`
}

type platformDataSourceMetadata struct {
	Name      types.String `tfsdk:"name"`
	CreatedOn types.String `tfsdk:"created_on"`
	DeletedOn types.String `tfsdk:"deleted_on"`
}

type platformDataSourceSpec struct {
	DisplayName  types.String `tfsdk:"display_name"`
	PlatformType types.String `tfsdk:"platform_type"`
	Config       types.Map    `tfsdk:"config"`
	Tags         types.Map    `tfsdk:"tags"`
}

func (d *platformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data platformDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platform, err := d.client.ReadPlatform(data.Metadata.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading platform",
			"Could not read platform, unexpected error: "+err.Error(),
		)
		return
	}

	if platform == nil {
		resp.Diagnostics.AddError(
			"Platform not found",
			fmt.Sprintf("Platform with name '%s' not found.", data.Metadata.Name.ValueString()),
		)
		return
	}

	// Convert configuration to string map for data source
	configurationMap := make(map[string]string)
	for k, v := range platform.Spec.Config {
		if strVal, ok := v.(string); ok {
			configurationMap[k] = strVal
		} else {
			configurationMap[k] = fmt.Sprintf("%v", v)
		}
	}

	data.ApiVersion = types.StringValue(platform.ApiVersion)
	data.Kind = types.StringValue(platform.Kind)
	data.Metadata.Name = types.StringValue(platform.Metadata.Name)
	data.Metadata.CreatedOn = types.StringValue(platform.Metadata.CreatedOn)
	data.Spec.DisplayName = types.StringValue(platform.Spec.DisplayName)
	data.Spec.PlatformType = types.StringValue(platform.Spec.PlatformType)

	if platform.Metadata.DeletedOn != nil {
		data.Metadata.DeletedOn = types.StringValue(*platform.Metadata.DeletedOn)
	} else {
		data.Metadata.DeletedOn = types.StringNull()
	}

	if len(configurationMap) > 0 {
		configMap, diags := types.MapValueFrom(ctx, types.StringType, configurationMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Spec.Config = configMap
	} else {
		data.Spec.Config = types.MapNull(types.StringType)
	}

	if len(platform.Spec.Tags) > 0 {
		tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, platform.Spec.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Spec.Tags = tagsMap
	} else {
		data.Spec.Tags = types.MapNull(types.ListType{ElemType: types.StringType})
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
