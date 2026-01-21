package client

import "github.com/meshcloud/terraform-provider-meshstack/client/types"

type AksPlatformConfig struct {
	BaseUrl              string                `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                  `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *AksReplicationConfig `json:"replication" tfsdk:"replication"`
	Metering             *AksMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type AksReplicationConfig struct {
	AccessToken             types.Secret              `json:"accessToken" tfsdk:"access_token"`
	NamespaceNamePattern    string                    `json:"namespaceNamePattern" tfsdk:"namespace_name_pattern"`
	GroupNamePattern        string                    `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	ServicePrincipal        AksServicePrincipalConfig `json:"servicePrincipal" tfsdk:"service_principal"`
	AksSubscriptionId       string                    `json:"aksSubscriptionId" tfsdk:"aks_subscription_id"`
	AksClusterName          string                    `json:"aksClusterName" tfsdk:"aks_cluster_name"`
	AksResourceGroup        string                    `json:"aksResourceGroup" tfsdk:"aks_resource_group"`
	RedirectUrl             *string                   `json:"redirectUrl,omitempty" tfsdk:"redirect_url"`
	SendAzureInvitationMail bool                      `json:"sendAzureInvitationMail" tfsdk:"send_azure_invitation_mail"`
	UserLookupStrategy      string                    `json:"userLookUpStrategy" tfsdk:"user_lookup_strategy"`
	AdministrativeUnitId    *string                   `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
}

type AksServicePrincipalConfig struct {
	EntraTenant string          `json:"entraTenant" tfsdk:"entra_tenant"`
	ObjectId    string          `json:"objectId" tfsdk:"object_id"`
	ClientId    string          `json:"clientId" tfsdk:"client_id"`
	Auth        AzureAuthConfig `json:"auth" tfsdk:"auth"`
}

type AksMeteringConfig struct {
	ClientConfig KubernetesClientConfig               `json:"clientConfig" tfsdk:"client_config"`
	Processing   MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
