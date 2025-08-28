package provider

type AKSPlatformConfigModel struct {
	BaseUrl              string                     `tfsdk:"base_url"`
	DisableSslValidation bool                       `tfsdk:"disable_ssl_validation"`
	Replication          *AKSReplicationConfigModel `tfsdk:"replication"`
}

type AKSReplicationConfigModel struct {
	AccessToken             string                         `tfsdk:"access_token"`
	NamespaceNamePattern    string                         `tfsdk:"namespace_name_pattern"`
	GroupNamePattern        string                         `tfsdk:"group_name_pattern"`
	ServicePrincipal        AKSServicePrincipalConfigModel `tfsdk:"service_principal"`
	AksSubscriptionId       string                         `tfsdk:"aks_subscription_id"`
	AksClusterName          string                         `tfsdk:"aks_cluster_name"`
	AksResourceGroup        string                         `tfsdk:"aks_resource_group"`
	RedirectUrl             *string                        `tfsdk:"redirect_url"`
	SendAzureInvitationMail bool                           `tfsdk:"send_azure_invitation_mail"`
	UserLookUpStrategy      string                         `tfsdk:"user_look_up_strategy"`
	AdministrativeUnitId    *string                        `tfsdk:"administrative_unit_id"`
}

type AKSServicePrincipalConfigModel struct {
	ClientId                    string  `tfsdk:"client_id"`
	AuthType                    string  `tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `tfsdk:"credentials_auth_client_secret"`
	EntraTenant                 string  `tfsdk:"entra_tenant"`
	ObjectId                    string  `tfsdk:"object_id"`
}
