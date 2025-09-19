package client

type GcpPlatformProperties struct {
	GcpCloudFunctionUrl *string          `json:"gcpCloudFunctionUrl,omitempty" tfsdk:"gcp_cloud_function_url"`
	GcpFolderId         *string          `json:"gcpFolderId,omitempty" tfsdk:"gcp_folder_id"`
	GcpRoleMappings     []GcpRoleMapping `json:"gcpRoleMappings" tfsdk:"gcp_role_mappings"`
}

type GcpRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	PlatformRoles      []string             `json:"platformRoles" tfsdk:"platform_roles"`
}
