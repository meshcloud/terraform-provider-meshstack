package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspace struct {
	ApiVersion string                `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                `json:"kind" tfsdk:"kind"`
	Metadata   MeshWorkspaceMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshWorkspaceSpec     `json:"spec" tfsdk:"spec"`
}

type MeshWorkspaceMetadata struct {
	Name      string              `json:"name" tfsdk:"name"`
	CreatedOn string              `json:"createdOn" tfsdk:"created_on"`
	DeletedOn *string             `json:"deletedOn" tfsdk:"deleted_on"`
	Tags      map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshWorkspaceSpec struct {
	DisplayName                  string `json:"displayName" tfsdk:"display_name"`
	PlatformBuilderAccessEnabled *bool  `json:"platformBuilderAccessEnabled,omitempty" tfsdk:"platform_builder_access_enabled"`
}

type MeshWorkspaceCreate struct {
	ApiVersion string                      `json:"apiVersion" tfsdk:"api_version"`
	Metadata   MeshWorkspaceCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshWorkspaceSpec           `json:"spec" tfsdk:"spec"`
}
type MeshWorkspaceCreateMetadata struct {
	Name string              `json:"name" tfsdk:"name"`
	Tags map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshWorkspaceClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspace]
}

func newWorkspaceClient(ctx context.Context, httpClient *internal.HttpClient) MeshWorkspaceClient {
	return MeshWorkspaceClient{internal.NewMeshObjectClient[MeshWorkspace](ctx, httpClient, "v2")}
}

func (c MeshWorkspaceClient) Read(ctx context.Context, name string) (*MeshWorkspace, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshWorkspaceClient) Create(ctx context.Context, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.meshObject.Post(ctx, workspace)
}

func (c MeshWorkspaceClient) Update(ctx context.Context, name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.meshObject.Put(ctx, name, workspace)
}

func (c MeshWorkspaceClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
