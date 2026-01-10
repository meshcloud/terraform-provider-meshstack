package client

import (
	"encoding/json"
	"time"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

// Enums

type BuildingBlockDeletionMode string

const (
	BuildingBlockDeletionModeDelete BuildingBlockDeletionMode = "DELETE"
	BuildingBlockDeletionModePurge  BuildingBlockDeletionMode = "PURGE"
)

type MeshBuildingBlockIOType string

const (
	MeshBuildingBlockIOTypeString       MeshBuildingBlockIOType = "STRING"
	MeshBuildingBlockIOTypeCode         MeshBuildingBlockIOType = "CODE"
	MeshBuildingBlockIOTypeInteger      MeshBuildingBlockIOType = "INTEGER"
	MeshBuildingBlockIOTypeBoolean      MeshBuildingBlockIOType = "BOOLEAN"
	MeshBuildingBlockIOTypeFile         MeshBuildingBlockIOType = "FILE"
	MeshBuildingBlockIOTypeList         MeshBuildingBlockIOType = "LIST"
	MeshBuildingBlockIOTypeSingleSelect MeshBuildingBlockIOType = "SINGLE_SELECT"
	MeshBuildingBlockIOTypeMultiSelect  MeshBuildingBlockIOType = "MULTI_SELECT"
)

type MeshBuildingBlockInputAssignmentType string

const (
	MeshBuildingBlockInputAssignmentTypeAuthor                      MeshBuildingBlockInputAssignmentType = "AUTHOR"
	MeshBuildingBlockInputAssignmentTypeUserInput                   MeshBuildingBlockInputAssignmentType = "USER_INPUT"
	MeshBuildingBlockInputAssignmentTypePlatformOperatorManualInput MeshBuildingBlockInputAssignmentType = "PLATFORM_OPERATOR_MANUAL_INPUT"
	MeshBuildingBlockInputAssignmentTypeBuildingBlockOutput         MeshBuildingBlockInputAssignmentType = "BUILDING_BLOCK_OUTPUT"
	MeshBuildingBlockInputAssignmentTypePlatformTenantID            MeshBuildingBlockInputAssignmentType = "PLATFORM_TENANT_ID"
	MeshBuildingBlockInputAssignmentTypeWorkspaceIdentifier         MeshBuildingBlockInputAssignmentType = "WORKSPACE_IDENTIFIER"
	MeshBuildingBlockInputAssignmentTypeProjectIdentifier           MeshBuildingBlockInputAssignmentType = "PROJECT_IDENTIFIER"
	MeshBuildingBlockInputAssignmentTypeFullPlatformIdentifier      MeshBuildingBlockInputAssignmentType = "FULL_PLATFORM_IDENTIFIER"
	MeshBuildingBlockInputAssignmentTypeTenantBuildingBlockUUID     MeshBuildingBlockInputAssignmentType = "TENANT_BUILDING_BLOCK_UUID"
	MeshBuildingBlockInputAssignmentTypeStatic                      MeshBuildingBlockInputAssignmentType = "STATIC"
	MeshBuildingBlockInputAssignmentTypeUserPermissions             MeshBuildingBlockInputAssignmentType = "USER_PERMISSIONS"
)

type MeshBuildingBlockDefinitionOutputAssignmentType string

const (
	MeshBuildingBlockDefinitionOutputAssignmentTypeNone             MeshBuildingBlockDefinitionOutputAssignmentType = "NONE"
	MeshBuildingBlockDefinitionOutputAssignmentTypePlatformTenantID MeshBuildingBlockDefinitionOutputAssignmentType = "PLATFORM_TENANT_ID"
	MeshBuildingBlockDefinitionOutputAssignmentTypeSignInURL        MeshBuildingBlockDefinitionOutputAssignmentType = "SIGN_IN_URL"
	MeshBuildingBlockDefinitionOutputAssignmentTypeResourceURL      MeshBuildingBlockDefinitionOutputAssignmentType = "RESOURCE_URL"
	MeshBuildingBlockDefinitionOutputAssignmentTypeSummary          MeshBuildingBlockDefinitionOutputAssignmentType = "SUMMARY"
)

// Ref types

type BuildingBlockDefinitionRef struct {
	UUID string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}

type MeshIntegrationRef struct {
	UUID string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}

// Implementation types

type MeshBuildingBlockDefinitionVersionTerraformImplementationKnownHost struct {
	Host     string `json:"host" tfsdk:"host"`
	KeyType  string `json:"keyType" tfsdk:"key_type"`
	KeyValue string `json:"keyValue" tfsdk:"key_value"`
}

type MeshBuildingBlockDefinitionVersionTerraformImplementation struct {
	TerraformVersion           string                                                              `json:"terraformVersion" tfsdk:"terraform_version"`
	RepositoryURL              string                                                              `json:"repositoryUrl" tfsdk:"repository_url"`
	Async                      bool                                                                `json:"async" tfsdk:"async"`
	RepositoryPath             *string                                                             `json:"repositoryPath,omitempty" tfsdk:"repository_path"`
	SSHPrivateKey              *SecretEmbedded                                                     `json:"sshPrivateKey,omitempty" tfsdk:"ssh_private_key"`
	RefName                    *string                                                             `json:"refName,omitempty" tfsdk:"ref_name"`
	SSHKnownHost               *MeshBuildingBlockDefinitionVersionTerraformImplementationKnownHost `json:"sshKnownHost,omitempty" tfsdk:"ssh_known_host"`
	UseMeshHTTPBackendFallback bool                                                                `json:"useMeshHttpBackendFallback" tfsdk:"use_mesh_http_backend_fallback"`
}

type MeshBuildingBlockDefinitionVersionGitHubWorkflowsImplementation struct {
	Repository         string             `json:"repository" tfsdk:"repository"`
	Branch             string             `json:"branch" tfsdk:"branch"`
	ApplyWorkflow      string             `json:"applyWorkflow" tfsdk:"apply_workflow"`
	DestroyWorkflow    *string            `json:"destroyWorkflow,omitempty" tfsdk:"destroy_workflow"`
	Async              bool               `json:"async" tfsdk:"async"`
	OmitRunObjectInput bool               `json:"omitRunObjectInput" tfsdk:"omit_run_object_input"`
	IntegrationRef     MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
}

type MeshBuildingBlockDefinitionVersionGitLabPipelineImplementation struct {
	ProjectID            string             `json:"projectId" tfsdk:"project_id"`
	RefName              string             `json:"refName" tfsdk:"ref_name"`
	PipelineTriggerToken SecretEmbedded     `json:"pipelineTriggerToken" tfsdk:"pipeline_trigger_token"`
	IntegrationRef       MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
}

type MeshBuildingBlockDefinitionVersionAzureDevOpsPipelineImplementation struct {
	Project        string             `json:"project" tfsdk:"project"`
	PipelineID     string             `json:"pipelineId" tfsdk:"pipeline_id"`
	Async          bool               `json:"async" tfsdk:"async"`
	IntegrationRef MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
}

// Implementation wrapper with polymorphism

type MeshBuildingBlockDefinitionVersionImplementation struct {
	Content any
}

func (i *MeshBuildingBlockDefinitionVersionImplementation) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a map to determine which implementation type it is
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Check which key is present to determine the type
	if _, ok := raw["terraform"]; ok {
		var impl struct {
			Terraform MeshBuildingBlockDefinitionVersionTerraformImplementation `json:"terraform"`
		}
		if err := json.Unmarshal(data, &impl); err != nil {
			return err
		}
		i.Content = &impl.Terraform
	} else if _, ok := raw["githubWorkflows"]; ok {
		var impl struct {
			GitHubWorkflows MeshBuildingBlockDefinitionVersionGitHubWorkflowsImplementation `json:"githubWorkflows"`
		}
		if err := json.Unmarshal(data, &impl); err != nil {
			return err
		}
		i.Content = &impl.GitHubWorkflows
	} else if _, ok := raw["gitlabPipeline"]; ok {
		var impl struct {
			GitLabPipeline MeshBuildingBlockDefinitionVersionGitLabPipelineImplementation `json:"gitlabPipeline"`
		}
		if err := json.Unmarshal(data, &impl); err != nil {
			return err
		}
		i.Content = &impl.GitLabPipeline
	} else if _, ok := raw["azureDevOpsPipeline"]; ok {
		var impl struct {
			AzureDevOpsPipeline MeshBuildingBlockDefinitionVersionAzureDevOpsPipelineImplementation `json:"azureDevOpsPipeline"`
		}
		if err := json.Unmarshal(data, &impl); err != nil {
			return err
		}
		i.Content = &impl.AzureDevOpsPipeline
	} else {
		// Manual implementation (no fields)
		i.Content = nil
	}

	return nil
}

