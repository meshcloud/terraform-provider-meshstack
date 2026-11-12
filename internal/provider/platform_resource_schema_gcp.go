package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

func gcpPlatformSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Google Cloud Platform (GCP) platform configuration.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"replication": gcpReplicationConfigSchema(),
			"metering":    gcpMeteringConfigSchema(),
		},
	}
}

func gcpReplicationConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "GCP-specific replication configuration for the platform.",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_account":      gcpServiceAccountConfigSchema(),
			"domain":               requiredString("The domain used for cloud identity directory-groups created and managed by meshStack. meshStack maintains separate groups for each meshProject role on each managed GCP project."),
			"customer_id":          requiredString("A Google Customer ID. It typically starts with a 'C'."),
			"group_name_pattern":   requiredString("All the commonly available replicator string template properties are available. Additionally you can also use 'platformGroupAlias' as a placeholder to access the specific project role from the role mappings done in this platform configuration or in the meshLandingZone configuration."),
			"project_name_pattern": requiredString("All the commonly available replicator string template properties are available. The result must be 4 to 30 characters. Allowed characters are: lowercase and uppercase letters, numbers, hyphen, single-quote, double-quote, space, and exclamation point. When length restrictions are applied, the abbreviation will be in the middle and marked by a single-quote."),
			"project_id_pattern":   requiredString("All the commonly available replicator string template properties are available. The resulting string must not exceed a total length of 30 characters. Only alphanumeric + hyphen are allowed. We recommend that configuration include at least 3 characters of the random parameter to reduce the chance of naming collisions as the project Ids must be globally unique within GCP."),
			"billing_account_id":   requiredString("The ID of the billing account to associate with all GCP projects managed by meshStack"),
			"user_lookup_strategy": requiredString("Users can either be looked up by E-Mail or externalAccountId. This must also be the property that is placed in the external user id (EUID) of your meshUser entity to match. E-Mail is usually a good choice as this is often set up as the EUID throughout all cloud platforms and meshStack. ('email' or 'externalId')"),
			"used_external_id_type": schema.StringAttribute{
				MarkdownDescription: "Used external ID type for user lookup",
				Optional:            true,
			},
			"gcp_role_mappings": schema.SetNestedAttribute{
				MarkdownDescription: "Mapping of platform roles to GCP IAM roles.",
				Required:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"project_role_ref": meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.ProjectRole, Description: "Reference to the meshProjectRole.", InSet: true}),
						"gcp_role":         requiredString("The GCP IAM role"),
					},
				},
			},
			"allow_hierarchical_folder_assignment": requiredBool("Configuration flag to enable or disable hierarchical folder assignment in GCP. If set to true: Projects can be moved to sub folders of the folder defined in the Landing Zone. This is useful if you want to manage the project location with a deeper and more granular hierarchy. If set to false: Projects will always be moved directly to the folder defined in the Landing Zone."),
			"tenant_tags":                          tenantTagsAttribute(),
			"skip_user_group_permission_cleanup":   requiredBool("For certain use cases you might want to preserve user groups and replicated permission after a tenant was deleted on the GCP platform. Checking this option preserves those permissions. Please keep in mind that the platform operator is then responsible for cleaning them up later."),
		},
	}
}

func gcpServiceAccountConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Service account configuration. Exactly one of credential or workload_identity must be provided.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"type": schema.StringAttribute{
				MarkdownDescription: "Service account type",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{authTypeDefault()},
			},
			"credential": secret.ResourceSchema(secret.ResourceSchemaOptions{
				MarkdownDescription: "Base64 encoded credentials.json file for a GCP ServiceAccount.",
				Optional:            true,
			}),
			"workload_identity": schema.SingleNestedAttribute{
				MarkdownDescription: "Workload identity configuration.",
				Optional:            true,
				Validators: []validator.Object{
					// This will result in an unclear path in the error message for credential because
					// it traverses up to the parent and back down.
					// see https://github.com/hashicorp/terraform-plugin-framework-validators/issues/274
					objectvalidator.ExactlyOneOf(
						path.MatchRelative(),
						path.MatchRelative().AtParent().AtName("credential"),
					),
				},
				Attributes: map[string]schema.Attribute{
					"audience":              requiredString("The audience associated with your workload identity pool provider."),
					"service_account_email": requiredString("The email address of the Service Account, that gets impersonated for calling Google APIs via Workload Identity Federation."),
				},
			},
		},
	}
}

func gcpMeteringConfigSchema() schema.Attribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Metering configuration for GCP (optional, but required for metering)",
		Optional:            true,
		Attributes: map[string]schema.Attribute{
			"service_account": gcpServiceAccountConfigSchema(),
			"bigquery_table":  requiredString("BigQuery table for metering data."),
			"bigquery_table_for_carbon_footprint": schema.StringAttribute{
				MarkdownDescription: "BigQuery table for carbon footprint data.",
				Optional:            true,
			},
			"carbon_footprint_data_collection_start_month": schema.StringAttribute{
				MarkdownDescription: "Start month for carbon footprint data collection.",
				Optional:            true,
			},
			"partition_time_column": requiredString("Partition time column for BigQuery table."),
			"additional_filter": schema.StringAttribute{
				MarkdownDescription: "Additional filter for metering data.",
				Optional:            true,
			},
			"processing": meteringProcessingConfigSchema(),
		},
	}
}
