package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/credential"
	"github.com/meshcloud/meshstack-cli/pkg/meshstack"
	"github.com/meshcloud/meshstack-cli/pkg/profile"

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
	Profile   types.String `tfsdk:"profile"`
	Workspace types.String `tfsdk:"workspace"`
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
			// Each description is the setting's own Help() — the text the meshStack CLI prints for
			// the same setting — plus, where there is one, a sentence that is true of Terraform
			// alone. A fact about the setting belongs in the declaration, so that the two front
			// ends cannot describe it differently.
			"endpoint": schema.StringAttribute{
				MarkdownDescription: meshstack.Endpoint.Help(),
				Optional:            true,
			},
			"profile": schema.StringAttribute{
				MarkdownDescription: profile.Name.Help() +
					"\n\nA block holding only `profile` is a complete configuration.",
				Optional: true,
			},
			"workspace": schema.StringAttribute{
				MarkdownDescription: meshstack.Workspace.Help(),
				Optional:            true,
			},
			"apikey": schema.StringAttribute{
				MarkdownDescription: credential.ApiKeyId.Help() +
					"\n\nSetting this here with the secret in `MESHSTACK_API_SECRET` keeps the secret out of the configuration and out of state.",
				Optional: true,
			},
			"apisecret": schema.StringAttribute{
				MarkdownDescription: credential.ApiSecret.Help() +
					"\n\nA value set here is written to Terraform state. The warning about a stored secret winning is a log record, which `TF_LOG=WARN` shows.",
				Optional:  true,
				Sensitive: true,
			},
			"apitoken": schema.StringAttribute{
				MarkdownDescription: credential.ApiToken.Help() +
					"\n\nA value set here is written to Terraform state.",
				Optional:  true,
				Sensitive: true,
			},
		},
	}
}

func (p *MeshStackProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// The meshStack CLI logs through the slog default logger throughout — its pkg/ packages and
	// the API client alike — so without this bridge its records land on stderr in a format
	// terraform does not expect. It is the only logging seam the CLI has.
	slog.SetDefault(slog.New(logging.SlogHandler{MessagePrefix: "meshstack: "}))
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

func newProviderClient(ctx context.Context, data MeshStackProviderModel, providerVersion string) (providerClient client.Client, diagnostics diag.Diagnostics) {
	// One package answers "who am I, against what, in which workspace" for both the provider
	// and the meshStack CLI, so both apply the same precedence — block, then environment,
	// then profile — and both renew through the same file lock. That lock is the reason to
	// share rather than to copy: keycloak rotates a refresh token on every refresh and ends
	// the whole session when one is reused, so a `terraform apply` racing a `meshstack`
	// command would otherwise destroy the user's login.
	session, err := auth.ResolveSession(ctx, auth.ResolveSessionOptions{Settings: blockSource{data: data}})
	if err != nil {
		diagnostics.Append(problemDiagnostics("Failed to resolve meshStack credentials.", err)...)
		return
	}
	if err := session.RequireWorkspace(); err != nil {
		diagnostics.Append(problemDiagnostics("meshStack workspace missing.", err)...)
		return
	}

	// Mint here rather than at the first request. A plan that only creates resources reads
	// nothing, so an expired login would otherwise pass the plan and fail the apply — the
	// point at which terraform has already told the user what it is about to do. This is the
	// provider's call and not pkg/auth's, because the meshStack CLI must stay lazy: `meshstack
	// profile view` and `meshstack auth logout` have to work when the credential is dead.
	if _, err := session.BearerToken(ctx); err != nil {
		diagnostics.Append(problemDiagnostics("Failed to authenticate against meshStack.", err)...)
		return
	}

	userAgent := fmt.Sprintf("terraform-provider-meshstack/%s", providerVersion)
	providerClient, err = session.Client(ctx, userAgent)
	if err != nil {
		diagnostics.Append(problemDiagnostics("Failed to create meshStack client.", err)...)
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
