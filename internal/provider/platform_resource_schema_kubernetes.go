package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
)

// Vanilla Kubernetes

func kubernetesBasedPlatformAttributes(platformName string, exampleBaseUrl string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base_url": schema.StringAttribute{
			MarkdownDescription: "This is the base URL to your " + platformName + " cluster, which is used to call the APIs to create new " + platformName + " tenants, get raw data for metering the " + platformName + " tenants, etc. An example base URL is: " + exampleBaseUrl,
			Required:            true,
		},
		"disable_ssl_validation": schema.BoolAttribute{
			MarkdownDescription: "Flag to disable SSL validation for the " + platformName + " cluster. SSL Validation should at best never be disabled, but for integration of some private cloud platforms in an early state, they might not yet be using valid SSL certificates. In that case it can make sense to disable SSL validation here to already test integration of these platforms.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

func kubernetesPlatformSchema() schema.Attribute {
	attributes := kubernetesBasedPlatformAttributes("Kubernetes", "https://k8s.dev.eu-de-central.msh.host:6443")
	attributes["replication"] = kubernetesReplicationConfigSchema()
	attributes["metering"] = kubernetesMeteringConfigSchema()

	return schema.SingleNestedAttribute{
		MarkdownDescription: "Kubernetes platform configuration.",
		Optional:            true,
		Attributes:          attributes,
	}
}

func kubernetesClientConfigSchema(description string) schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"access_token": secretEmbeddedSchema("The Access Token of the service account for replicator access.", false),
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
				Required:            true,
			},
		},
	}
}

func kubernetesBasedMeteringConfigSchema(platformName string) schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for " + platformName + " (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"client_config": kubernetesClientConfigSchema("Client configuration for " + platformName + " metering"),
			"processing":    meteringProcessingConfigSchema(),
		},
	}
}

func kubernetesMeteringConfigSchema() schema.Attribute {
	return kubernetesBasedMeteringConfigSchema("Kubernetes")
}

// OpenShift (OKD)

func openShiftPlatformSchema() schema.Attribute {
	attributes := kubernetesBasedPlatformAttributes("OpenShift", "https://api.okd4.dev.eu-de-central.msh.host:6443")
	attributes["replication"] = openShiftReplicationConfigSchema()
	attributes["metering"] = openShiftMeteringConfigSchema()
	return schema.SingleNestedAttribute{
		MarkdownDescription: "OpenShift platform configuration.",
		Optional:            true,
		Attributes:          attributes,
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
				Required:            true,
			},
			"enable_template_instantiation": schema.BoolAttribute{
				MarkdownDescription: "Here you can enable templates not only being rolled out to OpenShift but also instantiated during replication. Templates can be configured in meshLandingZones. Please keep in mind that the replication service account needs all the rights that are required to apply the templates that are configured in meshLandingZones.",
				Required:            true,
			},
			"openshift_role_mappings": schema.ListNestedAttribute{
				MarkdownDescription: "OpenShift role mappings for OpenShift roles.",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshProjectRoleAttribute(false),
						"openshift_role": schema.StringAttribute{
							MarkdownDescription: "The OpenShift role name",
							Required:            true,
						},
					},
				},
			},
			"identity_provider_name": schema.StringAttribute{
				MarkdownDescription: "Identity provider name",
				Required:            true,
			},
			"tenant_tags": tenantTagsAttribute(),
		},
	}
}

func openShiftMeteringConfigSchema() schema.Attribute {
	return kubernetesBasedMeteringConfigSchema("OpenShift")
}

// AKS

func aksPlatformSchema() schema.Attribute {
	attributes := kubernetesBasedPlatformAttributes("AKS", "https://myaks-dns.westeurope.azmk8s.io:443")
	attributes["replication"] = aksReplicationConfigSchema()
	attributes["metering"] = aksMeteringConfigSchema()
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Azure Kubernetes Service configuration",
		Optional:            true,
		Attributes:          attributes,
	}
}

func aksReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Replication configuration for AKS (optional, but required for replication)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"access_token": secretEmbeddedSchema("The Access Token of the service account for replicator access.", false),
			"namespace_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming namespaces in AKS",
				Required:            true,
			},
			"group_name_pattern": schema.StringAttribute{
				MarkdownDescription: "Pattern for naming groups in AKS",
				Required:            true,
			},
			"service_principal": schema.SingleNestedAttribute{
				MarkdownDescription: "Service principal configuration for AKS",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"entra_tenant": schema.StringAttribute{
						MarkdownDescription: "Domain name or ID of the Entra Tenant that holds the Service Principal.",
						Required:            true,
					},
					"object_id": schema.StringAttribute{
						MarkdownDescription: "The Object ID of the Enterprise Application. You can get this Object ID via the API (e.g. when using our Terraform provider) or from Enterprise applications pane in Microsoft Entra admin center.",
						Required:            true,
					},
					"client_id": schema.StringAttribute{
						MarkdownDescription: "The Application (Client) ID. In Azure Portal, this is the Application ID of the 'Enterprise Application' but can also be retrieved via the 'App Registration' object as 'Application (Client) ID'.",
						Required:            true,
					},
					"auth": azureAuthSchema(),
				},
			},
			"aks_subscription_id": schema.StringAttribute{
				MarkdownDescription: "Subscription ID for the AKS cluster",
				Required:            true,
			},
			"aks_cluster_name": schema.StringAttribute{
				MarkdownDescription: "Name of the AKS cluster.",
				Required:            true,
			},
			"aks_resource_group": schema.StringAttribute{
				MarkdownDescription: "Resource group for the AKS cluster",
				Required:            true,
			},
			"redirect_url": schema.StringAttribute{
				MarkdownDescription: "This is the URL that Azure's consent experience redirects users to after they accept their invitation.",
				Optional:            true,
			},
			"send_azure_invitation_mail": schema.BoolAttribute{
				MarkdownDescription: "Flag to send Azure invitation emails. When true, meshStack instructs Azure to send out Invitation mails to invited users.",
				Required:            true,
			},
			"user_lookup_strategy": azureUserLookupStrategySchema(),
			"administrative_unit_id": schema.StringAttribute{
				MarkdownDescription: "If you enter an administrative unit ID the replicated (and potentially existing) groups will be put into this AU. This can be used to limit the permission scopes which are required for the replicator principal. If you remove the AU ID again or change it, the groups will not be removed from the old AU.",
				Optional:            true,
			},
		},
	}
}

func aksMeteringConfigSchema() schema.Attribute {
	return kubernetesBasedMeteringConfigSchema("AKS")
}
