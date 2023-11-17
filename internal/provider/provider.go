package provider

import (
	"context"
	"net/http"
	"net/url"
	"time"

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

// TODO this will be an abstraction that does the login call, get a token and then use this token in the Auth header.
type MeshStackProviderClient struct {
	Url        *url.URL
	httpClient *http.Client
	apiKey     string
	apiSecret  string
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

	url, err := url.Parse(data.Endpoint.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Provider endpoint not valid.", "The value provided as the providers endpoint is not a valid URL.")
	} else {
		client := buildClient(url, data)
		resp.DataSourceData = client
		resp.ResourceData = client
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func buildClient(url *url.URL, model MeshStackProviderModel) *MeshStackProviderClient {
	client := MeshStackProviderClient{
		Url: url,
		httpClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		apiKey:    model.ApiKey.ValueString(),
		apiSecret: model.ApiKey.ValueString(),
	}

	return &client
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
