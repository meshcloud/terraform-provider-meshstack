package client

import "github.com/meshcloud/terraform-provider-meshstack/client/types"

type AwsPlatformConfig struct {
	Region      string                `json:"region,omitempty" tfsdk:"region"`
	Replication *AwsReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
	Metering    *AwsMeteringConfig    `json:"metering,omitempty" tfsdk:"metering"`
}

type AwsReplicationConfig struct {
	AccessConfig                                  AwsAccessConfig             `json:"accessConfig" tfsdk:"access_config"`
	WaitForExternalAvm                            bool                        `json:"waitForExternalAvm" tfsdk:"wait_for_external_avm"`
	AutomationAccountRole                         string                      `json:"automationAccountRole" tfsdk:"automation_account_role"`
	AutomationAccountExternalId                   *string                     `json:"automationAccountExternalId,omitempty" tfsdk:"automation_account_external_id"`
	AccountAccessRole                             string                      `json:"accountAccessRole" tfsdk:"account_access_role"`
	AccountAliasPattern                           string                      `json:"accountAliasPattern" tfsdk:"account_alias_pattern"`
	EnforceAccountAlias                           bool                        `json:"enforceAccountAlias" tfsdk:"enforce_account_alias"`
	AccountEmailPattern                           string                      `json:"accountEmailPattern" tfsdk:"account_email_pattern"`
	TenantTags                                    *MeshTenantTags             `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	AwsSso                                        *AwsSsoConfig               `json:"awsSso,omitempty" tfsdk:"aws_sso"`
	EnrollmentConfiguration                       *AwsEnrollmentConfiguration `json:"enrollmentConfiguration,omitempty" tfsdk:"enrollment_configuration"`
	SelfDowngradeAccessRole                       bool                        `json:"selfDowngradeAccessRole" tfsdk:"self_downgrade_access_role"`
	SkipUserGroupPermissionCleanup                bool                        `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
	AllowHierarchicalOrganizationalUnitAssignment bool                        `json:"allowHierarchicalOrganizationalUnitAssignment" tfsdk:"allow_hierarchical_organizational_unit_assignment"`
}

type AwsAccessConfig struct {
	OrganizationRootAccountRole       string  `json:"organizationRootAccountRole" tfsdk:"organization_root_account_role"`
	OrganizationRootAccountExternalId *string `json:"organizationRootAccountExternalId,omitempty" tfsdk:"organization_root_account_external_id"`
	Auth                              AwsAuth `json:"auth" tfsdk:"auth"`
}

type AwsAuth struct {
	Type             string                         `json:"type" tfsdk:"type"`
	Credential       *AwsServiceUserCredential      `json:"credential,omitempty" tfsdk:"credential"`
	WorkloadIdentity *AwsWorkloadIdentityCredential `json:"workloadIdentity,omitempty" tfsdk:"workload_identity"`
}

type AwsServiceUserCredential struct {
	AccessKey string       `json:"accessKey" tfsdk:"access_key"`
	SecretKey types.Secret `json:"secretKey" tfsdk:"secret_key"`
}

type AwsWorkloadIdentityCredential struct {
	RoleArn string `json:"roleArn" tfsdk:"role_arn"`
}

type AwsSsoConfig struct {
	ScimEndpoint     string              `json:"scimEndpoint" tfsdk:"scim_endpoint"`
	Arn              string              `json:"arn" tfsdk:"arn"`
	GroupNamePattern string              `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	SsoAccessToken   types.Secret        `json:"ssoAccessToken" tfsdk:"sso_access_token"`
	AwsRoleMappings  []AwsSsoRoleMapping `json:"awsRoleMappings" tfsdk:"aws_role_mappings"`
	SignInUrl        string              `json:"signInUrl" tfsdk:"sign_in_url"`
}

type AwsSsoRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AwsRole            string               `json:"awsRole" tfsdk:"aws_role"`
	PermissionSetArns  []string             `json:"permissionSetArns" tfsdk:"permission_set_arns"`
}

type AwsEnrollmentConfiguration struct {
	ManagementAccountId     string `json:"managementAccountId" tfsdk:"management_account_id"`
	AccountFactoryProductId string `json:"accountFactoryProductId" tfsdk:"account_factory_product_id"`
}

type AwsMeteringConfig struct {
	AccessConfig                   AwsAccessConfig                      `json:"accessConfig" tfsdk:"access_config"`
	Filter                         string                               `json:"filter" tfsdk:"filter"`
	ReservedInstanceFairChargeback bool                                 `json:"reservedInstanceFairChargeback" tfsdk:"reserved_instance_fair_chargeback"`
	SavingsPlanFairChargeback      bool                                 `json:"savingsPlanFairChargeback" tfsdk:"savings_plan_fair_chargeback"`
	Processing                     MeshPlatformMeteringProcessingConfig `json:"processing" tfsdk:"processing"`
}
