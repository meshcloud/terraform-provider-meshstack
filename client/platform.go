package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_PLATFORM = "application/vnd.meshcloud.api.meshplatform.v2-preview.hal+json"

type MeshPlatform struct {
	ApiVersion string               `json:"apiVersion" tfsdk:"api_version"`
	Kind       string               `json:"kind" tfsdk:"kind"`
	Metadata   MeshPlatformMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec     `json:"spec" tfsdk:"spec"`
}

type MeshPlatformMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string  `json:"uuid" tfsdk:"uuid"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshPlatformSpec struct {
	DisplayName            string               `json:"displayName" tfsdk:"display_name"`
	Description            string               `json:"description" tfsdk:"description"`
	Endpoint               string               `json:"endpoint" tfsdk:"endpoint"`
	SupportUrl             *string              `json:"supportUrl,omitempty" tfsdk:"support_url"`
	DocumentationUrl       *string              `json:"documentationUrl,omitempty" tfsdk:"documentation_url"`
	LocationRef            LocationRef          `json:"locationRef" tfsdk:"location_ref"`
	ContributingWorkspaces []string             `json:"contributingWorkspaces" tfsdk:"contributing_workspaces"`
	Availability           PlatformAvailability `json:"availability" tfsdk:"availability"`
	Config                 PlatformConfig       `json:"config" tfsdk:"config"`
	QuotaDefinitions       []QuotaDefinition    `json:"quotaDefinitions" tfsdk:"quota_definitions"`
}

type QuotaDefinition struct {
	QuotaKey              string `json:"quotaKey" tfsdk:"quota_key"`
	MinValue              int    `json:"minValue" tfsdk:"min_value"`
	MaxValue              int    `json:"maxValue" tfsdk:"max_value"`
	Unit                  string `json:"unit" tfsdk:"unit"`
	AutoApprovalThreshold int    `json:"autoApprovalThreshold" tfsdk:"auto_approval_threshold"`
	Description           string `json:"description" tfsdk:"description"`
	Label                 string `json:"label" tfsdk:"label"`
}

type LocationRef struct {
	Kind string `json:"kind" tfsdk:"kind"`
	Name string `json:"name" tfsdk:"name"`
}

type PlatformAvailability struct {
	Restriction            string   `json:"restriction" tfsdk:"restriction"`
	PublicationState       string   `json:"publicationState" tfsdk:"publication_state"`
	RestrictedToWorkspaces []string `json:"restrictedToWorkspaces,omitempty" tfsdk:"restricted_to_workspaces"`
}

type PlatformConfig struct {
	Type       string                    `json:"type" tfsdk:"type"`
	Aws        *AwsPlatformConfig        `json:"aws,omitempty" tfsdk:"aws"`
	Aks        *AksPlatformConfig        `json:"aks,omitempty" tfsdk:"aks"`
	Azure      *AzurePlatformConfig      `json:"azure,omitempty" tfsdk:"azure"`
	AzureRg    *AzureRgPlatformConfig    `json:"azurerg,omitempty" tfsdk:"azurerg"`
	Gcp        *GcpPlatformConfig        `json:"gcp,omitempty" tfsdk:"gcp"`
	Kubernetes *KubernetesPlatformConfig `json:"kubernetes,omitempty" tfsdk:"kubernetes"`
	OpenShift  *OpenShiftPlatformConfig  `json:"openshift,omitempty" tfsdk:"openshift"`
}

type AwsPlatformConfig struct {
	Region      *string               `json:"region,omitempty" tfsdk:"region"`
	Replication *AwsReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AksPlatformConfig struct {
	BaseUrl              string                `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                  `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *AksReplicationConfig `json:"replication" tfsdk:"replication"`
}

type AzurePlatformConfig struct {
	EntraTenant *string                 `json:"entraTenant,omitempty" tfsdk:"entra_tenant"`
	Replication *AzureReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AzureRgPlatformConfig struct {
	EntraTenant *string                   `json:"entraTenant,omitempty" tfsdk:"entra_tenant"`
	Replication *AzureRgReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type GcpPlatformConfig struct {
	Replication *GcpReplicationConfig `json:"replication" tfsdk:"replication"`
}

type KubernetesPlatformConfig struct {
	BaseUrl              string                       `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                         `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *KubernetesReplicationConfig `json:"replication" tfsdk:"replication"`
}

type OpenShiftPlatformConfig struct {
	BaseUrl              string                      `json:"baseUrl" tfsdk:"base_url"`
	DisableSslValidation bool                        `json:"disableSslValidation" tfsdk:"disable_ssl_validation"`
	Replication          *OpenShiftReplicationConfig `json:"replication" tfsdk:"replication"`
}

type AzureRgReplicationConfig struct {
	ServicePrincipal                           *AzureServicePrincipalConfig `json:"servicePrincipal,omitempty" tfsdk:"service_principal"`
	Subscription                               *string                      `json:"subscription,omitempty" tfsdk:"subscription"`
	ResourceGroupNamePattern                   *string                      `json:"resourceGroupNamePattern,omitempty" tfsdk:"resource_group_name_pattern"`
	UserGroupNamePattern                       *string                      `json:"userGroupNamePattern,omitempty" tfsdk:"user_group_name_pattern"`
	B2bUserInvitation                          *AzureB2bUserInvitation      `json:"b2bUserInvitation,omitempty" tfsdk:"b2b_user_invitation"`
	UserLookUpStrategy                         *string                      `json:"userLookUpStrategy,omitempty" tfsdk:"user_look_up_strategy"`
	TenantTags                                 *AzureTenantTags             `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup             *bool                        `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
	AdministrativeUnitId                       *string                      `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
	AllowHierarchicalManagementGroupAssignment *bool                        `json:"allowHierarchicalManagementGroupAssignment,omitempty" tfsdk:"allow_hierarchical_management_group_assignment"`
}

