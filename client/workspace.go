package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

type MeshWorkspace struct {
	Metadata MeshWorkspaceMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshWorkspaceSpec     `json:"spec" tfsdk:"spec"`
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
	Metadata MeshWorkspaceCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshWorkspaceSpec           `json:"spec" tfsdk:"spec"`
}
type MeshWorkspaceCreateMetadata struct {
	Name string              `json:"name" tfsdk:"name"`
	Tags map[string][]string `json:"tags" tfsdk:"tags"`
}

type MeshWorkspaceClient interface {
	Read(ctx context.Context, name string) (*MeshWorkspace, error)
	Create(ctx context.Context, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error)
	Update(ctx context.Context, name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error)
	Delete(ctx context.Context, name string) error
}

type meshWorkspaceClient struct {
	meshObject internal.MeshObjectClient[MeshWorkspace]
}

func newWorkspaceClient(ctx context.Context, httpClient *internal.HttpClient) meshWorkspaceClient {
	return meshWorkspaceClient{internal.NewMeshObjectClient[MeshWorkspace](ctx, httpClient, "v2")}
}

func (c meshWorkspaceClient) Read(ctx context.Context, name string) (*MeshWorkspace, error) {
	return c.meshObject.Get(ctx, name)
}

func (c meshWorkspaceClient) Create(ctx context.Context, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.meshObject.Post(ctx, workspace)
}

func (c meshWorkspaceClient) Update(ctx context.Context, name string, workspace *MeshWorkspaceCreate) (*MeshWorkspace, error) {
	return c.meshObject.Put(ctx, name, workspace)
}

func (c meshWorkspaceClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
