package client

type GCPPlatformConfig struct {
	Replication *GCPReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type GCPReplicationConfig struct {
	ServiceAccountConfig              GCPServiceAccountConfig   `json:"serviceAccountConfig" tfsdk:"service_account_config"`
	Domain                            string                    `json:"domain" tfsdk:"domain"`
	CustomerId                        string                    `json:"customerId" tfsdk:"customer_id"`
	GroupNamePattern                  string                    `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	ProjectNamePattern                string                    `json:"projectNamePattern" tfsdk:"project_name_pattern"`
	ProjectIdPattern                  string                    `json:"projectIdPattern" tfsdk:"project_id_pattern"`
	BillingAccountId                  string                    `json:"billingAccountId" tfsdk:"billing_account_id"`
	UserLookupStrategy                string                    `json:"userLookupStrategy" tfsdk:"user_lookup_strategy"`
	UsedExternalIdType                *string                   `json:"usedExternalIdType,omitempty" tfsdk:"used_external_id_type"`
	RoleMappings                      map[string]string         `json:"roleMappings" tfsdk:"role_mappings"`
	AllowHierarchicalFolderAssignment bool                      `json:"allowHierarchicalFolderAssignment" tfsdk:"allow_hierarchical_folder_assignment"`
	TenantTags                        *MeshTagConfig            `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup    bool                      `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
}

type GCPServiceAccountConfig struct {
	ServiceAccountCredentialsConfig    *GCPServiceAccountCredentialsConfig    `json:"serviceAccountCredentialsConfig,omitempty" tfsdk:"service_account_credentials_config"`
	ServiceAccountWorkloadIdentityConfig *GCPServiceAccountWorkloadIdentityConfig `json:"serviceAccountWorkloadIdentityConfig,omitempty" tfsdk:"service_account_workload_identity_config"`
}

type GCPServiceAccountCredentialsConfig struct {
	ServiceAccountCredentialsB64 string `json:"serviceAccountCredentialsB64" tfsdk:"service_account_credentials_b64"`
}

type GCPServiceAccountWorkloadIdentityConfig struct {
	Audience            string `json:"audience" tfsdk:"audience"`
	ServiceAccountEmail string `json:"serviceAccountEmail" tfsdk:"service_account_email"`
}
