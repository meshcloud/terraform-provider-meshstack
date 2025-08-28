package client

type AWSPlatformConfig struct {
	Region      string                `json:"region" tfsdk:"region"`
	Replication *AWSReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

type AWSReplicationConfig struct {
	AccessConfig                   AWSAccessConfig             `json:"accessConfig" tfsdk:"access_config"`
	WaitForExternalAvm             bool                        `json:"waitForExternalAvm" tfsdk:"wait_for_external_avm"`
	AutomationAccountRole          string                      `json:"automationAccountRole" tfsdk:"automation_account_role"`
	AutomationAccountExternalId    *string                     `json:"automationAccountExternalId,omitempty" tfsdk:"automation_account_external_id"`
	AccountAccessRole              string                      `json:"accountAccessRole" tfsdk:"account_access_role"`
	AccountAliasPattern            string                      `json:"accountAliasPattern" tfsdk:"account_alias_pattern"`
	EnforceAccountAlias            bool                        `json:"enforceAccountAlias" tfsdk:"enforce_account_alias"`
	AccountEmailPattern            string                      `json:"accountEmailPattern" tfsdk:"account_email_pattern"`
	TenantTags                     *MeshTagConfig              `json:"tenantTags,omitempty" tfsdk:"tenant_tags"`
	AwsSso                         *AWSSsoConfiguration        `json:"awsSso,omitempty" tfsdk:"aws_sso"`
	EnrollmentConfiguration        *AWSEnrollmentConfiguration `json:"enrollmentConfiguration,omitempty" tfsdk:"enrollment_configuration"`
	SelfDowngradeAccessRole        bool                        `json:"selfDowngradeAccessRole" tfsdk:"self_downgrade_access_role"`
	SkipUserGroupPermissionCleanup bool                        `json:"skipUserGroupPermissionCleanup" tfsdk:"skip_user_group_permission_cleanup"`
}

type AWSAccessConfig struct {
	OrganizationRootAccountRole       string                     `json:"organizationRootAccountRole" tfsdk:"organization_root_account_role"`
	OrganizationRootAccountExternalId *string                    `json:"organizationRootAccountExternalId,omitempty" tfsdk:"organization_root_account_external_id"`
	ServiceUserConfig                 *AWSServiceUserConfig      `json:"serviceUserConfig,omitempty" tfsdk:"service_user_config"`
	WorkloadIdentityConfig            *AWSWorkloadIdentityConfig `json:"workloadIdentityConfig,omitempty" tfsdk:"workload_identity_config"`
}

type AWSServiceUserConfig struct {
	AccessKey string `json:"accessKey" tfsdk:"access_key"`
	SecretKey string `json:"secretKey" tfsdk:"secret_key"`
}

type AWSWorkloadIdentityConfig struct {
	RoleArn string `json:"roleArn" tfsdk:"role_arn"`
}

type AWSSsoConfiguration struct {
	ScimEndpoint     string                `json:"scimEndpoint" tfsdk:"scim_endpoint"`
	Arn              string                `json:"arn" tfsdk:"arn"`
	GroupNamePattern string                `json:"groupNamePattern" tfsdk:"group_name_pattern"`
	SsoAccessToken   string                `json:"ssoAccessToken" tfsdk:"sso_access_token"`
	RoleMappings     map[string]AWSSsoRole `json:"roleMappings" tfsdk:"role_mappings"`
	SignInUrl        string                `json:"signInUrl" tfsdk:"sign_in_url"`
}

type AWSSsoRole struct {
	AwsRoleName       string   `json:"awsRoleName" tfsdk:"aws_role_name"`
	PermissionSetArns []string `json:"permissionSetArns" tfsdk:"permission_set_arns"`
}

type AWSEnrollmentConfiguration struct {
	ManagementAccountId     string `json:"managementAccountId" tfsdk:"management_account_id"`
	AccountFactoryProductId string `json:"accountFactoryProductId" tfsdk:"account_factory_product_id"`
}
