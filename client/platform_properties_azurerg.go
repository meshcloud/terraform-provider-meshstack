package client

type AzureRgPlatformProperties struct {
	AzureRgLocation     string               `json:"azureRgLocation" tfsdk:"azure_rg_location"`
	AzureRgRoleMappings []AzureRgRoleMapping `json:"azureRgRoleMappings" tfsdk:"azure_rg_role_mappings"`
	AzureFunction       *AzureFunction       `json:"azureFunction,omitempty" tfsdk:"azure_function"`
}

type AzureRgRoleMapping struct {
	MeshProjectRoleRef     MeshProjectRoleRefV2 `json:"projectRoleRef" tfsdk:"project_role_ref"`
	AzureGroupSuffix       string               `json:"azureGroupSuffix" tfsdk:"azure_group_suffix"`
	AzureRoleDefinitionIds []string             `json:"azureRoleDefinitionIds" tfsdk:"azure_role_definition_ids"`
}

type AzureFunction struct {
	AzureFunctionUrl   string `json:"azureFunctionUrl" tfsdk:"azure_function_url"`
	AzureFunctionScope string `json:"azureFunctionScope" tfsdk:"azure_function_scope"`
}
