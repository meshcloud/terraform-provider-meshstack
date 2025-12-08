package client

type AzureRgPlatformConfig struct {
	EntraTenant string                    `json:"entraTenant" tfsdk:"entra_tenant"`
	Replication *AzureRgReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AzureRgReplicationConfig struct {
	ServicePrincipal                           AzureServicePrincipalConfig `json:"servicePrincipal" tfsdk:"service_principal"`
	Subscription                               string                      `json:"subscription" tfsdk:"subscription"`
	ResourceGroupNamePattern                   string                      `json:"resourceGroupNamePattern" tfsdk:"resource_group_name_pattern"`
	UserGroupNamePattern                       string                      `json:"userGroupNamePattern" tfsdk:"user_group_name_pattern"`
	B2bUserInvitation                          *AzureInviteB2BUserConfig   `json:"b2bUserInvitation,omitempty" tfsdk:"b2b_user_invitation"`
	UserLookUpStrategy                         string                      `json:"userLookUpStrategy" tfsdk:"user_lookup_strategy"`
	TenantTags                                 *MeshTenantTags             `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup             bool                        `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId                       *string                     `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
	AllowHierarchicalManagementGroupAssignment bool                        `json:"allowHierarchicalManagementGroupAssignment" tfsdk:"allow_hierarchical_management_group_assignment"`
}
