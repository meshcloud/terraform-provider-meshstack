package client

type AksPlatformConfig struct {
	BaseUrl              string                `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                  `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *AksReplicationConfig `json:"replication" tfsdk:"replication"`
	Metering             *AksMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type AksReplicationConfig struct {
	AccessToken             string                    `json:"accessToken" tfsdk:"access_token"`
	NamespaceNamePattern    string                    `json:"namespaceNamePattern" tfsdk:"namespace_name_pattern"`
	GroupNamePattern        string                    `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	ServicePrincipal        AksServicePrincipalConfig `json:"servicePrincipal" tfsdk:"service_principal"`
	AksSubscriptionId       string                    `json:"aksSubscriptionId" tfsdk:"aks_subscription_id"`
	AksClusterName          string                    `json:"aksClusterName" tfsdk:"aks_cluster_name"`
	AksResourceGroup        string                    `json:"aksResourceGroup" tfsdk:"aks_resource_group"`
	RedirectUrl             *string                   `json:"redirectUrl,omitempty" tfsdk:"redirect_url"`
	SendAzureInvitationMail bool                      `json:"sendAzureInvitationMail" tfsdk:"send_azure_invitation_mail"`
	UserLookUpStrategy      string                    `json:"userLookUpStrategy" tfsdk:"user_look_up_strategy"`
	AdministrativeUnitId    *string                   `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
}

type AksServicePrincipalConfig struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
	EntraTenant                 string  `json:"entraTenant" tfsdk:"entra_tenant"`
	ObjectId                    string  `json:"objectId" tfsdk:"object_id"`
}

type AksMeteringConfig struct {
	ClientConfig KubernetesClientConfig               `json:"clientConfig" tfsdk:"client_config"`
	Processing   MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
