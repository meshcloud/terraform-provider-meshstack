package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource = &integrationsDataSource{}
)

// NewIntegrationsDataSource is a helper function to simplify the provider implementation.
func NewIntegrationsDataSource() datasource.DataSource {
	return &integrationsDataSource{}
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
								"uuid": schema.StringAttribute{
									MarkdownDescription: "UUID of the integration.",
									Computed:            true,
								},
								"owned_by_workspace": schema.StringAttribute{
									MarkdownDescription: "Identifier of the workspace that owns this integration.",
									Computed:            true,
								},
							},
						},
						"spec": schema.SingleNestedAttribute{
							MarkdownDescription: "Specification of the integration. Contains configuration settings specific to the integration type.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"display_name": schema.StringAttribute{
									MarkdownDescription: "Display name of the integration.",
									Computed:            true,
								},
								"config": schema.SingleNestedAttribute{
									MarkdownDescription: "Configuration for the integration. Specifies one of github, gitlab, or azuredevops integration types.",
									Computed:            true,
									Attributes: map[string]schema.Attribute{
										"github": schema.SingleNestedAttribute{
											MarkdownDescription: "GitHub integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"owner": schema.StringAttribute{
													MarkdownDescription: "GitHub organization or user that owns the repositories.",
													Computed:            true,
												},
												"base_url": schema.StringAttribute{
													MarkdownDescription: "Base URL of the GitHub instance (e.g., https://github.com for GitHub.com or your GitHub Enterprise URL).",
													Computed:            true,
												},
												"app_id": schema.StringAttribute{
													MarkdownDescription: "GitHub App ID for authentication.",
													Computed:            true,
												},
												"app_private_key": secret.DatasourceSchema(secret.SchemaOptions{}),
												"runner_ref": schema.SingleNestedAttribute{
													MarkdownDescription: "Reference to the building block runner that executes GitHub workflows.",
													Computed:            true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															MarkdownDescription: "UUID of the meshBuildingBlockRunner.",
															Computed:            true,
														},
														"kind": schema.StringAttribute{
															MarkdownDescription: "meshObject type, always meshBuildingBlockRunner.",
															Computed:            true,
														},
													},
												},
											},
										},
										"gitlab": schema.SingleNestedAttribute{
											MarkdownDescription: "GitLab integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"base_url": schema.StringAttribute{
													MarkdownDescription: "Base URL of the GitLab instance (e.g., https://gitlab.com or your self-hosted GitLab URL).",
													Computed:            true,
												},
												"runner_ref": schema.SingleNestedAttribute{
													MarkdownDescription: "Reference to the building block runner that executes GitLab pipelines.",
													Computed:            true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															MarkdownDescription: "UUID of the meshBuildingBlockRunner.",
															Computed:            true,
														},
														"kind": schema.StringAttribute{
															MarkdownDescription: "meshObject type, always meshBuildingBlockRunner.",
															Computed:            true,
														},
													},
												},
											},
										},
										"azuredevops": schema.SingleNestedAttribute{
											MarkdownDescription: "Azure DevOps integration configuration.",
											Computed:            true,
											Optional:            true,
											Attributes: map[string]schema.Attribute{
												"base_url": schema.StringAttribute{
													MarkdownDescription: "Base URL of the Azure DevOps instance (e.g., https://dev.azure.com).",
													Computed:            true,
												},
												"organization": schema.StringAttribute{
													MarkdownDescription: "Azure DevOps organization name.",
													Computed:            true,
												},
												"personal_access_token": secret.DatasourceSchema(secret.SchemaOptions{}),
												"runner_ref": schema.SingleNestedAttribute{
													MarkdownDescription: "Reference to the building block runner that executes Azure DevOps pipelines.",
													Computed:            true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															MarkdownDescription: "UUID of the meshBuildingBlockRunner.",
															Computed:            true,
														},
														"kind": schema.StringAttribute{
															MarkdownDescription: "meshObject type, always meshBuildingBlockRunner.",
															Computed:            true,
														},
													},
												},
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
								"is_built_in": schema.BoolAttribute{
									MarkdownDescription: "Indicates whether this is a built-in integration (replicator or metering).",
									Computed:            true,
								},
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
	integrationDtos, err := d.meshIntegrationClient.List(ctx)
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
	integrations, err := generic.ValueFrom(integrationDtos, secret.WithDatasourceConverter())
	if err != nil {
		resp.Diagnostics.AddError("Value conversion error", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integrations"), integrations)...)
}