// Input and Output types

type MeshBuildingBlockDefinitionInput struct {
	DisplayName                 string                               `json:"displayName" tfsdk:"display_name"`
	Type                        MeshBuildingBlockIOType              `json:"type" tfsdk:"type"`
	AssignmentType              MeshBuildingBlockInputAssignmentType `json:"assignmentType" tfsdk:"assignment_type"`
	Argument                    any                                  `json:"argument,omitempty" tfsdk:"argument"`
	IsEnvironment               bool                                 `json:"isEnvironment" tfsdk:"is_environment"`
	IsSensitive                 bool                                 `json:"isSensitive" tfsdk:"is_sensitive"`
	UpdateableByConsumer        bool                                 `json:"updateableByConsumer" tfsdk:"updateable_by_consumer"`
	SelectableValues            []string                             `json:"selectableValues,omitempty" tfsdk:"selectable_values"`
	DefaultValue                any                                  `json:"defaultValue,omitempty" tfsdk:"default_value"`
	Description                 *string                              `json:"description,omitempty" tfsdk:"description"`
	ValueValidationRegex        *string                              `json:"valueValidationRegex,omitempty" tfsdk:"value_validation_regex"`
	ValidationRegexErrorMessage *string                              `json:"validationRegexErrorMessage,omitempty" tfsdk:"validation_regex_error_message"`
}

