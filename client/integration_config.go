package client

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

type MeshIntegrationConfigType string

var (
	MeshIntegrationConfigTypes           = enum.Enum[MeshIntegrationConfigType]{}
	MeshIntegrationConfigTypeGithub      = MeshIntegrationConfigTypes.Entry("github")
	MeshIntegrationConfigTypeGitlab      = MeshIntegrationConfigTypes.Entry("gitlab")
	MeshIntegrationConfigTypeAzureDevops = MeshIntegrationConfigTypes.Entry("azuredevops")
)

type MeshIntegrationGithubConfig struct {
	Owner         string                  `json:"owner" tfsdk:"owner"`
	BaseUrl       string                  `json:"baseUrl" tfsdk:"base_url"`
	AppId         string                  `json:"appId" tfsdk:"app_id"`
	AppPrivateKey types.Secret            `json:"appPrivateKey" tfsdk:"app_private_key"`
	RunnerRef     *BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

type MeshIntegrationGitlabConfig struct {
	BaseUrl   string                  `json:"baseUrl" tfsdk:"base_url"`
	RunnerRef *BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

type MeshIntegrationAzureDevopsConfig struct {
	BaseUrl             string                  `json:"baseUrl" tfsdk:"base_url"`
	Organization        string                  `json:"organization" tfsdk:"organization"`
	PersonalAccessToken types.Secret            `json:"personalAccessToken" tfsdk:"personal_access_token"`
	RunnerRef           *BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

type MeshIntegrationConfig struct {
	Type        enum.Entry[MeshIntegrationConfigType] `json:"type" tfsdk:"-"`
	Github      *MeshIntegrationGithubConfig          `json:"github,omitempty" tfsdk:"github"`
	Gitlab      *MeshIntegrationGitlabConfig          `json:"gitlab,omitempty" tfsdk:"gitlab"`
	AzureDevops *MeshIntegrationAzureDevopsConfig     `json:"azuredevops,omitempty" tfsdk:"azuredevops"`
}

func (m MeshIntegrationConfig) InferTypeFromNonNilField() (result enum.Entry[MeshIntegrationConfigType]) {
	setResultIfNotNil := func(implType enum.Entry[MeshIntegrationConfigType], v any) {
		if !reflect.ValueOf(v).IsZero() {
			if len(result) > 0 && result != implType {
				panic(fmt.Errorf("inferred config type %s but already set to %s", implType, result))
			}
			result = implType
		}
	}
	setResultIfNotNil(MeshIntegrationConfigTypeGithub, m.Github)
	setResultIfNotNil(MeshIntegrationConfigTypeGitlab, m.Gitlab)
	setResultIfNotNil(MeshIntegrationConfigTypeAzureDevops, m.AzureDevops)
	if len(result) == 0 {
		panic("cannot infer config type")
	}
	return
}

func (m MeshIntegrationConfig) MarshalJSON() ([]byte, error) {
	m.Type = m.InferTypeFromNonNilField()
	// Using wrapped type avoids calling MarshalJSON recursively!
	type wrapped MeshIntegrationConfig
	return json.Marshal(wrapped(m))
}

func (m *MeshIntegrationConfig) UnmarshalJSON(bytes []byte) error {
	type wrapped MeshIntegrationConfig
	var target wrapped
	if err := json.Unmarshal(bytes, &target); err != nil {
		return err
	}
	*m = MeshIntegrationConfig(target)
	return nil
}
