package provider

import (
	"context"
	"net/http"

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
	resp.TypeName = "meshStack"
	resp.Version = p.version
}

func (p *MeshStackProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            false,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: "API Key to authenticate against the meshStack API",
				Optional:            false,
			},
			"apisecret": schema.StringAttribute{
				MarkdownDescription: "API Secret to authenticate against the meshStack API",
				Optional:            false,
			},
		},
	}
}

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data MeshStackProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// TODO validate config and append errors is case there are any e.g.
	// resp.Diagnostics.AddAttributeError()

	// TODO create client e.g. like this:
	// Example client configuration for data sources and resources

	client := http.DefaultClient
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *MeshStackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewExampleResource,
	}
}

func (p *MeshStackProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MeshStackProvider{
			version: version,
		}
	}
}