type GcpReplicationConfig struct {
	ServiceAccountConfig              *GcpServiceAccountConfig `json:"serviceAccountConfig,omitempty" tfsdk:"service_account_config"`
	Domain                            *string                  `json:"domain,omitempty" tfsdk:"domain"`
	CustomerId                        *string                  `json:"customerId,omitempty" tfsdk:"customer_id"`
	GroupNamePattern                  *string                  `json:"groupNamePattern,omitempty" tfsdk:"group_name_pattern"`
	ProjectNamePattern                *string                  `json:"projectNamePattern,omitempty" tfsdk:"project_name_pattern"`
	ProjectIdPattern                  *string                  `json:"projectIdPattern,omitempty" tfsdk:"project_id_pattern"`
	BillingAccountId                  *string                  `json:"billingAccountId,omitempty" tfsdk:"billing_account_id"`
	UserLookupStrategy                *string                  `json:"userLookupStrategy,omitempty" tfsdk:"user_lookup_strategy"`
	GcpRoleMappings                   []GcpPlatformRoleMapping `json:"gcpRoleMappings,omitempty" tfsdk:"gcp_role_mappings"`
	AllowHierarchicalFolderAssignment *bool                    `json:"allowHierarchicalFolderAssignment,omitempty" tfsdk:"allow_hierarchical_folder_assignment"`
	TenantTags                        *GcpTenantTags           `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	SkipUserGroupPermissionCleanup    *bool                    `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
}

type GcpServiceAccountConfig struct {
	ServiceAccountCredentialsConfig      *GcpServiceAccountCredentialsConfig      `json:"serviceAccountCredentialsConfig,omitempty" tfsdk:"service_account_credentials_config"`
	ServiceAccountWorkloadIdentityConfig *GcpServiceAccountWorkloadIdentityConfig `json:"serviceAccountWorkloadIdentityConfig,omitempty" tfsdk:"service_account_workload_identity_config"`
}

type GcpServiceAccountCredentialsConfig struct {
	ServiceAccountCredentialsB64 *string `json:"serviceAccountCredentialsB64,omitempty" tfsdk:"service_account_credentials_b64"`
}

type GcpServiceAccountWorkloadIdentityConfig struct {
	Audience            *string `json:"audience,omitempty" tfsdk:"audience"`
	ServiceAccountEmail *string `json:"serviceAccountEmail,omitempty" tfsdk:"service_account_email"`
}

