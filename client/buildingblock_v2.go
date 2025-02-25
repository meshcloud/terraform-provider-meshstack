package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	CONTENT_TYPE_BUILDING_BLOCK_V2 = "application/vnd.meshcloud.api.meshbuildingblock.v2-preview.hal+json"
)

type MeshBuildingBlockV2 struct {
	ApiVersion string                      `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                      `json:"kind" tfsdk:"kind"`
	Metadata   MeshBuildingBlockV2Metadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshBuildingBlockV2Spec     `json:"spec" tfsdk:"spec"`
	Status     MeshBuildingBlockV2Status   `json:"status" tfsdk:"status"`
}

type MeshBuildingBlockV2Metadata struct {
	Uuid                string  `json:"uuid" tfsdk:"uuid"`
	OwnedByWorkspace    string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn           string  `json:"createdOn" tfsdk:"created_on"`
	MarkedForDeletionOn *string `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy *string `json:"markedForDeletionBy" tfsdk:"marked_for_deletion_by"`
}

type MeshBuildingBlockV2Spec struct {
	BuildingBlockDefinitionVersionRef MeshBuildingBlockV2DefinitionVersionRef `json:"buildingBlockDefinitionVersionRef" tfsdk:"building_block_definition_version_ref"`
	TargetRef                         MeshBuildingBlockV2TargetRef            `json:"targetRef" tfsdk:"target_ref"`
	DisplayName                       string                                  `json:"displayName" tfsdk:"display_name"`

	Inputs               []MeshBuildingBlockIO     `json:"inputs" tfsdk:"inputs"`
	ParentBuildingBlocks []MeshBuildingBlockParent `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type MeshBuildingBlockV2DefinitionVersionRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
}

type MeshBuildingBlockV2TargetRef struct {
	Kind       string  `json:"kind" tfsdk:"kind"`
	Uuid       *string `json:"uuid" tfsdk:"uuid"`
	Identifier *string `json:"identifier" tfsdk:"identifier"`
}

type MeshBuildingBlockV2Create struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                  `json:"kind" tfsdk:"kind"`
	Spec       MeshBuildingBlockV2Spec `json:"spec" tfsdk:"spec"`
}

type MeshBuildingBlockV2Status struct {
	Status     string                `json:"status" tfsdk:"status"`
	Outputs    []MeshBuildingBlockIO `json:"outputs" tfsdk:"outputs"`
	ForcePurge bool                  `json:"forcePurge" tfsdk:"force_purge"`
}

func (c *MeshStackProviderClient) ReadBuildingBlockV2(uuid string) (*MeshBuildingBlockV2, error) {
	targetUrl := c.urlForBuildingBlock(uuid)

	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_BUILDING_BLOCK_V2)

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

	var bb MeshBuildingBlockV2
	err = json.Unmarshal(data, &bb)
	if err != nil {
		return nil, err
	}

	return &bb, nil
}

func (c *MeshStackProviderClient) CreateBuildingBlockV2(bb *MeshBuildingBlockV2Create) (*MeshBuildingBlockV2, error) {
	payload, err := json.Marshal(bb)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.BuildingBlocks.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_BUILDING_BLOCK_V2)
	req.Header.Set("Accept", CONTENT_TYPE_BUILDING_BLOCK_V2)

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

	var createdBb MeshBuildingBlockV2
	err = json.Unmarshal(data, &createdBb)
	if err != nil {
		return nil, err
	}

	return &createdBb, nil
}

func (c *MeshStackProviderClient) DeleteBuildingBlockV2(uuid string) error {
	targetUrl := c.urlForBuildingBlock(uuid)
	return c.deleteMeshObject(*targetUrl, 202)
}
