package client

type AksPlatformProperties struct {
	KubernetesRoleMappings []KubernetesRoleMapping `json:"kubernetesRoleMappings" tfsdk:"kubernetes_role_mappings"`
}

type KubernetesRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	PlatformRoles      []string             `json:"platformRoles" tfsdk:"platform_roles"`
}
