package client

type OpenShiftPlatformConfig struct {
	BaseUrl              string                        `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                          `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *OpenShiftReplicationConfig   `json:"replication,omitempty" tfsdk:"replication"`
}

type OpenShiftReplicationConfig struct {
	ClientConfig                 KubernetesClientConfig `json:"clientConfig" tfsdk:"client_config"`
	WebConsoleUrl                *string                `json:"webConsoleUrl,omitempty" tfsdk:"web_console_url"`
	ProjectNamePattern           string                 `json:"projectNamePattern" tfsdk:"project_name_pattern"`
	EnableTemplateInstantiation  bool                   `json:"enableTemplateInstantiation" tfsdk:"enable_template_instantiation"`
	RoleMappings                 map[string]string      `json:"roleMappings" tfsdk:"role_mappings"`
	IdentityProviderName         string                 `json:"identityProviderName" tfsdk:"identity_provider_name"`
	TenantTags                   *MeshTagConfig         `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
}
