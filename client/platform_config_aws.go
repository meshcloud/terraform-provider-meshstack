package client

type AwsPlatformConfig struct {
	Region      *string               `json:"region,omitempty" tfsdk:"region"`
	Replication *AwsReplicationConfig `json:"replication,omitempty" tfsdk:"replication"`
}

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

type AwsEnrollmentConfiguration struct {
	ManagementAccountId     string `json:"managementAccountId" tfsdk:"management_account_id"`
	AccountFactoryProductId string `json:"accountFactoryProductId" tfsdk:"account_factory_product_id"`
}
