package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
			"metering":    awsMeteringConfigSchema(),
		},
	}
}

func awsAccessConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "meshStack currently supports 2 types of authentication. Workload Identity Federation (using OIDC) is the one that we recommend as it enables secure access to your AWS account without using long lived credentials. Alternatively, you can use credential based authentication by providing access and secret keys. Either the `service_user_config` or `workload_identity_config` must be provided.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"organization_root_account_role": schema.StringAttribute{
				MarkdownDescription: "ARN of the Management Account Role. The Management Account contains your AWS organization. E.g. `arn:aws:iam::123456789:role/MeshfedServiceRole`.",
				Required:            true,
			},
			"organization_root_account_external_id": schema.StringAttribute{
				MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the organization root account role.",
				Optional:            true,
			},
			"auth": schema.SingleNestedAttribute{
				MarkdownDescription: "Authentication configuration",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						MarkdownDescription: "Authentication type (credential or workloadIdentity)",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{authTypeDefault()},
					},
					"credential": schema.SingleNestedAttribute{
						MarkdownDescription: "Service user credential configuration",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"access_key": schema.StringAttribute{
								MarkdownDescription: "AWS access key for service user",
								Required:            true,
							},
							"secret_key": secretEmbeddedSchema("AWS secret key for service user", false),
						},
					},
					"workload_identity": schema.SingleNestedAttribute{
						MarkdownDescription: "Workload identity configuration",
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
		},
	}
}

func awsMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for AWS (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"access_config": awsAccessConfigSchema(),
			"filter": schema.StringAttribute{
				MarkdownDescription: "Filter for AWS metering data.",
				Required:            true,
			},
			"reserved_instance_fair_chargeback": schema.BoolAttribute{
				MarkdownDescription: "Flag to enable fair chargeback for reserved instances.",
				Required:            true,
			},
			"savings_plan_fair_chargeback": schema.BoolAttribute{
				MarkdownDescription: "Flag to enable fair chargeback for savings plans.",
				Required:            true,
			},
			"processing": meteringProcessingConfigSchema(),
		},
	}
}

func awsReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AWS (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"access_config": awsAccessConfigSchema(),
			"wait_for_external_avm": schema.BoolAttribute{
				MarkdownDescription: "Flag to wait for external AVM. Please use this setting with care! It is currently very specific to certain tags being present on the account! In general, we recommend not to activate this functionality! In a meshLandingZone an AVM can be triggered via an AWS StackSet or via a Lambda Function. If meshStack shall wait for the AVM to complete when creating a new platform tenant, this flag must be checked. meshStack will identify completion of the AVM by checking the presence of the following tags on the AWS account: 'ProductName' is set to workspace identifier and 'Stage' is set to project identifier.",
				Required:            true,
			},
			"automation_account_role": schema.StringAttribute{
				MarkdownDescription: "ARN of the Automation Account Role. The Automation Account contains all AWS StackSets and Lambda Functions that shall be executed via meshLandingZones. E.g. `arn:aws:iam::123456789:role/MeshfedAutomationRole`.",
				Required:            true,
			},
			"automation_account_external_id": schema.StringAttribute{
				MarkdownDescription: "ExternalId to enhance security in a multi account setup when assuming the automation account role.",
				Optional:            true,
			},
			"account_access_role": schema.StringAttribute{
				MarkdownDescription: "The name for the Account Access Role that will be rolled out to all managed accounts. Only a name, not an ARN must be set here, as the ARN must be built dynamically for every managed AWS Account. The replicator service user needs to assume this role in all accounts to manage them.",
				Required:            true,
			},
			"account_alias_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account alias of the created AWS account will be named. E.g. `#{workspaceIdentifier}-#{projectIdentifier}`. Attention: Account Alias must be globally unique in AWS. So consider defining a unique prefix.",
				Required:            true,
			},
			"enforce_account_alias": schema.BoolAttribute{
				MarkdownDescription: "Flag to enforce account alias. If set, meshStack will guarantee on every replication that the configured Account Alias is applied. Otherwise it will only set the Account Alias once during tenant creation.",
				Required:            true,
			},
			"account_email_pattern": schema.StringAttribute{
				MarkdownDescription: "With a String Pattern you can define how the account email address of the created AWS account will be set. E.g. `aws+#{workspaceIdentifier}.#{projectIdentifier}@yourcompany.com`. Please consider that this email address is limited to 64 characters! Also have a look at our docs for more information.",
				Required:            true,
			},
			"tenant_tags": tenantTagsAttribute(),
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
					"sso_access_token": secretEmbeddedSchema("The AWS IAM Identity Center SCIM Access Token that was generated via the Automatic provisioning config in AWS IAM Identity Center.", true),
					"aws_role_mappings": schema.ListNestedAttribute{
						MarkdownDescription: "AWS role mappings for AWS SSO",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"project_role_ref": meshProjectRoleAttribute(false),
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
				Required:            true,
			},
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the AWS platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Required:            true,
			},
			"allow_hierarchical_organizational_unit_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical organizational unit assignment in AWS. If set to true: Accounts can be moved to child organizational units of the organizational unit defined in the Landing Zone. This is useful if you want to manage the account location with a deeper and more granular hierarchy. If set to false: Accounts will always be moved directly to the organizational unit defined in the Landing Zone.",
				Required:            true,
			},
		},
	}
}
