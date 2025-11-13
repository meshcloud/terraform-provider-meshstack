package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/modifiers/platformtypemodifier"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &platformResource{}
	_ resource.ResourceWithConfigure   = &platformResource{}
	_ resource.ResourceWithImportState = &platformResource{}
)

// NewPlatformResource is a helper function to simplify the provider implementation.
func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

// platformResource is the resource implementation.
type platformResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the resource.
func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	markdownDescription := "Represents a meshStack platform.\n\n" +
		"Please note that the meshPlatform API endpoints are still in preview state and therefore the following limitations apply:\n" +
		"* Deleting and re-creating a platform with the same identifier is not possible. Once you have used a platform identifier, you cannot use it again, even if the platform has been deleted. You may run into this issue when you attempt to modify an immutable attribute and terraform therefore attempts to replace (i.e., delete and recreate) the entire platform, which will result in an error with a status code of `409` due to the identifier already being used by a deleted platform.\n" +
		"* Changing the owning workspace of a platform (`metadata.owned_by_workspace`) is not possible. To transfer the ownership of a platform, you must use meshPanel."

	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescription,
		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2-preview"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Default:             stringdefault.StaticString("meshPlatform"),
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"uuid": schema.StringAttribute{
						MarkdownDescription: "Unique identifier of the platform (server-generated).",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						MarkdownDescription: "Make sure you use a unique platform identifier within a Location. Location + Platform identifiers are being used to uniquely identify a platform in meshStack. You cannot change this identifier after creation of a platform.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`),
								"must be alphanumeric with dashes, must be lowercase, and have no leading, trailing or consecutive dashes",
							),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "The identifier of the workspace that owns this meshPlatform.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation timestamp of the platform (server-generated).",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Timestamp when the meshPlatform was deleted, null if not deleted.",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "The human-readable display name of the meshPlatform.",
						Required:            true,
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of the meshPlatform.",
						Required:            true,
					},
					"endpoint": schema.StringAttribute{
						MarkdownDescription: "The web console URL endpoint of the platform.",
						Required:            true,
					},
					"support_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform support documentation.",
						Optional:            true,
					},
					"documentation_url": schema.StringAttribute{
						MarkdownDescription: "URL for platform documentation.",
						Optional:            true,
					},
					"location_ref": schema.SingleNestedAttribute{
						MarkdownDescription: "Reference to the location where this platform is situated.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"kind": schema.StringAttribute{
								MarkdownDescription: "meshObject type, always `meshLocation`.",
								Computed:            true,
								Default:             stringdefault.StaticString("meshLocation"),
								Validators: []validator.String{
									stringvalidator.OneOf([]string{"meshLocation"}...),
								},
								PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
							},
							"name": schema.StringAttribute{
								MarkdownDescription: "Identifier of the Location.",
								Required:            true,
							},
						},
					},
					"contributing_workspaces": schema.ListAttribute{
						MarkdownDescription: "A list of workspace identifiers that may contribute to this meshPlatform.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
					},
					"availability": schema.SingleNestedAttribute{
						MarkdownDescription: "Availability configuration for the meshPlatform.",
						Required:            true,
						Attributes: map[string]schema.Attribute{
							"restriction": schema.StringAttribute{
								MarkdownDescription: "Access restriction for the platform. Must be one of: `PUBLIC`, `PRIVATE`, `RESTRICTED`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLIC", "PRIVATE", "RESTRICTED"),
								},
							},
							"publication_state": schema.StringAttribute{
								MarkdownDescription: "Marketplace publication state of the platform. Must be one of: `PUBLISHED`, `UNPUBLISHED`.",
								Required:            true,
								Validators: []validator.String{
									stringvalidator.OneOf("PUBLISHED", "UNPUBLISHED"),
								},
							},
							"restricted_to_workspaces": schema.ListAttribute{
								MarkdownDescription: "If the restriction is set to `RESTRICTED`, you can specify the workspace identifiers this meshPlatform is restricted to.",
								ElementType:         types.StringType,
								Optional:            true,
								Computed:            true,
								Default:             listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
							},
						},
					},
					"quota_definitions": schema.ListAttribute{
						MarkdownDescription: "List of quota definitions for the platform.",
						Required:            true,
						Sensitive:           false,
						ElementType: types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"quota_key":               types.StringType,
								"label":                   types.StringType,
								"description":             types.StringType,
								"unit":                    types.StringType,
								"min_value":               types.Int64Type,
								"max_value":               types.Int64Type,
								"auto_approval_threshold": types.Int64Type,
							},
						},
					},
					"config": schema.SingleNestedAttribute{
						MarkdownDescription: "Platform-specific configuration settings.",
						Required:            true,
						Sensitive:           false,
						PlanModifiers: []planmodifier.Object{
							platformtypemodifier.ValidateSinglePlatform(),
						},
						Attributes: map[string]schema.Attribute{
							"aws":        awsPlatformSchema(),
							"aks":        aksPlatformSchema(),
							"azure":      azurePlatformSchema(),
							"azurerg":    azureRgPlatformSchema(),
							"gcp":        gcpPlatformSchema(),
							"kubernetes": kubernetesPlatformSchema(),
							"openshift":  openShiftPlatformSchema(),
							"type": schema.StringAttribute{
								MarkdownDescription: "Type of the platform. This field is automatically inferred from which platform configuration is provided and cannot be set manually.",
								Computed:            true,
								PlanModifiers: []planmodifier.String{
									stringplanmodifier.RequiresReplace(),
									platformtypemodifier.SetTypeFromPlatform(),
								},
							},
						},
					},
				},
			},
		},
	}
}

func awsPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Configuration for AWS",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region",
				Optional:            true,
			},
			"replication": awsReplicationConfigSchema(),
		},
	}
}

func aksPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Kubernetes Service configuration",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL of the AKS cluster",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the AKS cluster. (SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"replication": aksReplicationConfigSchema(),
			"metering":    aksMeteringConfigSchema(),
		},
	}
}

func azurePlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Optional:            true,
			},
			"replication": azureReplicationConfigSchema(),
		},
	}
}

func azureRgPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Optional:            true,
			},
			"replication": azureRgReplicationConfigSchema(),
		},
	}
}

func gcpPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Google Cloud Platform (GCP) platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"replication": gcpReplicationConfigSchema(),
		},
	}
}

func kubernetesPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Kubernetes platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "This URL is the base URL to your Kubernetes Cluster, which is used to call the APIs to create new Kubernetes projects, get raw data for metering the Kubernetes projects, etc. An example base URL is: https://k8s.dev.eu-de-central.msh.host:6443",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the Kubernetes cluster. SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"replication": kubernetesReplicationConfigSchema(),
			"metering":    kubernetesMeteringConfigSchema(),
		},
	}
}

func openShiftPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "OpenShift platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				MarkdownDescription: "This URL is the base URL to your OpenShift Cluster, which is used to call the APIs to create new OpenShift projects, get raw data for metering the OpenShift projects, etc. An example base URL is: https://api.okd4.dev.eu-de-central.msh.host:6443",
				Required:            true,
			},
			"disable_ssl_validation": schema.BoolAttribute{
				MarkdownDescription: "Flag to disable SSL validation for the OpenShift cluster. SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"replication": openShiftReplicationConfigSchema(),
			"metering":    openShiftMeteringConfigSchema(),
		},
	}
}

func aksReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AKS (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "The Access Token of the service account for replicator access.",
				Optional:            true,
				Sensitive:           true,
			},
			"namespace_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming namespaces in AKS",
				Optional:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming groups in AKS",
				Optional:            true,
			},
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for AKS",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID'.",
						Required:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type for the service principal (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Required:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret for the service principal (if `authType` is `CREDENTIALS`)",
						Optional:            true,
						Sensitive:           true,
					},
					"entra_tenant": schema.StringAttribute{
						MarkdownDescription: "Domain name or ID of the Entra Tenant that holds the Service Principal.",
						Required:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
				},
			},
			"aks_subscription_id": schema.StringAttribute{
				MarkdownDescription: "Subscription ID for the AKS cluster",
				Optional:            true,
			},
			"aks_cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the AKS cluster.",
				Optional:            true,
			},
			"aks_resource_group": schema.StringAttribute{
				MarkdownDescription: "Resource group for the AKS cluster",
				Optional:            true,
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "This is the URL that Azure’s consent experience redirects users to after they accept their invitation.",
				Optional:            true,
			},
			"send_azure_invitation_mail": schema.BoolAttribute{
				MarkdownDescription: "Flag to send Azure invitation emails. When true, meshStack instructs Azure to send out Invitation mails to invited users.",
				Optional:            true,
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "Strategy for user lookup in Azure (`userPrincipalName` or `email`)",
				Optional:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
		},
	}
}

func aksMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for AKS (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for AKS metering"),
			"processing":    meteringProcessingConfigSchema(),
		},
	}
}

func awsReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AWS (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"access_config": schema.SingleNestedAttribute{
				MarkdownDescription: "meshStack currently supports 2 types of authentication. Workload Identity Federation (using OIDC) is the one that we recommend as it enables secure access to your AWS account without using long lived credentials. Alternatively, you can use credential based authentication by providing access and secret keys. Either the `service_user_config` or `workload_identity_config` must be provided.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"organization_root_account_role": schema.StringAttribute{
						MarkdownDescription: "ARN of the Management Account Role. The Management Account contains your AWS organization. E.g. `arn:aws:iam::123456789:role/MeshfedServiceRole`.",
						Required:            true,
					},
					"organization_root_account_external_id": schema.StringAttribute{
						MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the organization root account role.",
						Optional:            true,
					},
					"service_user_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service user configuration (alternative to `workload_identity_config`)",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"access_key": schema.StringAttribute{
								MarkdownDescription: "AWS access key for service user",
								Required:            true,
							},
							"secret_key": schema.StringAttribute{
								MarkdownDescription: "AWS secret key for service user",
								Optional:            true,
								Sensitive:           true,
							},
						},
					},
					"workload_identity_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Workload identity configuration (alternative to `service_user_config`)",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"role_arn": schema.StringAttribute{
								MarkdownDescription: "ARN of the role that should be used as the entry point for meshStack by assuming it via web identity.",
								Required:            true,
							},
						},
					},
				},
			},
			"wait_for_external_avm": schema.BoolAttribute{
				MarkdownDescription: "Flag to wait for external AVM. Please use this setting with care! It is currently very specific to certain tags being present on the account! In general, we recommend not to activate this functionality! In a meshLandingZone an AVM can be triggered via an AWS StackSet or via a Lambda Function. If meshStack shall wait for the AVM to complete when creating a new platform tenant, this flag must be checked. meshStack will identify completion of the AVM by checking the presence of the following tags on the AWS account: 'ProductName' is set to workspace identifier and 'Stage' is set to project identifier.",
				Optional:            true,
			},
			"automation_account_role": schema.StringAttribute{
				MarkdownDescription: "ARN of the Automation Account Role. The Automation Account contains all AWS StackSets and Lambda Functions that shall be executed via meshLandingZones. E.g. `arn:aws:iam::123456789:role/MeshfedAutomationRole`.",
				Optional:            true,
			},
			"automation_account_external_id": schema.StringAttribute{
				MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the automation account role.",
				Optional:            true,
			},
			"account_access_role": schema.StringAttribute{
				MarkdownDescription: "The name for the Account Access Role that will be rolled out to all managed accounts. Only a name, not an ARN must be set here, as the ARN must be built dynamically for every managed AWS Account. The replicator service user needs to assume this role in all accounts to manage them.",
				Optional:            true,
			},
			"account_alias_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account alias of the created AWS account will be named. E.g. `#{workspaceIdentifier}-#{projectIdentifier}`. Attention: Account Alias must be globally unique in AWS. So consider defining a unique prefix.",
				Optional:            true,
			},
			"enforce_account_alias": schema.BoolAttribute{
				MarkdownDescription: "Flag to enforce account alias. If set, meshStack will guarantee on every replication that the configured Account Alias is applied. Otherwise it will only set the Account Alias once during tenant creation.",
				Optional:            true,
			},
			"account_email_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account email address of the created AWS account will be set. E.g. `aws+#{workspaceIdentifier}.#{projectIdentifier}@yourcompany.com`. Please consider that this email address is limited to 64 characters! Also have a look at our docs for more information.",
				Optional:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "Namespace prefix for tenant tags",
						Required:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Required:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"aws_sso": schema.SingleNestedAttribute{
				MarkdownDescription: "AWS SSO configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"scim_endpoint": schema.StringAttribute{
						MarkdownDescription: "The SCIM endpoint you can find in your AWS IAM Identity Center Automatic provisioning config.",
						Required:            true,
					},
					"arn": schema.StringAttribute{
						MarkdownDescription: "The ARN of your AWS IAM Identity Center Instance. E.g. `arn:aws:sso:::instance/ssoins-123456789abc`.",
						Required:            true,
					},
					"group_name_pattern": schema.StringAttribute{
						MarkdownDescription: "Configures the pattern that defines the desired name of AWS IAM Identity Center groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names will be unique within the same AWS IAM Identity Center Instance with that configuration. meshStack will additionally prefix the group name with 'mst-' to be able to identify the groups that are managed by meshStack.",
						Required:            true,
					},
					"sso_access_token": schema.StringAttribute{
						MarkdownDescription: "The AWS IAM Identity Center SCIM Access Token that was generated via the Automatic provisioning config in AWS IAM Identity Center.",
						Optional:            true,
						Sensitive:           true,
					},
					"aws_role_mappings": schema.ListNestedAttribute{
						MarkdownDescription: "AWS role mappings for AWS SSO",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"project_role_ref": meshProjectRoleAttribute(),
								"aws_role": schema.StringAttribute{
									MarkdownDescription: "The AWS role name",
									Required:            true,
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
						Optional:            true,
					},
				},
			},
			"enrollment_configuration": schema.SingleNestedAttribute{
				MarkdownDescription: "AWS account enrollment configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"management_account_id": schema.StringAttribute{
						MarkdownDescription: "The Account ID of the management account configured for the platform instance.",
						Required:            true,
					},
					"account_factory_product_id": schema.StringAttribute{
						MarkdownDescription: "The Product ID of the AWS Account Factory Product in AWS Service Catalog that should be used for enrollment. Starts with `prod-`.",
						Required:            true,
					},
				},
			},
			"self_downgrade_access_role": schema.BoolAttribute{
				MarkdownDescription: "Flag for self downgrade access role. If set, meshStack will revoke its rights on the managed account that were only needed for initial account creation.",
				Optional:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the AWS platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Optional:            true,
			},
			"allow_hierarchical_organizational_unit_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical organizational unit assignment in AWS. If set to true: Accounts can be moved to child organizational units of the organizational unit defined in the Landing Zone. This is useful if you want to manage the account location with a deeper and more granular hierarchy. If set to false: Accounts will always be moved directly to the organizational unit defined in the Landing Zone.",
				Optional:            true,
			},
		},
	}
}

func azureReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure-specific replication configuration for the platform.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for Azure",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Required:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Required:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret (if authType is `CREDENTIALS`)",
						Optional:            true,
						Sensitive:           true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
				},
			},
			"provisioning": schema.SingleNestedAttribute{
				MarkdownDescription: "To provide Azure Subscription for your organization’s meshProjects, meshcloud supports using Enterprise Enrollment or allocating from a pool of pre-provisioned subscriptions. One of the subFields enterpriseEnrollment, customerAgreement or preProvisioned must be provided!",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"subscription_owner_object_ids": schema.ListAttribute{
						MarkdownDescription: "One or more principals Object IDs (e.g. user groups, SPNs) that meshStack will ensure have an 'Owner' role assignment on the managed subscriptions. This can be useful to satisfy Azure’s constraint of at least one direct 'Owner' role assignment per Subscription. If you want to use a Service Principal please use the Enterprise Application Object ID. You can not use the replicator object ID here, because meshStack always removes its high privilege access after a Subscription creation.",
						Optional:            true,
						ElementType:         types.StringType,
					},
					"enterprise_enrollment": schema.SingleNestedAttribute{
						MarkdownDescription: "meshcloud can automatically provision new subscriptions from an Enterprise Enrollment Account owned by your organization. This is suitable for large organizations that have a Microsoft Enterprise Agreement, Microsoft Customer Agreement or a Microsoft Partner Agreement and want to provide a large number of subscriptions in a fully automated fashion.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"enrollment_account_id": schema.StringAttribute{
								MarkdownDescription: "ID of the EA Enrollment Account used for the Subscription creation. Should look like this: `/providers/Microsoft.Billing/billingAccounts/1234567/enrollmentAccounts/7654321`. For more information, review the [Azure docs](https://docs.microsoft.com/en-us/azure/cost-management-billing/manage/programmatically-create-subscription-enterprise-agreement?tabs=rest-getEnrollments%2Crest-EA#find-accounts-you-have-access-to).",
								Required:            true,
							},
							"subscription_offer_type": schema.StringAttribute{
								MarkdownDescription: "The Microsoft Subscription offer type to use when creating subscriptions. Only Production for standard and DevTest for Dev/Test subscriptions are supported for the Non Legacy Subscription Enrollment. For the Legacy Subscription Enrollment also other types can be defined.",
								Required:            true,
							},
							"use_legacy_subscription_enrollment": schema.BoolAttribute{
								MarkdownDescription: "Deprecated: Uses the old Subscription enrollment API in its preview version. This enrollment is less reliable and should not be used for new Azure Platform Integrations.",
								Optional:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure’s MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Optional:            true,
							},
						},
					},
					"customer_agreement": schema.SingleNestedAttribute{
						MarkdownDescription: "meshcloud can automatically provision new subscriptions from a Customer Agreement Account owned by your organization. This is suitable for larger organizations that have such a Customer Agreement with Microsoft, and want to provide a large number of subscriptions in a fully automated fashion.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"source_service_principal": schema.SingleNestedAttribute{
								MarkdownDescription: "Configure the SPN used by meshStack to create a new Subscription in your MCA billing scope. For more information on the required permissions, see the [Azure docs](https://learn.microsoft.com/en-us/azure/cost-management-billing/manage/programmatically-create-subscription-microsoft-customer-agreement-across-tenants).",
								Optional:            true,
								Attributes: map[string]schema.Attribute{
									"client_id": schema.StringAttribute{
										MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the \"Enterprise Application\" but can also be retrieved via the \"App Registration\" object as \"Application (Client) ID\".",
										Required:            true,
									},
									"auth_type": schema.StringAttribute{
										MarkdownDescription: "Must be one of `CREDENTIALS` or `WORKLOAD_IDENTITY`. Workload Identity Federation is the one that we recommend as it enables the most secure approach to provide access to your Azure tenant without using long lived credentials. Credential Authentication is an alternative approach where you have to provide a clientSecret manually to meshStack and meshStack stores it encrypted.",
										Required:            true,
									},
									"credentials_auth_client_secret": schema.StringAttribute{
										MarkdownDescription: "Must be set if and only if authType is CREDENTIALS. A valid secret for accessing the application. In Azure Portal, this can be configured on the \"App Registration\" under Certificates & secrets. [How is this information secured?](https://docs.meshcloud.io/operations/security-faq/#how-does-meshstack-securely-handle-my-cloud-platform-credentials)",
										Optional:            true,
										Sensitive:           true,
									},
								},
							},
							"destination_entra_id": schema.StringAttribute{
								MarkdownDescription: "Microsoft Entra ID Tenant UUID where created subscriptions should be moved. Set this to the Microsoft Entra ID Tenant hosting your landing zones.",
								Required:            true,
							},
							"source_entra_tenant": schema.StringAttribute{
								MarkdownDescription: "Microsoft Entra ID Tenant UUID or domain name used for creating subscriptions. Set this to the Microsoft Entra ID Tenant owning the MCA Billing Scope. If source and destination Microsoft Entra ID Tenants are the same, you need to use UUID.",
								Required:            true,
							},
							"billing_scope": schema.StringAttribute{
								MarkdownDescription: "ID of the MCA Billing Scope used for creating subscriptions. Must follow this format: `/providers/Microsoft.Billing/billingAccounts/$accountId/billingProfiles/$profileId/invoiceSections/$sectionId`.",
								Required:            true,
							},
							"subscription_creation_error_cooldown_sec": schema.Int64Attribute{
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure’s MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Optional:            true,
							},
						},
					},
					"pre_provisioned": schema.SingleNestedAttribute{
						MarkdownDescription: "If your organization does not have access to an Enterprise Enrollment, you can alternatively configure meshcloud to consume subscriptions from a pool of externally-provisioned subscriptions. This is useful for smaller organizations that wish to use 'Pay-as-you-go' subscriptions or if you’re organization partners with an Azure Cloud Solution Provider to provide your subscriptions. The meshcloud Azure replication detects externally-provisioned subscriptions based on a configurable prefix in the subscription name. Upon assignment to a meshProject, the subscription is inflated with the right Landing Zone configuration and removed from the subscription pool.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"unused_subscription_name_prefix": schema.StringAttribute{
								MarkdownDescription: "The prefix that identifies unused subscriptions. Subscriptions will be renamed during meshStack’s project replication, at which point they should no longer carry this prefix.",
								Required:            true,
							},
						},
					},
				},
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional B2B user invitation configuration. When configured, instructs the replicator to create AAD B2B guest invitations for users missing in the AAD tenant managed by this meshPlatform.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure’s consent experience redirects users to after they accept their invitation.",
						Optional:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Optional:            true,
					},
				},
			},
			"subscription_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of Azure Subscriptions managed by meshStack.",
				Optional:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names are unique in the managed AAD Tenant.",
				Optional:            true,
			},
			"blueprint_service_principal": schema.StringAttribute{
				MarkdownDescription: " \t\n\nObject ID of the Enterprise Application belonging to the Microsoft Application 'Azure Blueprints'. meshStack will grant the necessary permissions on managed Subscriptions to this SPN so that it can create System Assigned Managed Identities (SAMI) for Blueprint execution.",
				Optional:            true,
			},
			"blueprint_location": schema.StringAttribute{
				MarkdownDescription: "The Azure location where replication creates and updates Blueprint Assignments. Note that it’s still possible that the Blueprint creates resources in other locations, this is merely the location where the Blueprint Assignment is managed.",
				Optional:            true,
			},
			"azure_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Azure role mappings for Azure role definitions.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"azure_role": schema.SingleNestedAttribute{
							MarkdownDescription: "The Azure role definition.",
							Required:            true,
							Attributes: map[string]schema.Attribute{
								"alias": schema.StringAttribute{
									MarkdownDescription: "The alias/name of the Azure role.",
									Required:            true,
								},
								"id": schema.StringAttribute{
									MarkdownDescription: "The Azure role definition ID.",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tagging configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
						Required:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Required:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy (`userPrincipalName` or `email`). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Optional:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Optional:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to sub management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Optional:            true,
			},
		},
	}
}

func azureRgReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group-specific replication configuration for the platform.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for Azure Resource Group access.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Required:            true,
					},
					"auth_type": schema.StringAttribute{
						MarkdownDescription: "Authentication type (`CREDENTIALS` or `WORKLOAD_IDENTITY`)",
						Required:            true,
					},
					"credentials_auth_client_secret": schema.StringAttribute{
						MarkdownDescription: "Client secret (if authType is `CREDENTIALS`)",
						Optional:            true,
						Sensitive:           true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
				},
			},
			"subscription": schema.StringAttribute{
				MarkdownDescription: "The Subscription that will contain all the created Resource Groups. Once you set the Subscription, you must not change it.",
				Optional:            true,
			},
			"resource_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name Resource Group managed by meshStack. It follows the usual replicator string pattern features. Operators must ensure the group names are unique within the Subscription.",
				Optional:            true,
			},
			"user_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix. This suffix is configurable via Role Mappings in this platform config.",
				Optional:            true,
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional B2B user invitation configuration. When configured, instructs the replicator to create AAD B2B guest invitations for users missing in the AAD tenant managed by this meshPlatform.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure’s consent experience redirects users to after they accept their invitation.",
						Optional:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Optional:            true,
					},
				},
			},
			"user_look_up_strategy": schema.StringAttribute{
				MarkdownDescription: "User lookup strategy (`userPrincipalName` or `email`). Users can either be looked up in cloud platforms by email or UPN (User Principal Name). In most cases email is the matching way as it is the only identifier that is consistently used throughout all cloud platforms and meshStack.",
				Optional:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
						Required:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Required:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Optional:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to child management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Optional:            true,
			},
		},
	}
}

func gcpReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP-specific replication configuration for the platform.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_account_config": schema.SingleNestedAttribute{
				MarkdownDescription: "Service account configuration. Either `serviceAccountCredentialsConfig` or `serviceAccountWorkloadIdentityConfig` must be provided.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"service_account_credentials_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service account credentials configuration (alternative to serviceAccountWorkloadIdentityConfig)",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"service_account_credentials_b64": schema.StringAttribute{
								MarkdownDescription: "Base64 encoded credentials.json file for a GCP ServiceAccount. The replicator uses this Service Account to automate GCP API operations (IAM, ResourceManager etc.).",
								Optional:            true,
								Sensitive:           true,
							},
						},
					},
					"service_account_workload_identity_config": schema.SingleNestedAttribute{
						MarkdownDescription: "Service account workload identity configuration (alternative to serviceAccountCredentialsConfig)",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"audience": schema.StringAttribute{
								MarkdownDescription: "The audience associated with your workload identity pool provider.",
								Optional:            true,
							},
							"service_account_email": schema.StringAttribute{
								MarkdownDescription: "The email address of the Service Account, that gets impersonated for calling Google APIs via Workload Identity Federation.",
								Optional:            true,
							},
						},
					},
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "The domain used for cloud identity directory-groups created and managed by meshStack. meshStack maintains separate groups for each meshProject role on each managed GCP project.",
				Optional:            true,
			},
			"customer_id": schema.StringAttribute{
				MarkdownDescription: "A Google Customer ID. It typically starts with a 'C'.",
				Optional:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. Additionally you can also use 'platformGroupAlias' as a placeholder to access the specific project role from the role mappings done in this platform configuration or in the meshLandingZone configuration.",
				Optional:            true,
			},
			"project_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. The result must be 4 to 30 characters. Allowed characters are: lowercase and uppercase letters, numbers, hyphen, single-quote, double-quote, space, and exclamation point. When length restrictions are applied, the abbreviation will be in the middle and marked by a single-quote.",
				Optional:            true,
			},
			"project_id_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. The resulting string must not exceed a total length of 30 characters. Only alphanumeric + hyphen are allowed. We recommend that configuration include at least 3 characters of the random parameter to reduce the chance of naming collisions as the project Ids must be globally unique within GCP.",
				Optional:            true,
			},
			"billing_account_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the billing account to associate with all GCP projects managed by meshStack",
				Optional:            true,
			},
			"user_lookup_strategy": schema.StringAttribute{
				MarkdownDescription: "Users can either be looked up by E-Mail or externalAccountId. This must also be the property that is placed in the external user id (EUID) of your meshUser entity to match. E-Mail is usually a good choice as this is often set up as the EUID throughout all cloud platforms and meshStack. ('email' or 'externalId')",
				Optional:            true,
			},
			"gcp_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Mapping of platform roles to GCP IAM roles.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"gcp_role": schema.StringAttribute{
							MarkdownDescription: "The GCP IAM role",
							Required:            true,
						},
					},
				},
			},
			"allow_hierarchical_folder_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical folder assignment in GCP. If set to true: Projects can be moved to sub folders of the folder defined in the Landing Zone. This is useful if you want to manage the project location with a deeper and more granular hierarchy. If set to false: Projects will always be moved directly to the folder defined in the Landing Zone.",
				Optional:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "Namespace prefix for tenant tags",
						Optional:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Required:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Required:            true,
								},
							},
						},
					},
				},
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the GCP platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Optional:            true,
			},
		},
	}
}

