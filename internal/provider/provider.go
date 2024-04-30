package provider

import (
	"context"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure MeshStackProvider satisfies various provider interfaces.
var _ provider.Provider = &MeshStackProvider{}

type MeshStackProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

type MeshStackProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	ApiKey    types.String `tfsdk:"apikey"`
	ApiSecret types.String `tfsdk:"apisecret"`
}

func (p *MeshStackProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "meshstack"
	resp.Version = p.version
}

func (p *MeshStackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "URl of meshStack API, e.g. `https://api.my.meshstack.io`",
				Required:            true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "API Key to authenticate against the meshStack API",
				Required:            true,
			},
			"apisecret": schema.StringAttribute{
				MarkdownDescription: "API Secret to authenticate against the meshStack API",
				Required:            true,
			},
		},
	}
}

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MeshStackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	url, err := url.Parse(data.Endpoint.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Provider endpoint not valid.", "The value provided as the providers endpoint is not a valid URL.")
	} else {
		client, err := NewClient(url, data.ApiKey.ValueString(), data.ApiSecret.ValueString()) // TODO handle err
		if err != nil {
			resp.Diagnostics.AddError("Failed to create client.", err.Error())
		}
		resp.DataSourceData = client
		resp.ResourceData = client
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (p *MeshStackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}

func (p *MeshStackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBuildingBlockDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MeshStackProvider{
			version: version,
		}
	}
}
