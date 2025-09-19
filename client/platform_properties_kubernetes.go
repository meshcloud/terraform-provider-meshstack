package client

type KubernetesPlatformProperties struct {
	KubernetesRoleMappings []KubernetesRoleMapping `json:"kubernetesRoleMappings" tfsdk:"kubernetes_role_mappings"`
}
