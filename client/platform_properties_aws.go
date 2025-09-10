package client

type AwsPlatformProperties struct {
	AwsTargetOrgUnitId string           `json:"awsTargetOrgUnitId" tfsdk:"aws_target_org_unit_id"`
	AwsEnrollAccount   bool             `json:"awsEnrollAccount" tfsdk:"aws_enroll_account"`
	AwsLambdaArn       *string          `json:"awsLambdaArn" tfsdk:"aws_lambda_arn"`
	AwsRoleMappings    []AwsRoleMapping `json:"awsRoleMappings" tfsdk:"aws_role_mappings"`
}

type AwsRoleMapping struct {
	MeshProjectRoleRef MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	PlatformRole       string               `json:"platformRole" tfsdk:"platform_role"`
	Policies           []string             `json:"policies" tfsdk:"policies"`
}
