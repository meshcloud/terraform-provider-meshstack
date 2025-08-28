package provider

type KubernetesPlatformConfig struct {
	BaseUrl              string                            `tfsdk:"base_url"`
	DisableSslValidation bool                              `tfsdk:"disable_ssl_validation"`
	Replication          *KubernetesReplicationConfigModel `tfsdk:"replication"`
}

type KubernetesReplicationConfigModel struct {
	ClientConfig         KubernetesClientConfigModel `tfsdk:"client_config"`
	NamespaceNamePattern string                      `tfsdk:"namespace_name_pattern"`
}

type KubernetesClientConfigModel struct {
	AccessToken string `tfsdk:"access_token"`
}
