package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &platformDataSource{}
	_ datasource.DataSourceWithConfigure = &platformDataSource{}
)

// NewPlatformDataSource is a helper function to simplify the provider implementation.
func NewPlatformDataSource() datasource.DataSource {
	return &platformDataSource{}
}

// platformDataSource is the data source implementation.
type platformDataSource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the data source type name.
func (d *platformDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the data source.
func (d *platformDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

func (d *platformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack platform.",
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Platform UUID identifier.",
						Required:            true,
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Platform identifier.",
						Computed:            true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The identifier of the workspace that owns this meshPlatform.",
						Computed:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation timestamp of the platform (server-generated).",
						Computed:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the meshPlatform was deleted, null if not deleted.",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The human-readable display name of the meshPlatform.",
						Computed:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the meshPlatform.",
						Computed:            true,
					},
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "The web console URL endpoint of the platform.",
						Computed:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform support documentation.",
						Computed:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform documentation.",
						Computed:            true,
					},
					"location_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the location where this platform is situated.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "Must always be set to meshLocation",
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("meshLocation"),
								},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the Location.",
								Computed:            true,
							},
						},
					},
					"contributing_workspaces": schema.ListAttribute{
						MarkdownDescription: "A list of workspace identifiers that contribute to this meshPlatform.",
						ElementType:         types.StringType,
						Computed:            true,
					},
					"availability": schema.SingleNestedAttribute{
						MarkdownDescription: "Availability configuration for the meshPlatform.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"restriction": schema.StringAttribute{
								MarkdownDescription: "Access restriction for the platform. Must be one of: PUBLIC, PRIVATE, RESTRICTED.",
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLIC", "PRIVATE", "RESTRICTED"),
								},
							},
							"publication_state": schema.StringAttribute{
								MarkdownDescription: "Publication state of the platform. Must be one of: PUBLISHED, UNPUBLISHED.",
								Computed:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLISHED", "UNPUBLISHED"),
								},
							},
							"restricted_to_workspaces": schema.ListAttribute{
								MarkdownDescription: "If the restriction is set to `RESTRICTED`, you can specify the workspace identifiers this meshPlatform is restricted to.",
								ElementType:         types.StringType,
								Computed:            true,
							},
						},
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration options.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformDataSourceSchema(),
							"aks":        aksPlatformDataSourceSchema(),
							"azure":      azurePlatformDataSourceSchema(),
							"azurerg":    azureRgPlatformDataSourceSchema(),
							"gcp":        gcpPlatformDataSourceSchema(),
							"kubernetes": kubernetesPlatformDataSourceSchema(),
							"openshift":  openShiftPlatformDataSourceSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform.",
								Computed:            true,
							},
						},
					},
				},
			},
		},
	}
}

func awsPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Configuration for AWS",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region",
				Computed:            true,
			},
			"replication": awsReplicationConfigDataSourceSchema(),
		},
	}
}

func aksPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Kubernetes Service configuration",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the AKS cluster",
				Computed:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the AKS cluster. (SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.)",
				Computed:            true,
			},
			"replication": aksReplicationConfigDataSourceSchema(),
		},
	}
}

func azurePlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform configuration.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Computed:            true,
			},
			"replication": azureReplicationConfigDataSourceSchema(),
		},
	}
}

func azureRgPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group platform configuration.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Computed:            true,
			},
			"replication": azureRgReplicationConfigDataSourceSchema(),
		},
	}
}

func gcpPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Google Cloud Platform (GCP) platform configuration.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"replication": gcpReplicationConfigDataSourceSchema(),
		},
	}
}

func kubernetesPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Kubernetes platform configuration.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "This URL is the base URL to your Kubernetes Cluster, which is used to call the APIs to create new Kubernetes projects, get raw data for metering the Kubernetes projects, etc. An example base URL is: https://k8s.dev.eu-de-central.msh.host:6443",
				Computed:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the Kubernetes cluster. SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.",
				Computed:            true,
			},
			"replication": kubernetesReplicationConfigDataSourceSchema(),
		},
	}
}

func openShiftPlatformDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "OpenShift platform configuration.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "This URL is the base URL to your OpenShift Cluster, which is used to call the APIs to create new OpenShift projects, get raw data for metering the OpenShift projects, etc. An example base URL is: https://api.okd4.dev.eu-de-central.msh.host:6443",
				Computed:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the OpenShift cluster. SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.",
				Computed:            true,
			},
			"replication": openShiftReplicationConfigDataSourceSchema(),
		},
	}
}

// TODO review done until here

func aksReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AKS (optional, but required for replication)",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "The Access Token of the service account for replicator access.",
				Computed:            true,
			},
			"namespace_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming namespaces in AKS",
				Computed:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming groups in AKS",
				Computed:            true,
			},
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for AKS",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID'.",
						Computed:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type for the service principal (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Computed:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret for the service principal (if `authType` is `CREDENTIALS`)",
						Computed:            true,
					},
					"entra_tenant": schema.StringAttribute{
						MarkdownDescription: "Domain name or ID of the Entra Tenant that holds the Service Principal.",
						Computed:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Computed:            true,
					},
				},
			},
			"aks_subscription_id": schema.StringAttribute{
				MarkdownDescription: "Subscription ID for the AKS cluster",
				Computed:            true,
			},
			"aks_cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the AKS cluster",
				Computed:            true,
			},
			"aks_resource_group": schema.StringAttribute{
				MarkdownDescription: "Resource group for the AKS cluster",
				Computed:            true,
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "This is the URL that Azure’s consent experience redirects users to after they accept their invitation.",
				Computed:            true,
			},
			"send_azure_invitation_mail": schema.BoolAttribute{
				MarkdownDescription: "Flag to send Azure invitation emails. When true, meshStack instructs Azure to send out Invitation mails to invited users.",
				Computed:            true,
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "Strategy for user lookup in Azure (`userPrincipalName` or `email`)",
				Computed:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Computed:            true,
			},
		},
	}
}

func awsReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AWS (optional, but required for replication)",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"access_config": schema.SingleNestedAttribute{
				MarkdownDescription: "meshStack currently supports 2 types of authentication. Workload Identity Federation (using OIDC) is the one that we recommend as it enables secure access to your AWS account without using long lived credentials. Alternatively, you can use credential based authentication by providing access and secret keys. Either the `service_user_config` or `workload_identity_config` must be provided.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"organization_root_account_role": schema.StringAttribute{
						MarkdownDescription: "ARN of the Management Account Role. The Management Account contains your AWS organization. E.g. arn:aws:iam::123456789:role/MeshfedServiceRole.",
						Computed:            true,
					},
					"organization_root_account_external_id": schema.StringAttribute{
						MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the organization root account role.",
						Computed:            true,
					},
					"service_user_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service user configuration (alternative to `workload_identity_config`)",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"access_key": schema.StringAttribute{
								MarkdownDescription: "AWS access key for service user",
								Computed:            true,
							},
							"secret_key": schema.StringAttribute{
								MarkdownDescription: "AWS secret key for service user",
								Computed:            true,
							},
						},
					},
					"workload_identity_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Workload identity configuration (alternative to `service_user_config`)",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"role_arn": schema.StringAttribute{
								MarkdownDescription: "ARN of the role that should be used as the entry point for meshStack by assuming it via web identity.",
								Computed:            true,
							},
						},
					},
				},
			},
			"wait_for_external_avm": schema.BoolAttribute{
				MarkdownDescription: "Flag to wait for external AVM. Please use this setting with care! It is currently very specific to certain tags being present on the account! In general, we recommend not to activate this functionality! In a meshLandingZone an AVM can be triggered via an AWS StackSet or via a Lambda Function. If meshStack shall wait for the AVM to complete when creating a new platform tenant, this flag must be checked. meshStack will identify completion of the AVM by checking the presence of the following tags on the AWS account: 'ProductName' is set to workspace identifier and 'Stage' is set to project identifier.",
				Computed:            true,
			},
			"automation_account_role": schema.StringAttribute{
				MarkdownDescription: "ARN of the Automation Account Role. The Automation Account contains all AWS StackSets and Lambda Functions that shall be executed via meshLandingZones. E.g. `arn:aws:iam::123456789:role/MeshfedAutomationRole`.",
				Computed:            true,
			},
			"automation_account_external_id": schema.StringAttribute{
				MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the automation account role.",
				Computed:            true,
			},
			"account_access_role": schema.StringAttribute{
				MarkdownDescription: "The name for the Account Access Role that will be rolled out to all managed accounts. Only a name, not an ARN must be set here, as the ARN must be built dynamically for every managed AWS Account. The replicator service user needs to assume this role in all accounts to manage them.",
				Computed:            true,
			},
			"account_alias_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account alias of the created AWS account will be named. E.g. `#{workspaceIdentifier}-#{projectIdentifier}`. Attention: Account Alias must be globally unique in AWS. So consider defining a unique prefix.",
				Computed:            true,
			},
			"enforce_account_alias": schema.BoolAttribute{
				MarkdownDescription: "Flag to enforce account alias. If set, meshStack will guarantee on every replication that the configured Account Alias is applied. Otherwise it will only set the Account Alias once during tenant creation.",
				Computed:            true,
			},
			"account_email_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account email address of the created AWS account will be set. E.g. `aws+#{workspaceIdentifier}.#{projectIdentifier}@yourcompany.com`. Please consider that this email address is limited to 64 characters! Also have a look at our docs for more information.",
				Computed:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "Namespace prefix for tenant tags",
						Computed:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Computed:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"aws_sso": schema.SingleNestedAttribute{
				MarkdownDescription: "AWS SSO configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"scim_endpoint": schema.StringAttribute{
						MarkdownDescription: "The SCIM endpoint you can find in your AWS IAM Identity Center Automatic provisioning config.",
						Computed:            true,
					},
					"arn": schema.StringAttribute{
						MarkdownDescription: "The ARN of your AWS IAM Identity Center Instance. E.g. `arn:aws:sso:::instance/ssoins-123456789abc`.",
						Computed:            true,
					},
					"group_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Configures the pattern that defines the desired name of AWS IAM Identity Center groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names will be unique within the same AWS IAM Identity Center Instance with that configuration. meshStack will additionally prefix the group name with 'mst-' to be able to identify the groups that are managed by meshStack.",
						Computed:            true,
					},
					"sso_access_token": schema.StringAttribute{
						MarkdownDescription: "The AWS IAM Identity Center SCIM Access Token that was generated via the Automatic provisioning config in AWS IAM Identity Center.",
						Computed:            true,
					},
					"aws_role_mappings": schema.ListNestedAttribute{
						MarkdownDescription: "AWS role mappings for AWS SSO",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"project_role_ref": meshProjectRoleAttribute(),
								"aws_role": schema.StringAttribute{
									MarkdownDescription: "The AWS role name",
									Computed:            true,
								},
								"permission_set_arns": schema.ListAttribute{
									MarkdownDescription: "List of permission set ARNs associated with this role mapping",
									ElementType:         types.StringType,
									Optional:            true,
								},
							},
						},
					},
					"sign_in_url": schema.StringAttribute{
						MarkdownDescription: "The AWS IAM Identity Center sign in Url, that must be used by end-users to log in via AWS IAM Identity Center to AWS Management Console.",
						Computed:            true,
					},
				},
			},
			"enrollment_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "AWS account enrollment configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"management_account_id": schema.StringAttribute{
						MarkdownDescription: "The Account ID of the management account configured for the platform instance.",
						Computed:            true,
					},
					"account_factory_product_id": schema.StringAttribute{
						MarkdownDescription: "The Product ID of the AWS Account Factory Product in AWS Service Catalog that should be used for enrollment. Starts with `prod-`.",
						Computed:            true,
					},
				},
			},
			"self_downgrade_access_role": schema.BoolAttribute{
				MarkdownDescription: "Flag for self downgrade access role. If set, meshStack will revoke its rights on the managed account that were only needed for initial account creation.",
				Computed:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the AWS platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Computed:            true,
			},
			"allow_hierarchical_organizational_unit_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical organizational unit assignment in AWS. If set to true: Accounts can be moved to child organizational units of the organizational unit defined in the Landing Zone. This is useful if you want to manage the account location with a deeper and more granular hierarchy. If set to false: Accounts will always be moved directly to the organizational unit defined in the Landing Zone.",
				Computed:            true,
			},
		},
	}
}

func azureReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure-specific replication configuration for the platform.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for Azure",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Computed:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Computed:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret (if authType is `CREDENTIALS`)",
						Computed:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Computed:            true,
					},
				},
			},
			"provisioning": schema.SingleNestedAttribute{
				MarkdownDescription: "To provide Azure Subscription for your organization's meshProjects, meshcloud supports using Enterprise Enrollment or allocating from a pool of pre-provisioned subscriptions. One of the subFields enterpriseEnrollment, customerAgreement or preProvisioned must be provided!",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"subscription_owner_object_ids": schema.ListAttribute{
						MarkdownDescription: "One or more principals Object IDs (e.g. user groups, SPNs) that meshStack will ensure have an 'Owner' role assignment on the managed subscriptions. This can be useful to satisfy Azure's constraint of at least one direct 'Owner' role assignment per Subscription. If you want to use a Service Principal please use the Enterprise Application Object ID. You can not use the replicator object ID here, because meshStack always removes its high privilege access after a Subscription creation.",
						Computed:            true,
						ElementType:         types.StringType,
					},
					"enterprise_enrollment": schema.SingleNestedAttribute{
						MarkdownDescription: "meshcloud can automatically provision new subscriptions from an Enterprise Enrollment Account owned by your organization. This is suitable for large organizations that have a Microsoft Enterprise Agreement, Microsoft Customer Agreement or a Microsoft Partner Agreement and want to provide a large number of subscriptions in a fully automated fashion.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"enrollment_account_id": schema.StringAttribute{
								MarkdownDescription: "ID of the EA Enrollment Account used for the Subscription creation. Should look like this: `/providers/Microsoft.Billing/billingAccounts/1234567/enrollmentAccounts/7654321`. For more information, review the [Azure docs](https://docs.microsoft.com/en-us/azure/cost-management-billing/manage/programmatically-create-subscription-enterprise-agreement?tabs=rest-getEnrollments%2Crest-EA#find-accounts-you-have-access-to).",
								Computed:            true,
							},
							"subscription_offer_type": schema.StringAttribute{
								MarkdownDescription: "The Microsoft Subscription offer type to use when creating subscriptions. Only Production for standard and DevTest for Dev/Test subscriptions are supported for the Non Legacy Subscription Enrollment. For the Legacy Subscription Enrollment also other types can be defined.",
								Computed:            true,
							},
							"use_legacy_subscription_enrollment": schema.BoolAttribute{
								MarkdownDescription: "Deprecated: Uses the old Subscription enrollment API in its preview version. This enrollment is less reliable and should not be used for new Azure Platform Integrations.",
								Computed:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure's MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Computed:            true,
							},
						},
					},
					"customer_agreement": schema.SingleNestedAttribute{
						MarkdownDescription: "meshcloud can automatically provision new subscriptions from a Customer Agreement Account owned by your organization. This is suitable for larger organizations that have such a Customer Agreement with Microsoft, and want to provide a large number of subscriptions in a fully automated fashion.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"source_service_principal": schema.SingleNestedAttribute{
								MarkdownDescription: "Configure the SPN used by meshStack to create a new Subscription in your MCA billing scope. For more information on the required permissions, see the [Azure docs](https://learn.microsoft.com/en-us/azure/cost-management-billing/manage/programmatically-create-subscription-microsoft-customer-agreement-across-tenants).",
								Computed:            true,
								Attributes: map[string]schema.Attribute{
									"client_id": schema.StringAttribute{
										MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the \"Enterprise Application\" but can also be retrieved via the \"App Registration\" object as \"Application (Client) ID\".",
										Computed:            true,
									},
									"auth_type": schema.StringAttribute{
										MarkdownDescription: "Must be one of `CREDENTIALS` or `WORKLOAD_IDENTITY`. Workload Identity Federation is the one that we recommend as it enables the most secure approach to provide access to your Azure tenant without using long lived credentials. Credential Authentication is an alternative approach where you have to provide a clientSecret manually to meshStack and meshStack stores it encrypted.",
										Computed:            true,
									},
									"credentials_auth_client_secret": schema.StringAttribute{
										MarkdownDescription: "Must be set if and only if authType is CREDENTIALS. A valid secret for accessing the application. In Azure Portal, this can be configured on the \"App Registration\" under Certificates & secrets. [How is this information secured?](https://docs.meshcloud.io/operations/security-faq/#how-does-meshstack-securely-handle-my-cloud-platform-credentials)",
										Computed:            true,
									},
								},
							},
							"destination_entra_id": schema.StringAttribute{
								MarkdownDescription: "Microsoft Entra ID Tenant UUID where created subscriptions should be moved. Set this to the Microsoft Entra ID Tenant hosting your landing zones.",
								Computed:            true,
							},
							"source_entra_tenant": schema.StringAttribute{
								MarkdownDescription: "Microsoft Entra ID Tenant UUID or domain name used for creating subscriptions. Set this to the Microsoft Entra ID Tenant owning the MCA Billing Scope. If source and destination Microsoft Entra ID Tenants are the same, you need to use UUID.",
								Computed:            true,
							},
							"billing_scope": schema.StringAttribute{
								MarkdownDescription: "ID of the MCA Billing Scope used for creating subscriptions. Must follow this format: `/providers/Microsoft.Billing/billingAccounts/$accountId/billingProfiles/$profileId/invoiceSections/$sectionId`.",
								Computed:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure's MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Computed:            true,
							},
						},
					},
					"pre_provisioned": schema.SingleNestedAttribute{
						MarkdownDescription: "If your organization does not have access to an Enterprise Enrollment, you can alternatively configure meshcloud to consume subscriptions from a pool of externally-provisioned subscriptions. This is useful for smaller organizations that wish to use 'Pay-as-you-go' subscriptions or if you're organization partners with an Azure Cloud Solution Provider to provide your subscriptions. The meshcloud Azure replication detects externally-provisioned subscriptions based on a configurable prefix in the subscription name. Upon assignment to a meshProject, the subscription is inflated with the right Landing Zone configuration and removed from the subscription pool.",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"unused_subscription_name_prefix": schema.StringAttribute{
								MarkdownDescription: "The prefix that identifies unused subscriptions. Subscriptions will be renamed during meshStack's project replication, at which point they should no longer carry this prefix.",
								Computed:            true,
							},
						},
					},
				},
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional B2B user invitation configuration. When configured, instructs the replicator to create AAD B2B guest invitations for users missing in the AAD tenant managed by this meshPlatform.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Computed:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Computed:            true,
					},
				},
			},
			"subscription_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of Azure Subscriptions managed by meshStack.",
				Computed:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names are unique in the managed AAD Tenant.",
				Computed:            true,
			},
			"blueprint_service_principal": schema.StringAttribute{
				MarkdownDescription: " \t\n\nObject ID of the Enterprise Application belonging to the Microsoft Application 'Azure Blueprints'. meshStack will grant the necessary permissions on managed Subscriptions to this SPN so that it can create System Assigned Managed Identities (SAMI) for Blueprint execution.",
				Computed:            true,
			},
			"blueprint_location": schema.StringAttribute{
				MarkdownDescription: "The Azure location where replication creates and updates Blueprint Assignments. Note that it's still possible that the Blueprint creates resources in other locations, this is merely the location where the Blueprint Assignment is managed.",
				Computed:            true,
			},
			"azure_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Azure role mappings for Azure role definitions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"azure_role": schema.SingleNestedAttribute{
							MarkdownDescription: "The Azure role definition.",
							Computed:            true,
							Attributes: map[string]schema.Attribute{
								"alias": schema.StringAttribute{
									MarkdownDescription: "The alias/name of the Azure role.",
									Computed:            true,
								},
								"id": schema.StringAttribute{
									MarkdownDescription: "The Azure role definition ID.",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tagging configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
						Computed:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Computed:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy (`userPrincipalName` or `email`). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Computed:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Computed:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Computed:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to sub management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Computed:            true,
			},
		},
	}
}

