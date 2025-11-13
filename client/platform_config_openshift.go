package client

type OpenShiftPlatformConfig struct {
	BaseUrl              string                      `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                        `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *OpenShiftReplicationConfig `json:"replication" tfsdk:"replication"`
	Metering             *OpenShiftMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type OpenShiftReplicationConfig struct {
	ClientConfig                *OpenShiftClientConfig         `json:"clientConfig,omitempty" tfsdk:"client_config"`
	WebConsoleUrl               *string                        `json:"webConsoleUrl,omitempty" tfsdk:"web_console_url"`
	ProjectNamePattern          *string                        `json:"projectNamePattern,omitempty" tfsdk:"project_name_pattern"`
	EnableTemplateInstantiation *bool                          `json:"enableTemplateInstantiation,omitempty" tfsdk:"enable_template_instantiation"`
	OpenShiftRoleMappings       []OpenShiftPlatformRoleMapping `json:"openshiftRoleMappings,omitempty" tfsdk:"openshift_role_mappings"`
	IdentityProviderName        *string                        `json:"identityProviderName,omitempty" tfsdk:"identity_provider_name"`
	TenantTags                  *OpenShiftTenantTags           `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
}

type OpenShiftClientConfig struct {
	AccessToken *string `json:"accessToken,omitempty" tfsdk:"access_token"`
}

type OpenShiftMeteringConfig struct {
	ClientConfig *OpenShiftClientConfig                `json:"clientConfig,omitempty" tfsdk:"client_config"`
	Processing   *MeshPlatformMeteringProcessingConfig `json:"processing,omitempty" tfsdk:"processing"`
}

type OpenShiftTenantTags struct {
	NamespacePrefix string               `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []OpenShiftTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type OpenShiftTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type OpenShiftPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	OpenShiftRole      string               `json:"openshiftRole" tfsdk:"openshift_role"`
}
