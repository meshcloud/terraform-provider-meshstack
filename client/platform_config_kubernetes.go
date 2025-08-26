package client

type KubernetesPlatformConfig struct {
	BaseUrl              string                         `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                           `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *KubernetesReplicationConfig   `json:"replication,omitempty" tfsdk:"replication"`
}

type KubernetesReplicationConfig struct {
	ClientConfig           KubernetesClientConfig `json:"clientConfig" tfsdk:"client_config"`
	NamespaceNamePattern   string                 `json:"namespaceNamePattern" tfsdk:"namespace_name_pattern"`
}

type KubernetesClientConfig struct {
	AccessToken string `json:"accessToken" tfsdk:"access_token"`
}