// TODO continue here.

func azureRgReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group-specific replication configuration for the platform.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for Azure Resource Group access.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Computed:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Computed:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret (if authType is `CREDENTIALS`)",
						Computed:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Computed:            true,
					},
				},
			},
			"subscription": schema.StringAttribute{
				MarkdownDescription: "The Subscription that will contain all the created Resource Groups. Once you set the Subscription, you must not change it.",
				Computed:            true,
			},
			"resource_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name Resource Group managed by meshStack. It follows the usual replicator string pattern features. Operators must ensure the group names are unique within the Subscription.",
				Computed:            true,
			},
			"user_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix. This suffix is configurable via Role Mappings in this platform config.",
				Computed:            true,
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional B2B user invitation configuration. When configured, instructs the replicator to create AAD B2B guest invitations for users missing in the AAD tenant managed by this meshPlatform.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure’s consent experience redirects users to after they accept their invitation.",
						Computed:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Computed:            true,
					},
				},
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy (`userPrincipalName` or `email`). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Computed:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix for tag namespaces.",
						Computed:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Computed:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Computed:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Computed:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to child management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Computed:            true,
			},
		},
	}
}

func gcpReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP-specific replication configuration for the platform.",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"service_account_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Service account configuration. Either `serviceAccountCredentialsConfig` or `serviceAccountWorkloadIdentityConfig` must be provided.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"service_account_credentials_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service account credentials configuration (alternative to serviceAccountWorkloadIdentityConfig)",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"service_account_credentials_b64": schema.StringAttribute{
								MarkdownDescription: "Base64 encoded credentials.json file for a GCP ServiceAccount. The replicator uses this Service Account to automate GCP API operations (IAM, ResourceManager etc.).",
								Computed:            true,
								Sensitive:           true,
							},
						},
					},
					"service_account_workload_identity_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service account workload identity configuration (alternative to serviceAccountCredentialsConfig)",
						Computed:            true,
						Attributes: map[string]schema.Attribute{
							"audience": schema.StringAttribute{
								MarkdownDescription: "The audience associated with your workload identity pool provider.",
								Computed:            true,
							},
							"service_account_email": schema.StringAttribute{
								MarkdownDescription: "The email address of the Service Account, that gets impersonated for calling Google APIs via Workload Identity Federation.",
								Computed:            true,
							},
						},
					},
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain used for cloud identity directory-groups created and managed by meshStack. meshStack maintains separate groups for each meshProject role on each managed GCP project.",
				Computed:            true,
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "A Google Customer ID. It typically starts with a 'C'.",
				Computed:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. Additionally you can also use 'platformGroupAlias' as a placeholder to access the specific project role from the role mappings done in this platform configuration or in the meshLandingZone configuration.",
				Computed:            true,
			},
			"project_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. The result must be 4 to 30 characters. Allowed characters are: lowercase and uppercase letters, numbers, hyphen, single-quote, double-quote, space, and exclamation point. When length restrictions are applied, the abbreviation will be in the middle and marked by a single-quote.",
				Computed:            true,
			},
			"project_id_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. The resulting string must not exceed a total length of 30 characters. Only alphanumeric + hyphen are allowed. We recommend that configuration include at least 3 characters of the random parameter to reduce the chance of naming collisions as the project Ids must be globally unique within GCP.",
				Computed:            true,
			},
			"billing_account_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the billing account to associate with all GCP projects managed by meshStack",
				Computed:            true,
			},
			"user_lookup_strategy": schema.StringAttribute{
				MarkdownDescription: "Users can either be looked up by E-Mail or externalAccountId. This must also be the property that is placed in the external user id (EUID) of your meshUser entity to match. E-Mail is usually a good choice as this is often set up as the EUID throughout all cloud platforms and meshStack. ('email' or 'externalId')",
				Computed:            true,
			},
			"gcp_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Mapping of platform roles to GCP IAM roles.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"gcp_role": schema.StringAttribute{
							MarkdownDescription: "The GCP IAM role",
							Computed:            true,
						},
					},
				},
			},
			"allow_hierarchical_folder_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical folder assignment in GCP. If set to true: Projects can be moved to sub folders of the folder defined in the Landing Zone. This is useful if you want to manage the project location with a deeper and more granular hierarchy. If set to false: Projects will always be moved directly to the folder defined in the Landing Zone.",
				Computed:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "Prefix for tag namespaces.",
						Computed:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for generating tags.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Computed:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Computed:            true,
								},
							},
						},
					},
				},
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the GCP platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Computed:            true,
			},
		},
	}
}

func kubernetesReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for Kubernetes (optional, but required for replication)",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Client configuration for Kubernetes",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"access_token": schema.StringAttribute{
						MarkdownDescription: "The Access Token of the service account for replicator access.",
						Computed:            true,
						Sensitive:           true,
					},
				},
			},
			"namespace_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. Kubernetes Namespace Names must be no longer than 63 characters, must start and end with a lowercase letter or number, and may contain lowercase letters, numbers, and hyphens.",
				Computed:            true,
			},
		},
	}
}

// TODO continue here.

func openShiftReplicationConfigDataSourceSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for OpenShift (optional, but required for replication)",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Client configuration for OpenShift",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"access_token": schema.StringAttribute{
						MarkdownDescription: "The Access Token of the service account for replicator access.",
						Computed:            true,
						Sensitive:           true,
					},
				},
			},
			"web_console_url": schema.StringAttribute{
				MarkdownDescription: "The Web Console URL that is used to redirect the user to the cloud platform. An example Web Console URL is https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host",
				Computed:            true,
			},
			"project_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. OpenShift Project Names must be no longer than 63 characters, must start and end with a lowercase letter or number, and may contain lowercase letters, numbers, and hyphens.",
				Computed:            true,
			},
			"enable_template_instantiation": schema.BoolAttribute{
				MarkdownDescription: "Here you can enable templates not only being rolled out to OpenShift but also instantiated during replication. Templates can be configured in meshLandingZones. Please keep in mind that the replication service account needs all the rights that are required to apply the templates that are configured in meshLandingZones.",
				Computed:            true,
			},
			"openshift_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "OpenShift role mappings for OpenShift roles.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"openshift_role": schema.StringAttribute{
							MarkdownDescription: "The OpenShift role name",
							Computed:            true,
						},
					},
				},
			},
			"identity_provider_name": schema.StringAttribute{
				MarkdownDescription: "Identity provider name",
				Computed:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tagging configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
						Computed:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Computed:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Computed:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *platformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var uuid string

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if resp.Diagnostics.HasError() {
		return
	}

	platform, err := d.client.ReadPlatform(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform with UUID '%s'", uuid),
			err.Error(),
		)
		return
	}

	if platform == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Platform with UUID '%s' not found", uuid),
			"The platform does not exist.",
		)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, platform)...)
}
