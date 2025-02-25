package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	MESH_BUILDING_BLOCK_IO_TYPE_STRING        = "STRING"
	MESH_BUILDING_BLOCK_IO_TYPE_INTEGER       = "INTEGER"
	MESH_BUILDING_BLOCK_IO_TYPE_BOOLEAN       = "BOOLEAN"
	MESH_BUILDING_BLOCK_IO_TYPE_SINGLE_SELECT = "SINGLE_SELECT"
	MESH_BUILDING_BLOCK_IO_TYPE_FILE          = "FILE"
	MESH_BUILDING_BLOCK_IO_TYPE_LIST          = "LIST"

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

func (c *MeshStackProviderClient) urlForBuildingBlock(uuid string) *url.URL {
	return c.endpoints.BuildingBlocks.JoinPath(uuid)
}

func (c *MeshStackProviderClient) ReadBuildingBlock(uuid string) (*MeshBuildingBlock, error) {
	targetUrl := c.urlForBuildingBlock(uuid)

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_BUILDING_BLOCK)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, nil
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var bb MeshBuildingBlock
	err = json.Unmarshal(data, &bb)
	if err != nil {
		return nil, err
	}

	return &bb, nil
}

func (c *MeshStackProviderClient) CreateBuildingBlock(bb *MeshBuildingBlockCreate) (*MeshBuildingBlock, error) {
	payload, err := json.Marshal(bb)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.BuildingBlocks.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_BUILDING_BLOCK)
	req.Header.Set("Accept", CONTENT_TYPE_BUILDING_BLOCK)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdBb MeshBuildingBlock
	err = json.Unmarshal(data, &createdBb)
	if err != nil {
		return nil, err
	}

	return &createdBb, nil
}

func (c *MeshStackProviderClient) DeleteBuildingBlock(uuid string) error {
	targetUrl := c.urlForBuildingBlock(uuid)
	return c.deleteMeshObject(*targetUrl, 202)
}
