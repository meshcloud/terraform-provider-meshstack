package client

type GcpPlatformConfig struct {
	Replication *GcpReplicationConfig `json:"replication" tfsdk:"replication"`
}

type GcpReplicationConfig struct {
	ServiceAccountConfig              *GcpServiceAccountConfig `json:"serviceAccountConfig,omitempty" tfsdk:"service_account_config"`
	Domain                            *string                  `json:"domain,omitempty" tfsdk:"domain"`
	CustomerId                        *string                  `json:"customerId,omitempty" tfsdk:"customer_id"`
	GroupNamePattern                  *string                  `json:"groupNamePattern,omitempty" tfsdk:"group_name_pattern"`
	ProjectNamePattern                *string                  `json:"projectNamePattern,omitempty" tfsdk:"project_name_pattern"`
	ProjectIdPattern                  *string                  `json:"projectIdPattern,omitempty" tfsdk:"project_id_pattern"`
	BillingAccountId                  *string                  `json:"billingAccountId,omitempty" tfsdk:"billing_account_id"`
	UserLookupStrategy                *string                  `json:"userLookupStrategy,omitempty" tfsdk:"user_lookup_strategy"`
	GcpRoleMappings                   []GcpPlatformRoleMapping `json:"gcpRoleMappings,omitempty" tfsdk:"gcp_role_mappings"`
	AllowHierarchicalFolderAssignment *bool                    `json:"allowHierarchicalFolderAssignment,omitempty" tfsdk:"allow_hierarchical_folder_assignment"`
	TenantTags                        *GcpTenantTags           `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup    *bool                    `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
}

type GcpServiceAccountConfig struct {
	ServiceAccountCredentialsConfig      *GcpServiceAccountCredentialsConfig      `json:"serviceAccountCredentialsConfig,omitempty" tfsdk:"service_account_credentials_config"`
	ServiceAccountWorkloadIdentityConfig *GcpServiceAccountWorkloadIdentityConfig `json:"serviceAccountWorkloadIdentityConfig,omitempty" tfsdk:"service_account_workload_identity_config"`
}

type GcpServiceAccountCredentialsConfig struct {
	ServiceAccountCredentialsB64 *string `json:"serviceAccountCredentialsB64,omitempty" tfsdk:"service_account_credentials_b64"`
}

type GcpServiceAccountWorkloadIdentityConfig struct {
	Audience            *string `json:"audience,omitempty" tfsdk:"audience"`
	ServiceAccountEmail *string `json:"serviceAccountEmail,omitempty" tfsdk:"service_account_email"`
}

type GcpTenantTags struct {
	NamespacePrefix string         `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []GcpTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type GcpTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type GcpPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	GcpRole            string               `json:"gcpRole" tfsdk:"gcp_role"`
}
