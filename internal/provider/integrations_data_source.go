package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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
		MarkdownDescription: "List of integrations.",

		Attributes: map[string]schema.Attribute{
			"workload_identity_federation": schema.SingleNestedAttribute{
				MarkdownDescription: "Workload identity federation information for built in integrations.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"replicator": workloadIdentityFederation,
					"metering":   workloadIdentityFederation,
				},
			},
			"integrations": schema.ListNestedAttribute{
				MarkdownDescription: "List of integrations",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"metadata": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"uuid": schema.StringAttribute{
									Computed: true,
								},
								"owned_by_workspace": schema.StringAttribute{
									Computed: true,
								},
							},
						},
						"spec": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"display_name": schema.StringAttribute{
									Computed: true,
								},
								"config": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"github": schema.SingleNestedAttribute{
											Computed: true,
											Optional: true,
											Attributes: map[string]schema.Attribute{
												"owner": schema.StringAttribute{
													Computed: true,
												},
												"base_url": schema.StringAttribute{
													Computed: true,
												},
												"app_id": schema.StringAttribute{
													Computed: true,
												},
												"app_private_key": schema.StringAttribute{
													Computed: true,
												},
												"runner_ref": schema.SingleNestedAttribute{
													Computed: true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															Computed: true,
														},
														"kind": schema.StringAttribute{
															Computed: true,
														},
													},
												},
											},
										},
										"gitlab": schema.SingleNestedAttribute{
											Computed: true,
											Optional: true,
											Attributes: map[string]schema.Attribute{
												"base_url": schema.StringAttribute{
													Computed: true,
												},
												"runner_ref": schema.SingleNestedAttribute{
													Computed: true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															Computed: true,
														},
														"kind": schema.StringAttribute{
															Computed: true,
														},
													},
												},
											},
										},
										"azuredevops": schema.SingleNestedAttribute{
											Computed: true,
											Optional: true,
											Attributes: map[string]schema.Attribute{
												"base_url": schema.StringAttribute{
													Computed: true,
												},
												"organization": schema.StringAttribute{
													Computed: true,
												},
												"personal_access_token": schema.StringAttribute{
													Computed: true,
												},
												"runner_ref": schema.SingleNestedAttribute{
													Computed: true,
													Attributes: map[string]schema.Attribute{
														"uuid": schema.StringAttribute{
															Computed: true,
														},
														"kind": schema.StringAttribute{
															Computed: true,
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
							Computed: true,
							Optional: true,
							Attributes: map[string]schema.Attribute{
								"is_built_in": schema.BoolAttribute{
									Computed: true,
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
	var integrations []integration
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
		var state integration
		state.SetFromClientDto(&integrationDto, &resp.Diagnostics)
		integrations = append(integrations, state)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("integrations"), integrations)...)
}
