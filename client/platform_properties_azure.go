package client

type AzurePlatformProperties struct {
	AzureRoleMappings      []AzureRoleMapping `json:"azureRoleMappings" tfsdk:"azure_role_mappings"`
	AzureManagementGroupId string             `json:"azureManagementGroupId" tfsdk:"azure_management_group_id"`
}

type AzureRoleMapping struct {
	MeshProjectRoleRef   MeshProjectRoleRefV2  `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AzureGroupSuffix     string                `json:"azureGroupSuffix" tfsdk:"azure_group_suffix"`
	AzureRoleDefinitions []AzureRoleDefinition `json:"azureRoleDefinitions" tfsdk:"azure_role_definitions"`
}

type AzureRoleDefinition struct {
	AzureRoleDefinitionId string  `json:"azureRoleDefinitionId" tfsdk:"azure_role_definition_id"`
	AbacCondition         *string `json:"abacCondition" tfsdk:"abac_condition"`
}
