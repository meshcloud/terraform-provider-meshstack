package provider

type AWSPlatformConfigModel struct {
	Region      string                     `tfsdk:"region"`
	Replication *AWSReplicationConfigModel `tfsdk:"replication"`
}

type AWSReplicationConfigModel struct {
	AccessConfig                   AWSAccessConfigModel             `tfsdk:"access_config"`
	WaitForExternalAvm             bool                             `tfsdk:"wait_for_external_avm"`
	AutomationAccountRole          string                           `tfsdk:"automation_account_role"`
	AutomationAccountExternalId    *string                          `tfsdk:"automation_account_external_id"`
	AccountAccessRole              string                           `tfsdk:"account_access_role"`
	AccountAliasPattern            string                           `tfsdk:"account_alias_pattern"`
	EnforceAccountAlias            bool                             `tfsdk:"enforce_account_alias"`
	AccountEmailPattern            string                           `tfsdk:"account_email_pattern"`
	TenantTags                     *MeshTagConfigModel              `tfsdk:"tenant_tags"`
	AwsSso                         *AWSSsoConfigurationModel        `tfsdk:"aws_sso"`
	EnrollmentConfiguration        *AWSEnrollmentConfigurationModel `tfsdk:"enrollment_configuration"`
	SelfDowngradeAccessRole        bool                             `tfsdk:"self_downgrade_access_role"`
	SkipUserGroupPermissionCleanup bool                             `tfsdk:"skip_user_group_permission_cleanup"`
}

type AWSAccessConfigModel struct {
	OrganizationRootAccountRole       string                          `tfsdk:"organization_root_account_role"`
	OrganizationRootAccountExternalId *string                         `tfsdk:"organization_root_account_external_id"`
	ServiceUserConfig                 *AWSServiceUserConfigModel      `tfsdk:"service_user_config"`
	WorkloadIdentityConfig            *AWSWorkloadIdentityConfigModel `tfsdk:"workload_identity_config"`
}

type AWSServiceUserConfigModel struct {
	AccessKey string `tfsdk:"access_key"`
	SecretKey string `tfsdk:"secret_key"`
}

type AWSWorkloadIdentityConfigModel struct {
	RoleArn string `tfsdk:"role_arn"`
}

type AWSSsoConfigurationModel struct {
	ScimEndpoint     string                     `tfsdk:"scim_endpoint"`
	Arn              string                     `tfsdk:"arn"`
	GroupNamePattern string                     `tfsdk:"group_name_pattern"`
	SsoAccessToken   string                     `tfsdk:"sso_access_token"`
	RoleMappings     map[string]AWSSsoRoleModel `tfsdk:"role_mappings"`
	SignInUrl        string                     `tfsdk:"sign_in_url"`
}

type AWSSsoRoleModel struct {
	AwsRoleName       string   `tfsdk:"aws_role_name"`
	PermissionSetArns []string `tfsdk:"permission_set_arns"`
}

type AWSEnrollmentConfigurationModel struct {
	ManagementAccountId     string `tfsdk:"management_account_id"`
	AccountFactoryProductId string `tfsdk:"account_factory_product_id"`
}
