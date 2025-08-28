package provider

type GCPPlatformConfigModel struct {
	Replication *GCPReplicationConfigModel `tfsdk:"replication"`
}

type GCPReplicationConfigModel struct {
	ServiceAccountConfig              GCPServiceAccountConfigModel `tfsdk:"service_account_config"`
	Domain                            string                       `tfsdk:"domain"`
	CustomerId                        string                       `tfsdk:"customer_id"`
	GroupNamePattern                  string                       `tfsdk:"group_name_pattern"`
	ProjectNamePattern                string                       `tfsdk:"project_name_pattern"`
	ProjectIdPattern                  string                       `tfsdk:"project_id_pattern"`
	BillingAccountId                  string                       `tfsdk:"billing_account_id"`
	UserLookupStrategy                string                       `tfsdk:"user_lookup_strategy"`
	UsedExternalIdType                *string                      `tfsdk:"used_external_id_type"`
	RoleMappings                      map[string]string            `tfsdk:"role_mappings"`
	AllowHierarchicalFolderAssignment bool                         `tfsdk:"allow_hierarchical_folder_assignment"`
	TenantTags                        *MeshTagConfigModel          `tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup    bool                         `tfsdk:"skip_user_group_permission_cleanup"`
}

type GCPServiceAccountConfigModel struct {
	ServiceAccountCredentialsConfig      *GCPServiceAccountCredentialsConfigModel      `tfsdk:"service_account_credentials_config"`
	ServiceAccountWorkloadIdentityConfig *GCPServiceAccountWorkloadIdentityConfigModel `tfsdk:"service_account_workload_identity_config"`
}

type GCPServiceAccountCredentialsConfigModel struct {
	ServiceAccountCredentialsB64 string `tfsdk:"service_account_credentials_b64"`
}

type GCPServiceAccountWorkloadIdentityConfigModel struct {
	Audience            string `tfsdk:"audience"`
	ServiceAccountEmail string `tfsdk:"service_account_email"`
}