type MeshBuildingBlockDefinitionOutput struct {
	DisplayName    string                                          `json:"displayName" tfsdk:"display_name"`
	Type           MeshBuildingBlockIOType                         `json:"type" tfsdk:"type"`
	AssignmentType MeshBuildingBlockDefinitionOutputAssignmentType `json:"assignmentType" tfsdk:"assignment_type"`
}

// Main version types

type MeshBuildingBlockDefinitionVersionMetadata struct {
	UUID             *string    `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string     `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        *time.Time `json:"createdOn,omitempty" tfsdk:"created_on"`
}

type MeshBuildingBlockDefinitionVersionSpec struct {
	BuildingBlockDefinitionRef BuildingBlockDefinitionRef                       `json:"buildingBlockDefinitionRef" tfsdk:"building_block_definition_ref"`
	VersionNumber              *int64                                           `json:"versionNumber,omitempty" tfsdk:"version_number"`
	State                      MeshBuildingBlockDefinitionVersionState          `json:"state" tfsdk:"state"`
	Implementation             MeshBuildingBlockDefinitionVersionImplementation `json:"implementation" tfsdk:"implementation"`
	OnlyApplyOncePerTenant     bool                                             `json:"onlyApplyOncePerTenant" tfsdk:"only_apply_once_per_tenant"`
	DeletionMode               BuildingBlockDeletionMode                        `json:"deletionMode" tfsdk:"deletion_mode"`
	RunnerRef                  BuildingBlockRunnerRef                           `json:"runnerRef" tfsdk:"runner_ref"`
	DependencyDefinitionUUIDs  []string                                         `json:"dependencyDefinitionUuids" tfsdk:"dependency_definition_uuids"`
	Inputs                     map[string]MeshBuildingBlockDefinitionInput      `json:"inputs" tfsdk:"inputs"`
	Outputs                    map[string]MeshBuildingBlockDefinitionOutput     `json:"outputs" tfsdk:"outputs"`
}

type MeshBuildingBlockDefinitionVersionStatus struct {
	State      MeshBuildingBlockDefinitionVersionState `json:"state" tfsdk:"state"`
	UsageCount int64                                   `json:"usageCount" tfsdk:"usage_count"`
}

type MeshBuildingBlockDefinitionVersion struct {
	ApiVersion string                                      `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                                      `json:"kind" tfsdk:"kind"`
	Metadata   *MeshBuildingBlockDefinitionVersionMetadata `json:"metadata,omitempty" tfsdk:"metadata"`
	Spec       MeshBuildingBlockDefinitionVersionSpec      `json:"spec" tfsdk:"spec"`
	Status     MeshBuildingBlockDefinitionVersionStatus    `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockDefinitionVersionClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockDefinitionVersion]
}

func newBuildingBlockDefinitionVersionClient(httpClient *internal.HttpClient) MeshBuildingBlockDefinitionVersionClient {
	return MeshBuildingBlockDefinitionVersionClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlockDefinitionVersion](httpClient, "v1-preview"),
	}
}

func (c *MeshBuildingBlockDefinitionVersionClient) Read(uuid string) (*MeshBuildingBlockDefinitionVersion, error) {
	return c.meshObject.Get(uuid)
}
