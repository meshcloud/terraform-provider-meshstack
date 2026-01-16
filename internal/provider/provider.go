package provider

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/logging"
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

const (
	envKeyMeshstackEndpoint  = "MESHSTACK_ENDPOINT"
	envKeyMeshstackApiKey    = "MESHSTACK_API_KEY"
	envKeyMeshstackApiSecret = "MESHSTACK_API_SECRET"
)

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	client.SetLogger(logging.TerraformClientLogger{MessagePrefix: "client: "})
	var data MeshStackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	providerClient, diags := newProviderClient(ctx, data, p.version)
	resp.Diagnostics.Append(diags...)
	resp.DataSourceData = providerClient
	resp.ResourceData = providerClient
}

func configureProviderClient(providerData any, consumer func(client client.Client)) (diags diag.Diagnostics) {
	if providerData == nil {
		// do nothing as Terraform calls Configure without providerData
		return
	}
	if providerClient, ok := providerData.(client.Client); ok {
		consumer(providerClient)
	} else {
		diags.AddError(
			"Unexpected Provider Client type",
			fmt.Sprintf("Expected type client.Client, got: %T. Please report this issue to the provider developers.", providerData),
		)
	}
	return
}

func newProviderClient(ctx context.Context, data MeshStackProviderModel, providerVersion string) (providerClient client.Client, diags diag.Diagnostics) {
	var endpoint string
	if !data.Endpoint.IsNull() && !data.Endpoint.IsUnknown() {
		endpoint = data.Endpoint.ValueString()
	} else {
		var ok bool
		endpoint, ok = os.LookupEnv(envKeyMeshstackEndpoint)
		if !ok {
			diags.AddError("Provider endpoint missing.", "Set provider.meshstack.endpoint or use MESHSTACK_ENDPOINT environment variable.")
			return
		}
	}

	parsedEndpoint, err := url.Parse(endpoint)
	if err != nil {
		diags.AddError("Provider endpoint not valid.", "The value provided as the providers endpoint is not a valid URL.")
		return
	}

	var apiKey string
	if !data.ApiKey.IsNull() && !data.ApiKey.IsUnknown() {
		apiKey = data.ApiKey.ValueString()
	} else {
		var ok bool
		apiKey, ok = os.LookupEnv(envKeyMeshstackApiKey)
		if !ok {
			diags.AddError("Provider API key missing.", "Set provider.meshstack.apikey or use MESHSTACK_API_KEY environment variable.")
			return
		}
	}

	var apiSecret string
	if !data.ApiSecret.IsNull() && !data.ApiSecret.IsUnknown() {
		apiSecret = data.ApiSecret.ValueString()
	} else {
		var ok bool
		apiSecret, ok = os.LookupEnv(envKeyMeshstackApiSecret)
		if !ok {
			diags.AddError("Provider API secret missing.", "Set provider.meshstack.apisecret or use MESHSTACK_API_SECRET environment variable.")
			return
		}
	}

	userAgent := fmt.Sprintf("terraform-provider-meshstack/%s", providerVersion)
	providerClient = client.New(ctx, parsedEndpoint, userAgent, apiKey, apiSecret)
	return
}

func (p *MeshStackProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewTenantResource,
		NewTenantV4Resource,
		NewProjectUserBindingResource,
		NewProjectGroupBindingResource,
		NewWorkspaceUserBindingResource,
		NewWorkspaceGroupBindingResource,
		NewWorkspaceResource,
		NewBuildingBlockResource,
		NewBuildingBlockV2Resource,
		NewTagDefinitionResource,
		NewLandingZoneResource,
		NewPlatformResource,
		NewPaymentMethodResource,
		NewLocationResource,
		NewPlatformTypeResource,
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
		NewTenantV4DataSource,
		NewLandingZoneDataSource,
		NewPlatformDataSource,
		NewPaymentMethodDataSource,
		NewIntegrationsDataSource,
		NewPlatformTypesDataSource,
		NewPlatformTypeDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &MeshStackProvider{
			version: version,
		}
	}
}
