package client

import (
	"net/url"
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

	CONTENT_TYPE_BUILDING_BLOCK = "application/vnd.meshcloud.api.meshbuildingblock.v1.hal+json"
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

func (c *MeshStackProviderClient) urlForBuildingBlock(uuid string) *url.URL {
	return c.endpoints.BuildingBlocks.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadBuildingBlock(uuid string) (*MeshBuildingBlock, error) {
	return unmarshalBodyIfPresent[MeshBuildingBlock](c.doAuthenticatedRequest("GET", c.urlForBuildingBlock(uuid),
		withAccept(CONTENT_TYPE_BUILDING_BLOCK),
	))
}

func (c *MeshStackProviderClient) CreateBuildingBlock(bb *MeshBuildingBlockCreate) (*MeshBuildingBlock, error) {
	return unmarshalBody[MeshBuildingBlock](c.doAuthenticatedRequest("POST", c.endpoints.BuildingBlocks,
		withPayload(bb, CONTENT_TYPE_BUILDING_BLOCK),
	))
}

func (c *MeshStackProviderClient) DeleteBuildingBlock(uuid string) error {
	targetUrl := c.urlForBuildingBlock(uuid)
	return c.deleteMeshObject(targetUrl, 202)
}
