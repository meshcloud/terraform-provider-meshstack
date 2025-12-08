package client

type OpenShiftPlatformConfig struct {
	BaseUrl              string                      `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                        `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *OpenShiftReplicationConfig `json:"replication" tfsdk:"replication"`
	Metering             *OpenShiftMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type OpenShiftReplicationConfig struct {
	ClientConfig                KubernetesClientConfig         `json:"clientConfig" tfsdk:"client_config"`
	WebConsoleUrl               *string                        `json:"webConsoleUrl,omitempty" tfsdk:"web_console_url"`
	ProjectNamePattern          string                         `json:"projectNamePattern" tfsdk:"project_name_pattern"`
	EnableTemplateInstantiation bool                           `json:"enableTemplateInstantiation" tfsdk:"enable_template_instantiation"`
	OpenshiftRoleMappings       []OpenShiftPlatformRoleMapping `json:"openshiftRoleMappings" tfsdk:"openshift_role_mappings"`
	IdentityProviderName        string                         `json:"identityProviderName" tfsdk:"identity_provider_name"`
	TenantTags                  *MeshTenantTags                `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
}

type OpenShiftMeteringConfig struct {
	ClientConfig KubernetesClientConfig               `json:"clientConfig" tfsdk:"client_config"`
	Processing   MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}

type OpenShiftPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	OpenshiftRole      string               `json:"openshiftRole" tfsdk:"openshift_role"`
}
