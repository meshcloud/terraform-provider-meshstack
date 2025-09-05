package provider

type OpenShiftPlatformConfigModel struct {
	BaseUrl              string                           `tfsdk:"base_url"`
	DisableSslValidation bool                             `tfsdk:"disable_ssl_validation"`
	Replication          *OpenShiftReplicationConfigModel `tfsdk:"replication"`
}

type OpenShiftReplicationConfigModel struct {
	ClientConfig                KubernetesClientConfigModel `tfsdk:"client_config"`
	WebConsoleUrl               *string                     `tfsdk:"web_console_url"`
	ProjectNamePattern          string                      `tfsdk:"project_name_pattern"`
	EnableTemplateInstantiation bool                        `tfsdk:"enable_template_instantiation"`
	RoleMappings                map[string]string           `tfsdk:"role_mappings"`
	IdentityProviderName        string                      `tfsdk:"identity_provider_name"`
	TenantTags                  *MeshTagConfigModel         `tfsdk:"tenant_tags"`
}