type GcpTenantTags struct {
	NamespacePrefix string         `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []GcpTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type GcpTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type KubernetesReplicationConfig struct {
	ClientConfig         *KubernetesClientConfig `json:"clientConfig,omitempty" tfsdk:"client_config"`
	NamespaceNamePattern *string                 `json:"namespaceNamePattern,omitempty" tfsdk:"namespace_name_pattern"`
}

type KubernetesClientConfig struct {
	AccessToken *string `json:"accessToken,omitempty" tfsdk:"access_token"`
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

type OpenShiftTenantTags struct {
	NamespacePrefix string               `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []OpenShiftTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type OpenShiftTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type AksReplicationConfig struct {
	AccessToken             *string                 `json:"accessToken,omitempty" tfsdk:"access_token"`
	NamespaceNamePattern    *string                 `json:"namespaceNamePattern,omitempty" tfsdk:"namespace_name_pattern"`
	GroupNamePattern        *string                 `json:"groupNamePattern,omitempty" tfsdk:"group_name_pattern"`
	ServicePrincipal        *ServicePrincipalConfig `json:"servicePrincipal,omitempty" tfsdk:"service_principal"`
	AksSubscriptionId       *string                 `json:"aksSubscriptionId,omitempty" tfsdk:"aks_subscription_id"`
	AksClusterName          *string                 `json:"aksClusterName,omitempty" tfsdk:"aks_cluster_name"`
	AksResourceGroup        *string                 `json:"aksResourceGroup,omitempty" tfsdk:"aks_resource_group"`
	RedirectUrl             *string                 `json:"redirectUrl,omitempty" tfsdk:"redirect_url"`
	SendAzureInvitationMail *bool                   `json:"sendAzureInvitationMail,omitempty" tfsdk:"send_azure_invitation_mail"`
	UserLookUpStrategy      *string                 `json:"userLookUpStrategy,omitempty" tfsdk:"user_look_up_strategy"`
	AdministrativeUnitId    *string                 `json:"administrativeUnitId,omitempty" tfsdk:"administrative_unit_id"`
}

type ServicePrincipalConfig struct {
	ClientId                    string  `json:"clientId" tfsdk:"client_id"`
	AuthType                    string  `json:"authType" tfsdk:"auth_type"`
	CredentialsAuthClientSecret *string `json:"credentialsAuthClientSecret,omitempty" tfsdk:"credentials_auth_client_secret"`
	EntraTenant                 string  `json:"entraTenant" tfsdk:"entra_tenant"`
	ObjectId                    string  `json:"objectId" tfsdk:"object_id"`
}

// Azure-specific service principal configurations
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

// AWS-specific replication configuration structures
type AwsReplicationConfig struct {
	AccessConfig                                  *AwsAccessConfig            `json:"accessConfig,omitempty" tfsdk:"access_config"`
	WaitForExternalAvm                            *bool                       `json:"waitForExternalAvm,omitempty" tfsdk:"wait_for_external_avm"`
	AutomationAccountRole                         *string                     `json:"automationAccountRole,omitempty" tfsdk:"automation_account_role"`
	AutomationAccountExternalId                   *string                     `json:"automationAccountExternalId,omitempty" tfsdk:"automation_account_external_id"`
	AccountAccessRole                             *string                     `json:"accountAccessRole,omitempty" tfsdk:"account_access_role"`
	AccountAliasPattern                           *string                     `json:"accountAliasPattern,omitempty" tfsdk:"account_alias_pattern"`
	EnforceAccountAlias                           *bool                       `json:"enforceAccountAlias,omitempty" tfsdk:"enforce_account_alias"`
	AccountEmailPattern                           *string                     `json:"accountEmailPattern,omitempty" tfsdk:"account_email_pattern"`
	TenantTags                                    *AwsTenantTags              `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	AwsSso                                        *AwsSsoConfig               `json:"awsSso,omitempty" tfsdk:"aws_sso"`
	EnrollmentConfiguration                       *AwsEnrollmentConfiguration `json:"enrollmentConfiguration,omitempty" tfsdk:"enrollment_configuration"`
	SelfDowngradeAccessRole                       *bool                       `json:"selfDowngradeAccessRole,omitempty" tfsdk:"self_downgrade_access_role"`
	SkipUserGroupPermissionCleanup                *bool                       `json:"skipUserGroupPermissionCleanup,omitempty" tfsdk:"skip_user_group_permission_cleanup"`
	AllowHierarchicalOrganizationalUnitAssignment *bool                       `json:"allowHierarchicalOrganizationalUnitAssignment,omitempty" tfsdk:"allow_hierarchical_organizational_unit_assignment"`
}

type AwsAccessConfig struct {
	OrganizationRootAccountRole       string                     `json:"organizationRootAccountRole" tfsdk:"organization_root_account_role"`
	OrganizationRootAccountExternalId *string                    `json:"organizationRootAccountExternalId,omitempty" tfsdk:"organization_root_account_external_id"`
	ServiceUserConfig                 *AwsServiceUserConfig      `json:"serviceUserConfig,omitempty" tfsdk:"service_user_config"`
	WorkloadIdentityConfig            *AwsWorkloadIdentityConfig `json:"workloadIdentityConfig,omitempty" tfsdk:"workload_identity_config"`
}

type AwsServiceUserConfig struct {
	AccessKey string  `json:"accessKey" tfsdk:"access_key"`
	SecretKey *string `json:"secretKey,omitempty" tfsdk:"secret_key"`
}

type AwsWorkloadIdentityConfig struct {
	RoleArn string `json:"roleArn" tfsdk:"role_arn"`
}

type AwsTenantTags struct {
	NamespacePrefix string         `json:"namespacePrefix" tfsdk:"namespace_prefix"`
	TagMappers      []AwsTagMapper `json:"tagMappers" tfsdk:"tag_mappers"`
}

type AwsTagMapper struct {
	Key          string `json:"key" tfsdk:"key"`
	ValuePattern string `json:"valuePattern" tfsdk:"value_pattern"`
}

type AwsSsoConfig struct {
	ScimEndpoint     string              `json:"scimEndpoint" tfsdk:"scim_endpoint"`
	Arn              string              `json:"arn" tfsdk:"arn"`
	GroupNamePattern string              `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	SsoAccessToken   *string             `json:"ssoAccessToken,omitempty" tfsdk:"sso_access_token"`
	AwsRoleMappings  []AwsSsoRoleMapping `json:"awsRoleMappings" tfsdk:"aws_role_mappings"`
	SignInUrl        *string             `json:"signInUrl,omitempty" tfsdk:"sign_in_url"`
}

type AwsSsoRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AwsRole            string               `json:"awsRole" tfsdk:"aws_role"`
	PermissionSetArns  []string             `json:"permissionSetArns" tfsdk:"permission_set_arns"`
}

type GcpPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	GcpRole            string               `json:"gcpRole" tfsdk:"gcp_role"`
}

type OpenShiftPlatformRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	OpenShiftRole      string               `json:"openshiftRole" tfsdk:"openshift_role"`
}

type AwsEnrollmentConfiguration struct {
	ManagementAccountId     string `json:"managementAccountId" tfsdk:"management_account_id"`
	AccountFactoryProductId string `json:"accountFactoryProductId" tfsdk:"account_factory_product_id"`
}

// Azure-specific replication configuration structures
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

type MeshPlatformCreate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

type MeshPlatformUpdate struct {
	ApiVersion string                     `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshPlatformUpdateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshPlatformSpec           `json:"spec" tfsdk:"spec"`
}

type MeshPlatformUpdateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	Uuid             string `json:"uuid" tfsdk:"uuid"`
}

func (c *MeshStackProviderClient) urlForPlatform(uuid string) *url.URL {
	return c.endpoints.Platforms.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadPlatform(uuid string) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, nil // Not found is not an error
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var platform MeshPlatform
	err = json.Unmarshal(data, &platform)
	if err != nil {
		return nil, err
	}
	return &platform, nil
}

func (c *MeshStackProviderClient) CreatePlatform(platform *MeshPlatformCreate) (*MeshPlatform, error) {
	payload, err := json.Marshal(platform)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Platforms.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PLATFORM)
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdPlatform MeshPlatform
	err = json.Unmarshal(data, &createdPlatform)
	if err != nil {
		return nil, err
	}
	return &createdPlatform, nil
}

func (c *MeshStackProviderClient) DeletePlatform(uuid string) error {
	targetUrl := c.urlForPlatform(uuid)
	return c.deleteMeshObject(*targetUrl, 204)
}

func (c *MeshStackProviderClient) UpdatePlatform(uuid string, platform *MeshPlatformUpdate) (*MeshPlatform, error) {
	targetUrl := c.urlForPlatform(uuid)

	payload, err := json.Marshal(platform)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PLATFORM)
	req.Header.Set("Accept", CONTENT_TYPE_PLATFORM)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var updatedPlatform MeshPlatform
	err = json.Unmarshal(data, &updatedPlatform)
	if err != nil {
		return nil, err
	}
	return &updatedPlatform, nil
}
