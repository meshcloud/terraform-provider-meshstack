package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/iter"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &integrationsDataSource{}
)

// NewIntegrationsDataSource is a helper function to simplify the provider implementation.
func NewIntegrationsDataSource() datasource.DataSource {
	return &integrationsDataSource{}
}

// integrationsDataSourceItem shares integrationStatus with the resource, so the relocated redirect URL
// has one derivation for both.
type integrationsDataSourceItem struct {
	Metadata client.MeshIntegrationMetadata `tfsdk:"metadata"`
	Spec     client.MeshIntegrationSpec     `tfsdk:"spec"`
	Status   *integrationStatus             `tfsdk:"status"`
}

// integrationsDataSource is the data source implementation.
type integrationsDataSource struct {
	meshIntegrationClient client.MeshIntegrationClient
}

// Metadata returns the data source type name.
func (d *integrationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

// Schema defines the schema for the data source.
func (d *integrationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	workloadIdentityFederation := schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"issuer": schema.StringAttribute{
				Computed: true,
			},
			"subject": schema.StringAttribute{
				Computed: true,
			},
			"gcp": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"audience": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"aws": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"audience": schema.StringAttribute{
						Computed: true,
					},
					"thumbprint": schema.StringAttribute{
						Computed: true,
					},
				},
			},
			"azure": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"audience": schema.StringAttribute{
						Computed: true,
					},
				},
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve a list of integrations. " +
			"Includes integrations that belong to your workspace as well as built-in integrations (replicator and metering). " +
			"Platform administrators can retrieve any integration. " +
			"Sensitive fields are masked with a hash value for security purposes.",

		Attributes: map[string]schema.Attribute{
			"workload_identity_federation": schema.SingleNestedAttribute{
				MarkdownDescription: "Workload identity federation information for built-in integrations (replicator and metering).",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"replicator": workloadIdentityFederation,
					"metering":   workloadIdentityFederation,
				},
			},
			"integrations": schema.ListNestedAttribute{
				MarkdownDescription: "List of integrations. Each integration contains configuration for external CI/CD systems.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metadata": schema.SingleNestedAttribute{
							MarkdownDescription: "Metadata of the integration. Contains identifiers and ownership details.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"uuid":               computedString("UUID of the integration."),
								"owned_by_workspace": computedString("Identifier of the workspace that owns this integration. **Note**: Entra ID integrations can only be owned by the admin workspace."),
							},
						},
						"spec": schema.SingleNestedAttribute{
							MarkdownDescription: "Specification of the integration. Contains configuration settings specific to the integration type.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"display_name": computedString("Display name of the integration."),
								"config": schema.SingleNestedAttribute{
									MarkdownDescription: "Configuration for the integration. Specifies one of `github`, `gitlab`, `azuredevops`, or `entraid` integration types. Note that your meshStack does not support Entra ID integrations by default and needs to be configured explicitly to do so.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"github": schema.SingleNestedAttribute{
											MarkdownDescription: "GitHub integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"owner":           computedString("GitHub organization or user that owns the repositories."),
												"base_url":        computedString("Base URL of the GitHub instance (e.g., https://github.com for GitHub.com or your GitHub Enterprise URL)."),
												"app_id":          computedString("GitHub App ID for authentication."),
												"app_private_key": secret.DatasourceSchema(secret.DatasourceSchemaOptions{}),
												"runner_ref":      meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockRunner, Description: "Reference to the building block runner that executes GitHub workflows.", Output: true}),
											},
										},
										"gitlab": schema.SingleNestedAttribute{
											MarkdownDescription: "GitLab integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"base_url":   computedString("Base URL of the GitLab instance (e.g., https://gitlab.com or your self-hosted GitLab URL)."),
												"runner_ref": meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockRunner, Description: "Reference to the building block runner that executes GitLab pipelines.", Output: true}),
											},
										},
										"azuredevops": schema.SingleNestedAttribute{
											MarkdownDescription: "Azure DevOps integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"base_url":              computedString("Base URL of the Azure DevOps instance (e.g., https://dev.azure.com)."),
												"organization":          computedString("Azure DevOps organization name."),
												"personal_access_token": secret.DatasourceSchema(secret.DatasourceSchemaOptions{}),
												"runner_ref":            meshRefByUuid(meshRefOptions{Kind: client.MeshObjectKind.BuildingBlockRunner, Description: "Reference to the building block runner that executes Azure DevOps pipelines.", Output: true}),
											},
										},
										"entraid": schema.SingleNestedAttribute{
											MarkdownDescription: "Entra ID SSO integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"tenant_id":     computedString("Entra ID tenant ID."),
												"client_id":     computedString("Entra ID application (client) ID."),
												"client_secret": secret.DatasourceSchema(secret.DatasourceSchemaOptions{}),
												"idp_alias":     computedString("Alias of the identity provider backing this integration."),
											},
										},
									},
								},
							},
						},
						"status": schema.SingleNestedAttribute{
							MarkdownDescription: "Status information of the integration. System-managed state.",
							Computed:            true,
							Optional:            true,
							Attributes: map[string]schema.Attribute{
								"entraid": schema.SingleNestedAttribute{
									MarkdownDescription: "Derived state of an Entra ID integration.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"redirect_url": computedString("OAuth2 redirect URL, which meshStack derives from `idp_alias`."),
									},
								},
								"is_built_in":                  computedBool("Indicates whether this is a built-in integration (replicator or metering)."),
								"workload_identity_federation": workloadIdentityFederation,
							},
						},
					},
				},
			},
		},
	}
}

// Configure adds the provider configured client to the data source.
func (d *integrationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshIntegrationClient = client.Integration
	})...)
}

// Read refreshes the Terraform state with the latest data.
func (d *integrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	integrationDtos, err := iter.Collect(d.meshIntegrationClient.List(ctx, client.NoFilter{}))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read integrations, got error: %s", err))
		return
	}
	for _, integrationDto := range integrationDtos {
		if integrationDto.Status != nil && integrationDto.Status.IsBuiltIn {
			if integrationDto.Spec.Config.Type == "replicator" {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx,
					path.Root("workload_identity_federation").AtName("replicator"), integrationDto.Status.WorkloadIdentityFederation)...)
			}
			if integrationDto.Spec.Config.Type == "metering" {
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx,
					path.Root("workload_identity_federation").AtName("metering"), integrationDto.Status.WorkloadIdentityFederation)...)
			}
		}
	}
	items := make([]integrationsDataSourceItem, 0, len(integrationDtos))
	for i := range integrationDtos {
		items = append(items, integrationsDataSourceItem{
			Metadata: integrationDtos[i].Metadata,
			Spec:     integrationDtos[i].Spec,
			Status:   integrationStatusFromDto(&integrationDtos[i]),
		})
	}
	integrations, err := generic.ValueFrom(items, secret.WithDatasourceConverter())
	if err != nil {
		resp.Diagnostics.AddError("Value conversion error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integrations"), integrations)...)
}
