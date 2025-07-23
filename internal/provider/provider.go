package provider

import (
	"context"
	"net/url"
	"os"

	"github.com/meshcloud/terraform-provider-meshstack/client"

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
				Optional:            true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "API Key to authenticate against the meshStack API",
				Optional:            true,
			},
			"apisecret": schema.StringAttribute{
				MarkdownDescription: "API Secret to authenticate against the meshStack API",
				Optional:            true,
			},
		},
	}
}

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MeshStackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	endpoint, ok := os.LookupEnv("MESHSTACK_ENDPOINT")
	if !ok {
		if data.Endpoint.IsNull() || data.Endpoint.IsUnknown() {
			resp.Diagnostics.AddError("Provider endpoint missing.", "Set provider.meshstack.endpoint or use MESHSTACK_ENDPOINT environment variable.")
			return
		}
		endpoint = data.Endpoint.ValueString()
	}

	url, err := url.Parse(endpoint)
	if err != nil {
		resp.Diagnostics.AddError("Provider endpoint not valid.", "The value provided as the providers endpoint is not a valid URL.")
		return
	}

	apiKey, ok := os.LookupEnv("MESHSTACK_API_KEY")
	if !ok {
		if data.ApiKey.IsNull() || data.ApiKey.IsUnknown() {
			resp.Diagnostics.AddError("Provider API key missing.", "Set provider.meshstack.apikey or use MESHSTACK_API_KEY environment variable.")
			return
		}
		apiKey = data.ApiKey.ValueString()
	}

	apiSecret, ok := os.LookupEnv("MESHSTACK_API_SECRET")
	if !ok {
		if data.ApiSecret.IsNull() || data.ApiSecret.IsUnknown() {
			resp.Diagnostics.AddError("Provider API secret missing.", "Set provider.meshstack.apisecret or use MESHSTACK_API_SECRET environment variable.")
			return
		}
		apiSecret = data.ApiSecret.ValueString()
	}

	client, err := client.NewClient(url, apiKey, apiSecret)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create client.", err.Error())
		return
	}
	resp.DataSourceData = client
	resp.ResourceData = client

}

func (p *MeshStackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewTenantResource,
		NewProjectUserBindingResource,
		NewProjectGroupBindingResource,
		NewWorkspaceResource,
		NewBuildingBlockResource,
		NewBuildingBlockV2Resource,
		NewTagDefinitionResource,
	}
}

func (p *MeshStackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBuildingBlockDataSource,
		NewBuildingBlockV2DataSource,
		NewProjectDataSource,
		NewProjectsDataSource,
		NewProjectUserBindingDataSource,
		NewProjectGroupBindingDataSource,
		NewWorkspaceDataSource,
		NewTenantDataSource,
		NewTagDefinitionDataSource,
		NewTagDefinitionsDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MeshStackProvider{
			version: version,
		}
	}
}
