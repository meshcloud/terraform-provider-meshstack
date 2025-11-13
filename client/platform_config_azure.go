package client

type AzurePlatformConfig struct {
	EntraTenant *string                 `json:"entraTenant,omitempty" tfsdk:"entra_tenant"`
	Replication *AzureReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AzureReplicationConfig struct {
	ServicePrincipal                           *AzureServicePrincipalConfig `json:"servicePrincipal,omitempty" tfsdk:"service_principal"`
	Provisioning                               *AzureProvisioning           `json:"provisioning,omitempty" tfsdk:"provisioning"`
	B2bUserInvitation                          *AzureB2bUserInvitation      `json:"b2bUserInvitation,omitempty" tfsdk:"b2b_user_invitation"`
	SubscriptionNamePattern                    *string                      `json:"subscriptionNamePattern,omitempty" tfsdk:"subscription_name_pattern"`
	GroupNamePattern                           *string                      `json:"groupNamePattern,omitempty" tfsdk:"group_name_pattern"`
	BlueprintServicePrincipal                  *string                      `json:"blueprintServicePrincipal,omitempty" tfsdk:"blueprint_service_principal"`
	BlueprintLocation                          *string                      `json:"blueprintLocation,omitempty" tfsdk:"blueprint_location"`
	AzureRoleMappings                          []AzurePlatformRoleMapping   `json:"azureRoleMappings,omitempty" tfsdk:"azure_role_mappings"`
	TenantTags                                 *AzureTenantTags             `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	UserLookUpStrategy                         *string                      `json:"userLookUpStrategy,omitempty" tfsdk:"user_look_up_strategy"`
	SkipUserGroupPermissionCleanup             *bool                        `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId                       *string                      `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
	AllowHierarchicalManagementGroupAssignment *bool                        `json:"allowHierarchicalManagementGroupAssignment,omitempty" tfsdk:"allow_hierarchical_management_group_assignment"`
}

type AzureServicePrincipalConfig struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
	ObjectId                    string  `json:"objectId" tfsdk:"object_id"`
}

type AzureSourceServicePrincipalConfig struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
}

type AzureProvisioning struct {
	SubscriptionOwnerObjectIds []string                   `json:"subscriptionOwnerObjectIds,omitempty" tfsdk:"subscription_owner_object_ids"`
	EnterpriseEnrollment       *AzureEnterpriseEnrollment `json:"enterpriseEnrollment,omitempty" tfsdk:"enterprise_enrollment"`
	CustomerAgreement          *AzureCustomerAgreement    `json:"customerAgreement,omitempty" tfsdk:"customer_agreement"`
	PreProvisioned             *AzurePreProvisioned       `json:"preProvisioned,omitempty" tfsdk:"pre_provisioned"`
}

type AzureEnterpriseEnrollment struct {
	EnrollmentAccountId                  string `json:"enrollmentAccountId" tfsdk:"enrollment_account_id"`
	SubscriptionOfferType                string `json:"subscriptionOfferType" tfsdk:"subscription_offer_type"`
	UseLegacySubscriptionEnrollment      *bool  `json:"useLegacySubscriptionEnrollment,omitempty" tfsdk:"use_legacy_subscription_enrollment"`
	SubscriptionCreationErrorCooldownSec *int   `json:"subscriptionCreationErrorCooldownSec,omitempty" tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzureCustomerAgreement struct {
	SourceServicePrincipal               *AzureSourceServicePrincipalConfig `json:"sourceServicePrincipal,omitempty" tfsdk:"source_service_principal"`
	DestinationEntraId                   string                             `json:"destinationEntraId" tfsdk:"destination_entra_id"`
	SourceEntraTenant                    string                             `json:"sourceEntraTenant" tfsdk:"source_entra_tenant"`
	BillingScope                         string                             `json:"billingScope" tfsdk:"billing_scope"`
	SubscriptionCreationErrorCooldownSec *int                               `json:"subscriptionCreationErrorCooldownSec,omitempty" tfsdk:"subscription_creation_error_cooldown_sec"`
}

type AzurePreProvisioned struct {
	UnusedSubscriptionNamePrefix string `json:"unusedSubscriptionNamePrefix" tfsdk:"unused_subscription_name_prefix"`
}

type AzureB2bUserInvitation struct {
	RedirectUrl             *string `json:"redirectUrl,omitempty" tfsdk:"redirect_url"`
	SendAzureInvitationMail *bool   `json:"sendAzureInvitationMail,omitempty" tfsdk:"send_azure_invitation_mail"`
}

type AzurePlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2        `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AzureRole          AzurePlatformRoleDefinition `json:"azureRole" tfsdk:"azure_role"`
}

type AzurePlatformRoleDefinition struct {
	Alias string `json:"alias" tfsdk:"alias"`
	Id    string `json:"id" tfsdk:"id"`
}

type AzureTenantTags struct {
	NamespacePrefix string           `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []AzureTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type AzureTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}
