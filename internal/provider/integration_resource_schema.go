package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

func (r *integrationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	workloadIdentityFederationAttributes := map[string]schema.Attribute{
		"issuer": schema.StringAttribute{
			MarkdownDescription: "OIDC issuer URL for workload identity federation.",
			Computed:            true,
		},
		"subject": schema.StringAttribute{
			MarkdownDescription: "OIDC subject for workload identity federation.",
			Computed:            true,
		},
		"gcp": schema.SingleNestedAttribute{
			MarkdownDescription: "GCP-specific workload identity federation configuration.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"audience": schema.StringAttribute{
					MarkdownDescription: "Audience for GCP workload identity federation.",
					Computed:            true,
				},
			},
		},
		"aws": schema.SingleNestedAttribute{
			MarkdownDescription: "AWS-specific workload identity federation configuration.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"audience": schema.StringAttribute{
					MarkdownDescription: "Audience for AWS workload identity federation.",
					Computed:            true,
				},
				"thumbprint": schema.StringAttribute{
					MarkdownDescription: "Certificate thumbprint for AWS workload identity federation.",
					Computed:            true,
				},
			},
		},
		"azure": schema.SingleNestedAttribute{
			MarkdownDescription: "Azure-specific workload identity federation configuration.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"audience": schema.StringAttribute{
					MarkdownDescription: "Audience for Azure workload identity federation.",
					Computed:            true,
				},
			},
		},
	}

	allowSingleImplementation := objectvalidator.ConflictsWith(
		path.MatchRelative().AtParent().AtName("github"),
		path.MatchRelative().AtParent().AtName("gitlab"),
		path.MatchRelative().AtParent().AtName("azuredevops"),
	)
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a meshIntegration in meshStack. " +
			"Integrations configure external CI/CD systems (GitHub, GitLab, Azure DevOps) for building block execution. " +
			"Secrets in the integration configurations are encrypted and stored securely. When retrieving the integration, " +
			"these fields are returned with a hash value instead of the actual secret, enabling drift detection while maintaining security.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Metadata of the integration. Contains identifiers and ownership details.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "UUID of the integration.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this integration. The integration will be owned by the workspace specified here.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Specification of the integration. Contains configuration settings specific to the integration type.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the integration.",
						Required:            true,
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Configuration for the integration. Must specify exactly one of `github`, `gitlab`, or `azuredevops`.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"github": schema.SingleNestedAttribute{
								MarkdownDescription: "GitHub integration configuration.",
								Optional:            true,
								Validators:          []validator.Object{allowSingleImplementation},
								Attributes: map[string]schema.Attribute{
									"owner": schema.StringAttribute{
										MarkdownDescription: "GitHub organization or user that owns the repositories.",
										Required:            true,
									},
									"base_url": schema.StringAttribute{
										MarkdownDescription: "Base URL of the GitHub instance (e.g., `https://github.com` for GitHub.com or your GitHub Enterprise URL).",
										Required:            true,
									},
									"app_id": schema.StringAttribute{
										MarkdownDescription: "GitHub App ID for authentication.",
										Required:            true,
									},
									"app_private_key": secret.ResourceSchema(secret.ResourceSchemaOptions{
										MarkdownDescription: "Private key for the GitHub App.",
										Optional:            false,
									}),
									"runner_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the building block runner that executes GitHub workflows." +
											"If omitted, the pre-defined shared runner is used.",
										Optional:   true,
										Computed:   true,
										Attributes: meshUuidRefAttribute("meshBuildingBlockRunner"),
									},
								},
							},
							"gitlab": schema.SingleNestedAttribute{
								MarkdownDescription: "GitLab integration configuration.",
								Optional:            true,
								Validators:          []validator.Object{allowSingleImplementation},
								Attributes: map[string]schema.Attribute{
									"base_url": schema.StringAttribute{
										MarkdownDescription: "Base URL of the GitLab instance (e.g., `https://gitlab.com` or your self-hosted GitLab URL).",
										Required:            true,
									},
									"runner_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the building block runner that executes GitLab pipelines." +
											"If omitted, the pre-defined shared runner is used.",
										Optional:   true,
										Computed:   true,
										Attributes: meshUuidRefAttribute("meshBuildingBlockRunner"),
									},
								},
							},
							"azuredevops": schema.SingleNestedAttribute{
								MarkdownDescription: "Azure DevOps integration configuration.",
								Optional:            true,
								Validators:          []validator.Object{allowSingleImplementation},
								Attributes: map[string]schema.Attribute{
									"base_url": schema.StringAttribute{
										MarkdownDescription: "Base URL of the Azure DevOps instance (e.g., `https://dev.azure.com`).",
										Required:            true,
									},
									"organization": schema.StringAttribute{
										MarkdownDescription: "Azure DevOps organization name.",
										Required:            true,
									},
									"personal_access_token": secret.ResourceSchema(secret.ResourceSchemaOptions{
										MarkdownDescription: "Personal Access Token (PAT) for authentication.",
										Optional:            true,
									}),
									"runner_ref": schema.SingleNestedAttribute{
										MarkdownDescription: "Reference to the building block runner that executes Azure DevOps pipelines. " +
											"If omitted, the pre-defined shared runner is used.",
										Optional:   true,
										Computed:   true,
										Attributes: meshUuidRefAttribute("meshBuildingBlockRunner"),
									},
								},
							},
						},
					},
				},
			},
			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Status information of the integration. System-managed state computed by meshStack.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"is_built_in": schema.BoolAttribute{
						MarkdownDescription: "For integrations created by this resource, this flag is always `false`",
						Computed:            true,
					},
					"workload_identity_federation": schema.SingleNestedAttribute{
						MarkdownDescription: "Workload identity federation configuration for the integration.",
						Computed:            true,
						Attributes:          workloadIdentityFederationAttributes,
					},
				},
			},

			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Reference to integration, can be used in building block definitions.",
				Computed:            true,
				Attributes:          meshUuidRefOutputAttribute("meshIntegration"),
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}
