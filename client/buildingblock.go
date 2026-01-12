package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

const (
	MESH_BUILDING_BLOCK_IO_TYPE_STRING        = "STRING"
	MESH_BUILDING_BLOCK_IO_TYPE_INTEGER       = "INTEGER"
	MESH_BUILDING_BLOCK_IO_TYPE_BOOLEAN       = "BOOLEAN"
	MESH_BUILDING_BLOCK_IO_TYPE_SINGLE_SELECT = "SINGLE_SELECT"
	MESH_BUILDING_BLOCK_IO_TYPE_MULTI_SELECT  = "MULTI_SELECT"
	MESH_BUILDING_BLOCK_IO_TYPE_FILE          = "FILE"
	MESH_BUILDING_BLOCK_IO_TYPE_LIST          = "LIST"
	MESH_BUILDING_BLOCK_IO_TYPE_CODE          = "CODE"
)

type MeshBuildingBlock struct {
	ApiVersion string                    `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                    `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockSpec     `json:"spec" tfsdk:"spec"`
	Status     MeshBuildingBlockStatus   `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockMetadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	DefinitionUuid      string  `json:"definitionUuid" tfsdk:"definition_uuid"`
	DefinitionVersion   int64   `json:"definitionVersion" tfsdk:"definition_version"`
	TenantIdentifier    string  `json:"tenantIdentifier" tfsdk:"tenant_identifier"`
	ForcePurge          bool    `json:"forcePurge" tfsdk:"force_purge"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy *string `json:"markedForDeletionBy" tfsdk:"marked_for_deletion_by"`
}

type MeshBuildingBlockSpec struct {
	DisplayName          string                    `json:"displayName" tfsdk:"display_name"`
	Inputs               []MeshBuildingBlockIO     `json:"inputs" tfsdk:"inputs"`
	ParentBuildingBlocks []MeshBuildingBlockParent `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockIO struct {
	Key       string `json:"key" tfsdk:"key"`
	Value     any    `json:"value" tfsdk:"value"`
	ValueType string `json:"valueType" tfsdk:"value_type"`
}

type MeshBuildingBlockParent struct {
	BuildingBlockUuid string `json:"buildingBlockUuid" tfsdk:"buildingblock_uuid"`
	DefinitionUuid    string `json:"definitionUuid" tfsdk:"definition_uuid"`
}

type MeshBuildingBlockStatus struct {
	Status  string                `json:"status" tfsdk:"status"`
	Outputs []MeshBuildingBlockIO `json:"outputs" tfsdk:"outputs"`
}

type MeshBuildingBlockCreate struct {
	ApiVersion string                          `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                          `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockSpec           `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockCreateMetadata struct {
	DefinitionUuid    string `json:"definitionUuid" tfsdk:"definition_uuid"`
	DefinitionVersion int64  `json:"definitionVersion" tfsdk:"definition_version"`
	TenantIdentifier  string `json:"tenantIdentifier" tfsdk:"tenant_identifier"`
}

type MeshBuildingBlockDefinitionRef struct {
	Kind string `json:"kind" tfsdk:"kind"`
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshBuildingBlockClient struct {
	meshObject internal.MeshObjectClient[MeshBuildingBlock]
}

func newBuildingBlockClient(ctx context.Context, httpClient *internal.HttpClient) MeshBuildingBlockClient {
	return MeshBuildingBlockClient{
		meshObject: internal.NewMeshObjectClient[MeshBuildingBlock](ctx, httpClient, "v1"),
	}
}

func (c MeshBuildingBlockClient) Read(ctx context.Context, uuid string) (*MeshBuildingBlock, error) {
	return c.meshObject.Get(ctx, uuid)
}

func (c MeshBuildingBlockClient) Create(ctx context.Context, bb *MeshBuildingBlockCreate) (*MeshBuildingBlock, error) {
	return c.meshObject.Post(ctx, bb)
}

func (c MeshBuildingBlockClient) Delete(ctx context.Context, uuid string) error {
	return c.meshObject.Delete(ctx, uuid)
}
