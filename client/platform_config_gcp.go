package client

import "github.com/meshcloud/terraform-provider-meshstack/client/types"

type GcpPlatformConfig struct {
	Replication *GcpReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
	Metering    *GcpMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type GcpReplicationConfig struct {
	ServiceAccount                    GcpServiceAccountConfig  `json:"serviceAccount" tfsdk:"service_account"`
	Domain                            string                   `json:"domain" tfsdk:"domain"`
	CustomerId                        string                   `json:"customerId" tfsdk:"customer_id"`
	GroupNamePattern                  string                   `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	ProjectNamePattern                string                   `json:"projectNamePattern" tfsdk:"project_name_pattern"`
	ProjectIdPattern                  string                   `json:"projectIdPattern" tfsdk:"project_id_pattern"`
	BillingAccountId                  string                   `json:"billingAccountId" tfsdk:"billing_account_id"`
	UserLookupStrategy                string                   `json:"userLookupStrategy" tfsdk:"user_lookup_strategy"`
	UsedExternalIdType                *string                  `json:"usedExternalIdType,omitempty" tfsdk:"used_external_id_type"`
	GcpRoleMappings                   []GcpPlatformRoleMapping `json:"gcpRoleMappings" tfsdk:"gcp_role_mappings"`
	AllowHierarchicalFolderAssignment bool                     `json:"allowHierarchicalFolderAssignment" tfsdk:"allow_hierarchical_folder_assignment"`
	TenantTags                        *MeshTenantTags          `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup    bool                     `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
}

type GcpServiceAccountConfig struct {
	Type             string                                   `json:"type" tfsdk:"type"`
	Credential       *types.Secret                            `json:"credential,omitempty" tfsdk:"credential"`
	WorkloadIdentity *GcpServiceAccountWorkloadIdentityConfig `json:"workloadIdentity,omitempty" tfsdk:"workload_identity"`
}

type GcpServiceAccountWorkloadIdentityConfig struct {
	Audience            string `json:"audience" tfsdk:"audience"`
	ServiceAccountEmail string `json:"serviceAccountEmail" tfsdk:"service_account_email"`
}

type GcpPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	GcpRole            string               `json:"gcpRole" tfsdk:"gcp_role"`
}

type GcpMeteringConfig struct {
	ServiceAccount                          GcpServiceAccountConfig              `json:"serviceAccount" tfsdk:"service_account"`
	BigqueryTable                           string                               `json:"bigqueryTable" tfsdk:"bigquery_table"`
	BigqueryTableForCarbonFootprint         *string                              `json:"bigqueryTableForCarbonFootprint,omitempty" tfsdk:"bigquery_table_for_carbon_footprint"`
	CarbonFootprintDataCollectionStartMonth *string                              `json:"carbonFootprintDataCollectionStartMonth,omitempty" tfsdk:"carbon_footprint_data_collection_start_month"`
	PartitionTimeColumn                     string                               `json:"partitionTimeColumn" tfsdk:"partition_time_column"`
	AdditionalFilter                        *string                              `json:"additionalFilter,omitempty" tfsdk:"additional_filter"`
	Processing                              MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
