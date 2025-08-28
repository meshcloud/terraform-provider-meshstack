package provider

type AzurePlatformConfigModel struct {
	EntraTenant string                       `tfsdk:"entra_tenant"`
	Replication *AzureReplicationConfigModel `tfsdk:"replication"`
}

type AzureReplicationConfigModel struct {
	ServicePrincipal               AzureServicePrincipalConfigModel          `tfsdk:"service_principal"`
	Provisioning                   *AzureSubscriptionProvisioningConfigModel `tfsdk:"provisioning"`
	B2bUserInvitation              *AzureInviteB2BUserConfigModel            `tfsdk:"b2b_user_invitation"`
	SubscriptionNamePattern        string                                    `tfsdk:"subscription_name_pattern"`
	GroupNamePattern               string                                    `tfsdk:"group_name_pattern"`
	BlueprintServicePrincipal      string                                    `tfsdk:"blueprint_service_principal"`
	BlueprintLocation              string                                    `tfsdk:"blueprint_location"`
	RoleMappings                   map[string]AzureRoleModel                 `tfsdk:"role_mappings"`
	TenantTags                     *MeshTagConfigModel                       `tfsdk:"tenant_tags"`
	UserLookUpStrategy             string                                    `tfsdk:"user_look_up_strategy"`
	SkipUserGroupPermissionCleanup bool                                      `tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId           *string                                   `tfsdk:"administrative_unit_id"`
}

type AzureRGPlatformConfigModel struct {
	EntraTenant string                         `tfsdk:"entra_tenant"`
	Replication *AzureRGReplicationConfigModel `tfsdk:"replication"`
}

type AzureRGReplicationConfigModel struct {
	ServicePrincipal               AzureServicePrincipalConfigModel `tfsdk:"service_principal"`
	Subscription                   string                           `tfsdk:"subscription"`
	ResourceGroupNamePattern       string                           `tfsdk:"resource_group_name_pattern"`
	UserGroupNamePattern           string                           `tfsdk:"user_group_name_pattern"`
	B2bUserInvitation              *AzureInviteB2BUserConfigModel   `tfsdk:"b2b_user_invitation"`
	UserLookUpStrategy             string                           `tfsdk:"user_look_up_strategy"`
	TenantTags                     *MeshTagConfigModel              `tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup bool                             `tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId           *string                          `tfsdk:"administrative_unit_id"`
}

type AzureServicePrincipalConfigModel struct {
	ClientId                    string                             `tfsdk:"client_id"`
	AuthType                    AzureServicePrincipalAuthTypeModel `tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string                            `tfsdk:"credentials_auth_client_secret"`
	ObjectId                    string                             `tfsdk:"object_id"`
}

type AzureServicePrincipalAuthTypeModel string

const (
	AzureServicePrincipalAuthTypeCredentials      AzureServicePrincipalAuthTypeModel = "CREDENTIALS"
	AzureServicePrincipalAuthTypeWorkloadIdentity AzureServicePrincipalAuthTypeModel = "WORKLOAD_IDENTITY"
)

type AzureSubscriptionProvisioningConfigModel struct {
	SubscriptionOwnerObjectIds []string                                    `tfsdk:"subscription_owner_object_ids"`
	EnterpriseEnrollment       *AzureEnterpriseEnrollmentConfigModel       `tfsdk:"enterprise_enrollment"`
	CustomerAgreement          *AzureCustomerAgreementConfigModel          `tfsdk:"customer_agreement"`
	PreProvisioned             *AzurePreProvisionedSubscriptionConfigModel `tfsdk:"pre_provisioned"`
}

type AzureEnterpriseEnrollmentConfigModel struct {
	EnrollmentAccountId                  string `tfsdk:"enrollment_account_id"`
	SubscriptionOfferType                string `tfsdk:"subscription_offer_type"`
	UseLegacySubscriptionEnrollment      bool   `tfsdk:"use_legacy_subscription_enrollment"`
	SubscriptionCreationErrorCooldownSec int64  `tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzureCustomerAgreementConfigModel struct {
	SourceServicePrincipal               AzureGraphApiCredentialsModel `tfsdk:"source_service_principal"`
	DestinationEntraId                   string                        `tfsdk:"destination_entra_id"`
	SourceEntraTenant                    string                        `tfsdk:"source_entra_tenant"`
	BillingScope                         string                        `tfsdk:"billing_scope"`
	SubscriptionCreationErrorCooldownSec int64                         `tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzurePreProvisionedSubscriptionConfigModel struct {
	UnusedSubscriptionNamePrefix string `tfsdk:"unused_subscription_name_prefix"`
}

type AzureGraphApiCredentialsModel struct {
	ClientId                    string                             `tfsdk:"client_id"`
	AuthType                    AzureServicePrincipalAuthTypeModel `tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string                            `tfsdk:"credentials_auth_client_secret"`
}

type AzureInviteB2BUserConfigModel struct {
	RedirectUrl             string `tfsdk:"redirect_url"`
	SendAzureInvitationMail bool   `tfsdk:"send_azure_invitation_mail"`
}

type AzureRoleModel struct {
	Alias string `tfsdk:"alias"`
	Id    string `tfsdk:"id"`
}
