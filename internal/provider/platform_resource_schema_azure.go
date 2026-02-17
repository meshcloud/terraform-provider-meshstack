package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Plain Azure

func azurePlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Required:            true,
			},
			"replication": azureReplicationConfigSchema(),
			"metering":    azureMeteringConfigSchema(),
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
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID.",
						Required:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
					"auth": azureAuthSchema(),
				},
			},
			"update_subscription_name": schema.BoolAttribute{
				MarkdownDescription: "Update existing subscription names to match the subscription name pattern during replication.",
				Required:            true,
			},
			"provisioning": schema.SingleNestedAttribute{
				MarkdownDescription: "To provide Azure Subscription for your organization's meshProjects, meshcloud supports using Enterprise Enrollment or allocating from a pool of pre-provisioned subscriptions. One of the subFields enterpriseEnrollment, customerAgreement or preProvisioned must be provided!",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"subscription_owner_object_ids": schema.ListAttribute{
						MarkdownDescription: "One or more principals Object IDs (e.g. user groups, SPNs) that meshStack will ensure have an 'Owner' role assignment on the managed subscriptions. This can be useful to satisfy Azure's constraint of at least one direct 'Owner' role assignment per Subscription. If you want to use a Service Principal please use the Enterprise Application Object ID. You can not use the replicator object ID here, because meshStack always removes its high privilege access after a Subscription creation.",
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
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure's MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"customer_agreement": schema.SingleNestedAttribute{
						MarkdownDescription: "meshcloud can automatically provision new subscriptions from a Customer Agreement Account owned by your organization. This is suitable for larger organizations that have such a Customer Agreement with Microsoft, and want to provide a large number of subscriptions in a fully automated fashion.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"source_service_principal": schema.SingleNestedAttribute{
								MarkdownDescription: "Configure the SPN used by meshStack to create a new Subscription in your MCA billing scope. For more information on the required permissions, see the [Azure docs](https://learn.microsoft.com/en-us/azure/cost-management-billing/manage/programmatically-create-subscription-microsoft-customer-agreement-across-tenants).",
								Required:            true,
								Attributes: map[string]schema.Attribute{
									"client_id": schema.StringAttribute{
										MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the \"Enterprise Application\" but can also be retrieved via the \"App Registration\" object as \"Application (Client) ID\".",
										Required:            true,
									},
									"auth": azureAuthSchema(),
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
								MarkdownDescription: "This value must be defined in seconds. It is a safety mechanism to avoid duplicate Subscription creation in case of an error on Azure's MCA API. This delay should be a bit higher than it usually takes to create subscriptions. For big installations this is somewhere between 5-15 minutes. The default of 900s should be fine for most installations.",
								Optional:            true,
								Computed:            true,
							},
						},
					},
					"pre_provisioned": schema.SingleNestedAttribute{
						MarkdownDescription: "If your organization does not have access to an Enterprise Enrollment, you can alternatively configure meshcloud to consume subscriptions from a pool of externally-provisioned subscriptions. This is useful for smaller organizations that wish to use 'Pay-as-you-go' subscriptions or if you're organization partners with an Azure Cloud Solution Provider to provide your subscriptions. The meshcloud Azure replication detects externally-provisioned subscriptions based on a configurable prefix in the subscription name. Upon assignment to a meshProject, the subscription is inflated with the right Landing Zone configuration and removed from the subscription pool.",
						Optional:            true,
						Attributes: map[string]schema.Attribute{
							"unused_subscription_name_prefix": schema.StringAttribute{
								MarkdownDescription: "The prefix that identifies unused subscriptions. Subscriptions will be renamed during meshStack's project replication, at which point they should no longer carry this prefix.",
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
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Required:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Required:            true,
					},
				},
			},
			"subscription_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of Azure Subscriptions managed by meshStack.",
				Required:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix, which is configurable via Role Mappings in this platform config or via a meshLandingZone. Operators must ensure the group names are unique in the managed AAD Tenant.",
				Required:            true,
			},
			"blueprint_service_principal": schema.StringAttribute{
				MarkdownDescription: " \t\n\nObject ID of the Enterprise Application belonging to the Microsoft Application 'Azure Blueprints'. meshStack will grant the necessary permissions on managed Subscriptions to this SPN so that it can create System Assigned Managed Identities (SAMI) for Blueprint execution.",
				Required:            true,
			},
			"blueprint_location": schema.StringAttribute{
				MarkdownDescription: "The Azure location where replication creates and updates Blueprint Assignments. Note that it's still possible that the Blueprint creates resources in other locations, this is merely the location where the Blueprint Assignment is managed.",
				Required:            true,
			},
			"azure_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "Azure role mappings for Azure role definitions.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(false),
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
			"tenant_tags":          tenantTagsAttribute(),
			"user_lookup_strategy": azureUserLookupStrategySchema(),
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "Flag to skip user group permission cleanup. For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Required:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to sub management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Required:            true,
			},
		},
	}
}

func azureMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for Azure (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for Azure metering",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Required:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
					"auth": azureAuthSchema(),
				},
			},
			"processing": meteringProcessingConfigSchema(),
		},
	}
}

// Azure RG

func azureRgPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Resource Group platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"entra_tenant": schema.StringAttribute{
				MarkdownDescription: "Azure Active Directory (Entra ID) tenant",
				Required:            true,
			},
			"replication": azureRgReplicationConfigSchema(),
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
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID",
						Required:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
					"auth": azureAuthSchema(),
				},
			},
			"subscription": schema.StringAttribute{
				MarkdownDescription: "The Subscription that will contain all the created Resource Groups. Once you set the Subscription, you must not change it.",
				Required:            true,
			},
			"resource_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name Resource Group managed by meshStack. It follows the usual replicator string pattern features. Operators must ensure the group names are unique within the Subscription.",
				Required:            true,
			},
			"user_group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Configures the pattern that defines the desired name of AAD groups managed by meshStack. It follows the usual replicator string pattern features and provides the additional replacement 'platformGroupAlias', which contains the role name suffix. This suffix is configurable via Role Mappings in this platform config.",
				Required:            true,
			},
			"b2b_user_invitation": schema.SingleNestedAttribute{
				MarkdownDescription: "Optional B2B user invitation configuration. When configured, instructs the replicator to create AAD B2B guest invitations for users missing in the AAD tenant managed by this meshPlatform.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"redirect_url": schema.StringAttribute{
						MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
						Required:            true,
					},
					"send_azure_invitation_mail": schema.BoolAttribute{
						MarkdownDescription: "When true, meshStack instructs Azure to send out Invitation mails to invited users. These mails allow users to redeem their invitation to the AAD tenant only using email and Azure Portal.",
						Required:            true,
					},
				},
			},
			"user_lookup_strategy": azureUserLookupStrategySchema(),
			"tenant_tags":          tenantTagsAttribute(),
			"skip_user_group_permission_cleanup": schema.BoolAttribute{
				MarkdownDescription: "For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the Azure platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later.",
				Required:            true,
			},
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
			"allow_hierarchical_management_group_assignment": schema.BoolAttribute{
				MarkdownDescription: "Configuration flag to enable or disable hierarchical management group assignment in Azure. If set to true: Subscriptions can be moved to child management groups of the management group defined in the Landing Zone. This is useful if you want to manage the subscription location with a deeper and more granular hierarchy. If set to false: Subscriptions will always be moved directly to the management group defined in the Landing Zone.",
				Required:            true,
			},
		},
	}
}
func azureUserLookupStrategySchema() schema.Attribute {
	return schema.StringAttribute{
		MarkdownDescription: "Strategy for user lookup in Azure (`UserByMailLookupStrategy` or `UserByUsernameLookupStrategy`)",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf([]string{"UserByMailLookupStrategy", "UserByUsernameLookupStrategy"}...),
		},
	}
}

func azureAuthSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Authentication configuration\n\n" +
			"meshStack supports 2 types of authentication for Azure service principals:\n\n" +
			"1. **Workload Identity Federation (Recommended)**: Uses OIDC to enable secure access without long-lived credentials. " +
			"To enable workload identity federation, set `auth = {}` (empty object). The `type` will automatically be set to `workloadIdentity`.\n\n" +
			"2. **Credential-based authentication**: Uses client secrets for authentication. Provide the `credential` block with the plaintext secret.",
		Required: true,
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Authentication type (credential or workloadIdentity)",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{authTypeDefault()},
			},
			"credential": secretEmbeddedSchema("Client secret (if type is credential)", true),
		},
	}
}
