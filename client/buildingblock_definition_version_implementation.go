package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

type MeshBuildingBlockImplementationType string

var (
	MeshBuildingBlockImplementationTypes                   = enum.Enum[MeshBuildingBlockImplementationType]{}
	MeshBuildingBlockImplementationTypeManual              = MeshBuildingBlockImplementationTypes.Entry("manual")
	MeshBuildingBlockImplementationTypeTerraform           = MeshBuildingBlockImplementationTypes.Entry("terraform")
	MeshBuildingBlockImplementationTypeGithubWorkflows     = MeshBuildingBlockImplementationTypes.Entry("githubWorkflows")
	MeshBuildingBlockImplementationTypeGitlabPipeline      = MeshBuildingBlockImplementationTypes.Entry("gitlabPipeline")
	MeshBuildingBlockImplementationTypeAzureDevOpsPipeline = MeshBuildingBlockImplementationTypes.Entry("azureDevOpsPipeline")
)

type MeshBuildingBlockDefinitionSshKnownHost struct {
	Host     string `json:"host" tfsdk:"host"`
	KeyType  string `json:"keyType" tfsdk:"key_type"`
	KeyValue string `json:"keyValue" tfsdk:"key_value"`
}

type MeshBuildingBlockDefinitionTerraformImplementation struct {
	TerraformVersion           string                                   `json:"terraformVersion" tfsdk:"terraform_version"`
	RepositoryURL              string                                   `json:"repositoryUrl" tfsdk:"repository_url"`
	Async                      bool                                     `json:"async" tfsdk:"async"`
	RepositoryPath             *string                                  `json:"repositoryPath,omitempty" tfsdk:"repository_path"`
	RefName                    *string                                  `json:"refName,omitempty" tfsdk:"ref_name"`
	SSHKnownHost               *MeshBuildingBlockDefinitionSshKnownHost `json:"sshKnownHost,omitempty" tfsdk:"ssh_known_host"`
	UseMeshHTTPBackendFallback bool                                     `json:"useMeshHttpBackendFallback" tfsdk:"use_mesh_http_backend_fallback"`
	SSHPrivateKey              *types.Secret                            `json:"sshPrivateKey,omitempty" tfsdk:"ssh_private_key"`
}

type MeshBuildingBlockDefinitionGitHubWorkflowsImplementation struct {
	Repository         string             `json:"repository" tfsdk:"repository"`
	Branch             string             `json:"branch" tfsdk:"branch"`
	ApplyWorkflow      string             `json:"applyWorkflow" tfsdk:"apply_workflow"`
	DestroyWorkflow    *string            `json:"destroyWorkflow" tfsdk:"destroy_workflow"`
	Async              bool               `json:"async" tfsdk:"async"`
	OmitRunObjectInput bool               `json:"omitRunObjectInput" tfsdk:"omit_run_object_input"`
	IntegrationRef     MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
}

type MeshBuildingBlockDefinitionManualImplementation struct {
}

type MeshBuildingBlockDefinitionGitLabPipelineImplementation struct {
	ProjectID            string             `json:"projectId" tfsdk:"project_id"`
	RefName              string             `json:"refName" tfsdk:"ref_name"`
	IntegrationRef       MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
	PipelineTriggerToken types.Secret       `json:"pipelineTriggerToken" tfsdk:"pipeline_trigger_token"`
}

type MeshBuildingBlockDefinitionAzureDevOpsPipelineImplementation struct {
	Project        string             `json:"project" tfsdk:"project"`
	PipelineID     string             `json:"pipelineId" tfsdk:"pipeline_id"`
	Async          bool               `json:"async" tfsdk:"async"`
	IntegrationRef MeshIntegrationRef `json:"integrationRef" tfsdk:"integration_ref"`
}

type MeshBuildingBlockDefinitionImplementation struct {
	Type                enum.Entry[MeshBuildingBlockImplementationType]               `json:"type" tfsdk:"-"`
	Manual              *MeshBuildingBlockDefinitionManualImplementation              `json:"manual,omitempty" tfsdk:"manual"`
	GithubWorkflows     *MeshBuildingBlockDefinitionGitHubWorkflowsImplementation     `json:"githubWorkflows,omitempty" tfsdk:"github_workflows"`
	AzureDevOpsPipeline *MeshBuildingBlockDefinitionAzureDevOpsPipelineImplementation `json:"azureDevOpsPipeline,omitempty" tfsdk:"azure_devops_pipeline"`
	GitlabPipeline      *MeshBuildingBlockDefinitionGitLabPipelineImplementation      `json:"gitlabPipeline,omitempty" tfsdk:"gitlab_pipeline"`
	Terraform           *MeshBuildingBlockDefinitionTerraformImplementation           `json:"terraform,omitempty" tfsdk:"terraform"`
}

func (m MeshBuildingBlockDefinitionImplementation) InferTypeFromNonNilField() (result enum.Entry[MeshBuildingBlockImplementationType]) {
	setResultIfNotNil := func(implType enum.Entry[MeshBuildingBlockImplementationType], v any) {
		// Manual implementation is an empty struct, so carefully check v for nilness using reflection!
		if !reflect.ValueOf(v).IsZero() {
			if len(result) > 0 && result != implType {
				panic(fmt.Errorf("inferred implementation type %s but already set to %s", implType, result))
			}
			result = implType
		}
	}
	setResultIfNotNil(MeshBuildingBlockImplementationTypeManual, m.Manual)
	setResultIfNotNil(MeshBuildingBlockImplementationTypeTerraform, m.Terraform)
	setResultIfNotNil(MeshBuildingBlockImplementationTypeGithubWorkflows, m.GithubWorkflows)
	setResultIfNotNil(MeshBuildingBlockImplementationTypeGitlabPipeline, m.GitlabPipeline)
	setResultIfNotNil(MeshBuildingBlockImplementationTypeAzureDevOpsPipeline, m.AzureDevOpsPipeline)
	if len(result) == 0 {
		panic("cannot infer implementation type")
	}
	return
}

func (m MeshBuildingBlockDefinitionImplementation) MarshalJSON() ([]byte, error) {
	if len(m.Type) == 0 {
		m.Type = m.InferTypeFromNonNilField()
	}
	type wrapped MeshBuildingBlockDefinitionImplementation
	return json.Marshal(wrapped(m))
}

func (m *MeshBuildingBlockDefinitionImplementation) UnmarshalJSON(bytes []byte) error {
	type wrapped MeshBuildingBlockDefinitionImplementation
	var target wrapped
	if err := json.Unmarshal(bytes, &target); err != nil {
		return err
	}
	*m = MeshBuildingBlockDefinitionImplementation(target)
	if m.Type == MeshBuildingBlockImplementationTypeManual {
		m.Manual = &MeshBuildingBlockDefinitionManualImplementation{}
	}
	return nil
}
