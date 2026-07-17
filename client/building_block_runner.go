package client

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshBuildingBlockRunnerImplementationType string

const (
	MeshBuildingBlockRunnerImplementationTypeTerraform           MeshBuildingBlockRunnerImplementationType = "TERRAFORM"
	MeshBuildingBlockRunnerImplementationTypeGithubWorkflow      MeshBuildingBlockRunnerImplementationType = "GITHUB_WORKFLOW"
	MeshBuildingBlockRunnerImplementationTypeGitlabPipeline      MeshBuildingBlockRunnerImplementationType = "GITLAB_PIPELINE"
	MeshBuildingBlockRunnerImplementationTypeAzureDevopsPipeline MeshBuildingBlockRunnerImplementationType = "AZURE_DEVOPS_PIPELINE"
	MeshBuildingBlockRunnerImplementationTypeManual              MeshBuildingBlockRunnerImplementationType = "MANUAL"
	MeshBuildingBlockRunnerImplementationTypeAll                 MeshBuildingBlockRunnerImplementationType = "ALL"
)

var MeshBuildingBlockRunnerImplementationTypes = []string{
	string(MeshBuildingBlockRunnerImplementationTypeTerraform),
	string(MeshBuildingBlockRunnerImplementationTypeGithubWorkflow),
	string(MeshBuildingBlockRunnerImplementationTypeGitlabPipeline),
	string(MeshBuildingBlockRunnerImplementationTypeAzureDevopsPipeline),
	string(MeshBuildingBlockRunnerImplementationTypeManual),
	string(MeshBuildingBlockRunnerImplementationTypeAll),
}

type MeshBuildingBlockRunner struct {
	Metadata MeshBuildingBlockRunnerMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshBuildingBlockRunnerSpec     `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockRunnerMetadata struct {
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        *string `json:"createdOn,omitempty" tfsdk:"created_on"`
	LastSeen         *string `json:"lastSeen,omitempty" tfsdk:"last_seen"`
}

type MeshBuildingBlockRunnerSpec struct {
	DisplayName                string                         `json:"displayName" tfsdk:"display_name"`
	PublicKey                  string                         `json:"publicKey" tfsdk:"public_key"`
	ImplementationType         string                         `json:"implementationType" tfsdk:"implementation_type"`
	Restriction                *string                        `json:"restriction,omitempty" tfsdk:"restriction"`
	IsSelfHosted               *bool                          `json:"isSelfHosted,omitempty" tfsdk:"is_self_hosted"`
	WorkloadIdentityFederation *MeshRunnerWorkloadIdentityFed `json:"workloadIdentityFederation,omitempty" tfsdk:"workload_identity_federation"`
}

type MeshRunnerWorkloadIdentityFed struct {
	Subject *string                      `json:"subject,omitempty" tfsdk:"subject"`
	Issuer  *string                      `json:"issuer,omitempty" tfsdk:"issuer"`
	Gcp     *MeshRunnerWifProviderConfig `json:"gcp,omitempty" tfsdk:"gcp"`
	Aws     *MeshRunnerWifProviderConfig `json:"aws,omitempty" tfsdk:"aws"`
	Azure   *MeshRunnerWifProviderConfig `json:"azure,omitempty" tfsdk:"azure"`
}

type MeshRunnerWifProviderConfig struct {
	Audience  string `json:"audience" tfsdk:"audience"`
	TokenPath string `json:"tokenPath" tfsdk:"token_path"`
}

type MeshBuildingBlockRunnerClient interface {
	Create(ctx context.Context, runner MeshBuildingBlockRunner) (*MeshBuildingBlockRunner, error)
	Read(ctx context.Context, uuid string) (*MeshBuildingBlockRunner, error)
	Update(ctx context.Context, runner MeshBuildingBlockRunner) (*MeshBuildingBlockRunner, error)
	Delete(ctx context.Context, uuid string) error
}

type meshBuildingBlockRunnerClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlockRunner]
}

func newBuildingBlockRunnerClient(ctx context.Context, httpClient internal.HttpClient) MeshBuildingBlockRunnerClient {
	return meshBuildingBlockRunnerClient{internal.NewMeshObjectClient[MeshBuildingBlockRunner](ctx, httpClient, "v1-preview")}
}

func (c meshBuildingBlockRunnerClient) Create(ctx context.Context, runner MeshBuildingBlockRunner) (*MeshBuildingBlockRunner, error) {
	return c.meshObject.Post(ctx, runner)
}

func (c meshBuildingBlockRunnerClient) Read(ctx context.Context, uuid string) (*MeshBuildingBlockRunner, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c meshBuildingBlockRunnerClient) Update(ctx context.Context, runner MeshBuildingBlockRunner) (*MeshBuildingBlockRunner, error) {
	if runner.Metadata.Uuid == nil || *runner.Metadata.Uuid == "" {
		return nil, fmt.Errorf("missing metadata.uuid")
	}

	return c.meshObject.Put(ctx, *runner.Metadata.Uuid, runner)
}

func (c meshBuildingBlockRunnerClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
