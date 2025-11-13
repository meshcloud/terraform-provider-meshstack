package client

type AzureRgPlatformConfig struct {
	EntraTenant *string                   `json:"entraTenant,omitempty" tfsdk:"entra_tenant"`
	Replication *AzureRgReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AzureRgReplicationConfig struct {
	ServicePrincipal                           *AzureServicePrincipalConfig `json:"servicePrincipal,omitempty" tfsdk:"service_principal"`
	Subscription                               *string                      `json:"subscription,omitempty" tfsdk:"subscription"`
	ResourceGroupNamePattern                   *string                      `json:"resourceGroupNamePattern,omitempty" tfsdk:"resource_group_name_pattern"`
	UserGroupNamePattern                       *string                      `json:"userGroupNamePattern,omitempty" tfsdk:"user_group_name_pattern"`
	B2bUserInvitation                          *AzureB2bUserInvitation      `json:"b2bUserInvitation,omitempty" tfsdk:"b2b_user_invitation"`
	UserLookUpStrategy                         *string                      `json:"userLookUpStrategy,omitempty" tfsdk:"user_look_up_strategy"`
	TenantTags                                 *AzureTenantTags             `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup             *bool                        `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId                       *string                      `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
	AllowHierarchicalManagementGroupAssignment *bool                        `json:"allowHierarchicalManagementGroupAssignment,omitempty" tfsdk:"allow_hierarchical_management_group_assignment"`
}
