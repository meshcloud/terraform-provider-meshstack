package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/pkg/login"

	"github.com/meshcloud/terraform-provider-meshstack/internal/util/logging"
)

// Ensure MeshStackProvider satisfies various provider interfaces.
var _ provider.ProviderWithFunctions = &MeshStackProvider{}

type MeshStackProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	// clientFactory is helpful when injecting mocked clients during testing,
	// by default, newProviderClient is run.
	clientFactory func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics)
}

type MeshStackProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	ApiKey    types.String `tfsdk:"apikey"`
	ApiSecret types.String `tfsdk:"apisecret"`
	ApiToken  types.String `tfsdk:"apitoken"`
}

func (p *MeshStackProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "meshstack"
	resp.Version = p.version
}

func (p *MeshStackProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
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
				Sensitive:           true,
			},
			"apitoken": schema.StringAttribute{
				MarkdownDescription: "API Token to authenticate against the meshStack API",
				Optional:            true,
				Sensitive:           true,
			},
		},
	}
}

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	client.SetLogger(logging.TerraformClientLogger{MessagePrefix: "client: "})
	var data MeshStackProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	providerClient, diags := p.clientFactory(ctx, data, p.version)
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
	// Provider block attributes outrank the environment. A null or unknown attribute
	// reads as the empty string, which Merge treats as "not configured". The variable
	// names come from pkg/login, so the provider and the meshStack CLI share one
	// definition of them.
	credentials := login.FromEnv().Merge(login.Credentials{
		Endpoint:  data.Endpoint.ValueString(),
		ApiKey:    data.ApiKey.ValueString(),
		ApiSecret: data.ApiSecret.ValueString(),
		ApiToken:  data.ApiToken.ValueString(),
	})

	if credentials.Endpoint == "" {
		diags.AddError("Provider endpoint missing.", fmt.Sprintf("Set provider.meshstack.endpoint or use %s environment variable.", login.EnvKeyEndpoint))
		return
	}
	parsedEndpoint, err := credentials.EndpointURL()
	if err != nil {
		diags.AddError("Provider endpoint not valid.", "The value provided as the providers endpoint is not a valid URL.")
		return
	}

	// Either apiToken or apiKey/apiSecret must be set for authorization against backend.
	// Terraform diagnostics name the attribute that is missing, so the key and the
	// secret are reported one by one instead of through Authorization's single error.
	if credentials.ApiToken == "" {
		if credentials.ApiKey == "" {
			diags.AddError("Provider API key missing.", fmt.Sprintf("Set provider.meshstack.apikey or use %s environment variable.", login.EnvKeyApiKey))
		}
		if credentials.ApiSecret == "" {
			diags.AddError("Provider API secret missing.", fmt.Sprintf("Set provider.meshstack.apisecret or use %s environment variable.", login.EnvKeyApiSecret))
		}
		if diags.HasError() {
			return
		}
	}
	auth, err := credentials.Authorization()
	if err != nil {
		diags.AddError("Failed to build meshStack authorization.", err.Error())
		return
	}

	userAgent := fmt.Sprintf("terraform-provider-meshstack/%s", providerVersion)
	providerClient, err = client.New(ctx, parsedEndpoint, userAgent, auth)
	if err != nil {
		diags.AddError("Failed to create meshStack client.", err.Error())
		return
	}
	return
}

func (p *MeshStackProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewTenantResource,
		NewProjectUserBindingResource,
		NewProjectGroupBindingResource,
		NewWorkspaceUserBindingResource,
		NewWorkspaceGroupBindingResource,
		NewWorkspaceResource,
		NewWorkspaceTagsResource,
		NewWorkspaceTagResource,
		NewBuildingblockResource,
		NewBuildingBlockV2Resource,
		NewBuildingBlockResource,
		NewBuildingBlockDefinitionResource,
		NewTagDefinitionResource,
		NewLandingZoneResource,
		NewPlatformResource,
		NewPaymentMethodResource,
		NewLocationResource,
		NewPlatformTypeResource,
		NewIntegrationResource,
		NewBuildingBlockRunnerResource,
		NewApiKeyResource,
	}
}

func (p *MeshStackProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewBuildingblockDataSource,
		NewBuildingBlockV2DataSource,
		NewBuildingBlockDataSource,
		NewBuildingBlocksDataSource,
		NewBuildingBlockDefinitionsDataSource,
		NewMeshStackInstanceDataSource,
		NewProjectDataSource,
		NewProjectsDataSource,
		NewProjectUserBindingDataSource,
		NewProjectGroupBindingDataSource,
		NewWorkspaceDataSource,
		NewTenantDataSource,
		NewTagDefinitionDataSource,
		NewTagDefinitionsDataSource,
		NewTenantsDataSource,
		NewLandingZoneDataSource,
		NewLandingZonesDataSource,
		NewPlatformDataSource,
		NewPlatformsDataSource,
		NewPaymentMethodDataSource,
		NewIntegrationsDataSource,
		NewPlatformTypesDataSource,
		NewPlatformTypeDataSource,
		NewServiceInstanceDataSource,
		NewServiceInstancesDataSource,
	}
}

func (p *MeshStackProvider) Functions(_ context.Context) []func() function.Function {
	return []func() function.Function{
		NewLoadImageFileFunction,
		NewLoadFileFunction,
		NewEncodeFileFunction,
		NewNonEphemeralSecretFunction,
	}
}

type providerOption func(*MeshStackProvider)

func New(version string, opts ...providerOption) func() provider.Provider {
	return func() provider.Provider {
		p := &MeshStackProvider{
			version:       version,
			clientFactory: newProviderClient,
		}
		for _, opt := range opts {
			opt(p)
		}
		return p
	}
}
