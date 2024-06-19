package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
	Key       string      `json:"key" tfsdk:"key"`
	Value     interface{} `json:"value" tfsdk:"value"`
	ValueType string      `json:"valueType" tfsdk:"value_type"`
}

type MeshBuildingBlockParent struct {
	BuildingBlockUuid string `json:"buildingBlockUuid" tfsdk:"buildingblock_uuid"`
	DefinitionUuid    string `json:"definitionUuid" tfsdk:"definition_uuid"`
}

type MeshBuildingBlockStatus struct {
	Status  string                `json:"status" tfsdk:"status"`
	Outputs []MeshBuildingBlockIO `json:"outputs" tfsdk:"outputs"`
}

func (c *MeshStackProviderClient) ReadBuildingBlock(uuid string) (*MeshBuildingBlock, error) {
	if c.ensureValidToken() != nil {
		return nil, errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	targetUrl := c.endpoints.BuildingBlocks.JoinPath(uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}

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

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var bb MeshBuildingBlock
	err = json.Unmarshal(data, &bb)
	if err != nil {
		return nil, err
	}

	return &bb, nil
}
