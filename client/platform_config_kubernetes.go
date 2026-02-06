package client

import "github.com/meshcloud/terraform-provider-meshstack/client/types"

type KubernetesPlatformConfig struct {
	BaseUrl              string                       `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                         `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *KubernetesReplicationConfig `json:"replication" tfsdk:"replication"`
	Metering             *KubernetesMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type KubernetesReplicationConfig struct {
	ClientConfig         KubernetesClientConfig `json:"clientConfig" tfsdk:"client_config"`
	NamespaceNamePattern string                 `json:"namespaceNamePattern" tfsdk:"namespace_name_pattern"`
}

type KubernetesClientConfig struct {
	AccessToken types.Secret `json:"accessToken" tfsdk:"access_token"`
}

type KubernetesMeteringConfig struct {
	ClientConfig KubernetesClientConfig               `json:"clientConfig" tfsdk:"client_config"`
	Processing   MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