func kubernetesClientConfigSchema(description string) schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"access_token": schema.StringAttribute{
				MarkdownDescription: "The Access Token of the service account for replicator access.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func meteringProcessingConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Processing configuration for metering",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"compact_timelines_after_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days after which timelines should be compacted.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(30),
			},
			"delete_raw_data_after_days": schema.Int64Attribute{
				MarkdownDescription: "Number of days after which raw data should be deleted.",
				Computed:            true,
				Optional:            true,
				Default:             int64default.StaticInt64(65),
			},
		},
	}
}

func kubernetesReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for Kubernetes (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for Kubernetes"),
			"namespace_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. Kubernetes Namespace Names must be no longer than 63 characters, must start and end with a lowercase letter or number, and may contain lowercase letters, numbers, and hyphens.",
				Optional:            true,
			},
		},
	}
}

func kubernetesMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for Kubernetes (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for Kubernetes metering"),
			"processing":    meteringProcessingConfigSchema(),
		},
	}
}

func openShiftReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for OpenShift (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for OpenShift"),
			"web_console_url": schema.StringAttribute{
				MarkdownDescription: "The Web Console URL that is used to redirect the user to the cloud platform. An example Web Console URL is https://console-openshift-console.apps.okd4.dev.eu-de-central.msh.host",
				Optional:            true,
			},
			"project_name_pattern": schema.StringAttribute{
				MarkdownDescription: "All the commonly available replicator string template properties are available. OpenShift Project Names must be no longer than 63 characters, must start and end with a lowercase letter or number, and may contain lowercase letters, numbers, and hyphens.",
				Optional:            true,
			},
			"enable_template_instantiation": schema.BoolAttribute{
				MarkdownDescription: "Here you can enable templates not only being rolled out to OpenShift but also instantiated during replication. Templates can be configured in meshLandingZones. Please keep in mind that the replication service account needs all the rights that are required to apply the templates that are configured in meshLandingZones.",
				Optional:            true,
			},
			"openshift_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "OpenShift role mappings for OpenShift roles.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(),
						"openshift_role": schema.StringAttribute{
							MarkdownDescription: "The OpenShift role name",
							Required:            true,
						},
					},
				},
			},
			"identity_provider_name": schema.StringAttribute{
				MarkdownDescription: "Identity provider name",
				Optional:            true,
			},
			"tenant_tags": schema.SingleNestedAttribute{
				MarkdownDescription: "Tenant tags configuration",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace_prefix": schema.StringAttribute{
						MarkdownDescription: "This is the prefix for all labels created by meshStack. It helps to keep track of which labels are managed by meshStack. It is recommended to let this prefix end with a delimiter like an underscore.",
						Optional:            true,
					},
					"tag_mappers": schema.ListNestedAttribute{
						MarkdownDescription: "List of tag mappers for tenant tags",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"key": schema.StringAttribute{
									MarkdownDescription: "Key for the tag mapper",
									Required:            true,
								},
								"value_pattern": schema.StringAttribute{
									MarkdownDescription: "Value pattern for the tag mapper",
									Required:            true,
								},
							},
						},
					},
				},
			},
		},
	}
}

func openShiftMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for OpenShift (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for OpenShift metering"),
			"processing":    meteringProcessingConfigSchema(),
		},
	}
}

func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	platform := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{},
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdPlatform, err := r.client.CreatePlatform(&platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Platform",
			"Could not create platform, unexpected error: "+err.Error(),
		)
		return
	}

	handleObfuscatedSecrets(&createdPlatform.Spec.Config, &platform.Spec.Config, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, createdPlatform)...)
}

func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get the resource ID (which should be the UUID)
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	readPlatform, err := r.client.ReadPlatform(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read platform with UUID '%s'", uuid),
			err.Error(),
		)
		return
	}

	if readPlatform == nil {
		// The platform was deleted outside of Terraform, so we remove it from the state
		resp.State.RemoveResource(ctx)
		return
	}

	statePlatformSpec := client.MeshPlatformSpec{}
	req.State.GetAttribute(ctx, path.Root("spec"), &statePlatformSpec)
	if resp.Diagnostics.HasError() {
		return
	}

	handleObfuscatedSecrets(&readPlatform.Spec.Config, &statePlatformSpec.Config, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, readPlatform)...)
}

func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	platform := client.MeshPlatformUpdate{
		Metadata: client.MeshPlatformUpdateMetadata{},
	}

	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)

	if uuid == "" {
		resp.Diagnostics.AddError(
			"Resource ID Missing",
			"The resource ID is missing. This should not happen.",
		)
		return
	}

	// Retrieve values from plan
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &platform.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &platform.Spec)...)

	// Handle metadata fields including UUID for updates
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &platform.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &platform.Metadata.OwnedByWorkspace)...)
	platform.Metadata.Uuid = uuid

	if resp.Diagnostics.HasError() {
		return
	}

	updatedPlatform, err := r.client.UpdatePlatform(uuid, &platform)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}

	handleObfuscatedSecrets(&updatedPlatform.Spec.Config, &platform.Spec.Config, resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedPlatform)...)
}

func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var uuid string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &uuid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlatform(uuid)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete platform with UUID '%s'", uuid),
			err.Error(),
		)
		return
	}
}

func (r *platformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
