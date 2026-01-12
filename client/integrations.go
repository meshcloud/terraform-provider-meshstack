package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshIntegration struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                  `json:"kind" tfsdk:"kind"`
	Metadata   MeshIntegrationMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshIntegrationSpec     `json:"spec" tfsdk:"spec"`
	Status     *MeshIntegrationStatus  `json:"status,omitempty" tfsdk:"status"`
}

type MeshIntegrationMetadata struct {
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        *string `json:"createdOn,omitempty" tfsdk:"created_on"`
}

type MeshIntegrationSpec struct {
	DisplayName string                `json:"displayName" tfsdk:"display_name"`
	Config      MeshIntegrationConfig `json:"config" tfsdk:"config"`
}

type MeshIntegrationStatus struct {
	IsBuiltIn                  bool                            `json:"isBuiltIn" tfsdk:"is_built_in"`
	WorkloadIdentityFederation *MeshWorkloadIdentityFederation `json:"workloadIdentityFederation,omitempty" tfsdk:"workload_identity_federation"`
}

// Integration Config wrapper with type discrimination.
type MeshIntegrationConfig struct {
	Type        string                            `json:"type" tfsdk:"type"`
	Github      *MeshIntegrationGithubConfig      `json:"github,omitempty" tfsdk:"github"`
	Gitlab      *MeshIntegrationGitlabConfig      `json:"gitlab,omitempty" tfsdk:"gitlab"`
	AzureDevops *MeshIntegrationAzureDevopsConfig `json:"azuredevops,omitempty" tfsdk:"azuredevops"`
}

// GitHub Integration.
type MeshIntegrationGithubConfig struct {
	Owner         string                 `json:"owner" tfsdk:"owner"`
	BaseUrl       string                 `json:"baseUrl" tfsdk:"base_url"`
	AppId         string                 `json:"appId" tfsdk:"app_id"`
	AppPrivateKey string                 `json:"appPrivateKey" tfsdk:"app_private_key"`
	RunnerRef     BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// GitLab Integration.
type MeshIntegrationGitlabConfig struct {
	BaseUrl   string                 `json:"baseUrl" tfsdk:"base_url"`
	RunnerRef BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// Azure DevOps Integration.
type MeshIntegrationAzureDevopsConfig struct {
	BaseUrl             string                 `json:"baseUrl" tfsdk:"base_url"`
	Organization        string                 `json:"organization" tfsdk:"organization"`
	PersonalAccessToken string                 `json:"personalAccessToken" tfsdk:"personal_access_token"`
	RunnerRef           BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// Building Block Runner Reference.
type BuildingBlockRunnerRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind" tfsdk:"kind"`
}

// Workload Identity Federation.
type MeshWorkloadIdentityFederation struct {
	Issuer  string              `json:"issuer" tfsdk:"issuer"`
	Subject string              `json:"subject" tfsdk:"subject"`
	Gcp     *MeshWifProvider    `json:"gcp,omitempty" tfsdk:"gcp"`
	Aws     *MeshAwsWifProvider `json:"aws,omitempty" tfsdk:"aws"`
	Azure   *MeshWifProvider    `json:"azure,omitempty" tfsdk:"azure"`
}

type MeshWifProvider struct {
	Audience string `json:"audience" tfsdk:"audience"`
}

type MeshAwsWifProvider struct {
	Audience   string `json:"audience" tfsdk:"audience"`
	Thumbprint string `json:"thumbprint" tfsdk:"thumbprint"`
}

type MeshIntegrationClient struct {
	meshObject internal.MeshObjectClient[MeshIntegration]
}

func newIntegrationClient(ctx context.Context, httpClient *internal.HttpClient) MeshIntegrationClient {
	return MeshIntegrationClient{
		meshObject: internal.NewMeshObjectClient[MeshIntegration](ctx, httpClient, "v1-preview"),
	}
}

func (c MeshIntegrationClient) integrationId(workspace string, uuid string) string {
	return workspace + "/" + uuid
}

func (c MeshIntegrationClient) Read(ctx context.Context, workspace string, uuid string) (*MeshIntegration, error) {
	return c.meshObject.Get(ctx, c.integrationId(workspace, uuid))
}

func (c MeshIntegrationClient) List(ctx context.Context) ([]MeshIntegration, error) {
	return c.meshObject.List(ctx)
}
