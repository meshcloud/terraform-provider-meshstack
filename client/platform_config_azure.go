package client

type AzurePlatformConfig struct {
	EntraTenant string                  `json:"entraTenant" tfsdk:"entra_tenant"`
	Replication *AzureReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
	Metering    *AzureMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type AzureReplicationConfig struct {
	ServicePrincipal                           AzureServicePrincipalConfig          `json:"servicePrincipal" tfsdk:"service_principal"`
	Provisioning                               *AzureSubscriptionProvisioningConfig `json:"provisioning,omitempty" tfsdk:"provisioning"`
	B2bUserInvitation                          *AzureInviteB2BUserConfig            `json:"b2bUserInvitation,omitempty" tfsdk:"b2b_user_invitation"`
	SubscriptionNamePattern                    string                               `json:"subscriptionNamePattern" tfsdk:"subscription_name_pattern"`
	GroupNamePattern                           string                               `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	BlueprintServicePrincipal                  string                               `json:"blueprintServicePrincipal" tfsdk:"blueprint_service_principal"`
	BlueprintLocation                          string                               `json:"blueprintLocation" tfsdk:"blueprint_location"`
	AzureRoleMappings                          []AzureRoleMapping                   `json:"azureRoleMappings" tfsdk:"azure_role_mappings"`
	TenantTags                                 *MeshTenantTags                      `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	UserLookUpStrategy                         string                               `json:"userLookUpStrategy" tfsdk:"user_look_up_strategy"`
	SkipUserGroupPermissionCleanup             bool                                 `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId                       *string                              `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
	AllowHierarchicalManagementGroupAssignment bool                                 `json:"allowHierarchicalManagementGroupAssignment" tfsdk:"allow_hierarchical_management_group_assignment"`
}

type AzureServicePrincipalConfig struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
	ObjectId                    string  `json:"objectId" tfsdk:"object_id"`
}

type AzureGraphApiCredentials struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
}

type AzureSubscriptionProvisioningConfig struct {
	SubscriptionOwnerObjectIds []string                               `json:"subscriptionOwnerObjectIds" tfsdk:"subscription_owner_object_ids"`
	EnterpriseEnrollment       *AzureEnterpriseEnrollmentConfig       `json:"enterpriseEnrollment,omitempty" tfsdk:"enterprise_enrollment"`
	CustomerAgreement          *AzureCustomerAgreementConfig          `json:"customerAgreement,omitempty" tfsdk:"customer_agreement"`
	PreProvisioned             *AzurePreProvisionedSubscriptionConfig `json:"preProvisioned,omitempty" tfsdk:"pre_provisioned"`
}

type AzureEnterpriseEnrollmentConfig struct {
	EnrollmentAccountId                  string `json:"enrollmentAccountId" tfsdk:"enrollment_account_id"`
	SubscriptionOfferType                string `json:"subscriptionOfferType" tfsdk:"subscription_offer_type"`
	UseLegacySubscriptionEnrollment      bool   `json:"useLegacySubscriptionEnrollment" tfsdk:"use_legacy_subscription_enrollment"`
	SubscriptionCreationErrorCooldownSec int64  `json:"subscriptionCreationErrorCooldownSec" tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzureCustomerAgreementConfig struct {
	SourceServicePrincipal               AzureGraphApiCredentials `json:"sourceServicePrincipal" tfsdk:"source_service_principal"`
	DestinationEntraId                   string                   `json:"destinationEntraId" tfsdk:"destination_entra_id"`
	SourceEntraTenant                    string                   `json:"sourceEntraTenant" tfsdk:"source_entra_tenant"`
	BillingScope                         string                   `json:"billingScope" tfsdk:"billing_scope"`
	SubscriptionCreationErrorCooldownSec int64                    `json:"subscriptionCreationErrorCooldownSec" tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzurePreProvisionedSubscriptionConfig struct {
	UnusedSubscriptionNamePrefix string `json:"unusedSubscriptionNamePrefix" tfsdk:"unused_subscription_name_prefix"`
}

type AzureInviteB2BUserConfig struct {
	RedirectUrl             string `json:"redirectUrl" tfsdk:"redirect_url"`
	SendAzureInvitationMail bool   `json:"sendAzureInvitationMail" tfsdk:"send_azure_invitation_mail"`
}

type AzureRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AzureRole          AzureRole            `json:"azureRole" tfsdk:"azure_role"`
}

type AzureRole struct {
	Alias string `json:"alias" tfsdk:"alias"`
	Id    string `json:"id" tfsdk:"id"`
}

type AzureMeteringConfig struct {
	ServicePrincipal AzureServicePrincipalConfig          `json:"servicePrincipal" tfsdk:"service_principal"`
	Processing       MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
