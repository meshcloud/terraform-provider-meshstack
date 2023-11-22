package provider

import "time"

type MeshBuildingBlock struct {
	ApiVersion string                    `json:"apiVersion"`
	Kind       string                    `json:"kind"`
	Metadata   MeshBuildingBlockMetadata `json:"metadata"`
	Spec       MeshBuildingBlockSpec     `json:"spec"`
	Status     MeshBuildingBlockStatus   `json:"status"`
}

type MeshBuildingBlockMetadata struct {
	Uuid                string    `json:"uuid"`
	DefinitionUuid      string    `json:"definitionUuid"`
	DefinitionVersion   int64     `json:"definitionVersion"`
	TenantIdentifier    string    `json:"tenantIdentifier"`
	CreatedOn           time.Time `json:"createdOn"`
	MarkedForDeletionOn time.Time `json:"markedForDeletionOn"`
	MarkedForDeletionBy string    `json:"markedForDeletionBy"`
}

type MeshBuildingBlockSpec struct {
	DisplayName          string                `json:"displayName"`
	Inputs               []MeshBuildingBlockIO `json:"inputs"`
	ParentBuildingBlocks []ParentBuildingBlock `json:"parentBuildingBlocks"`
}

type MeshBuildingBlockIO struct {
	Key       string `json:"key"`
	Value     any    `json:"value"`
	ValueType string `json:"valueType"`
}

type ParentBuildingBlock struct {
	BuildingBlockUuid string `json:"buildingBlockUuid"`
	DefinitionUuid    string `json:"definitionUuid"`
}

type MeshBuildingBlockStatus struct {
	Status  string                `json:"status"`
	Outputs []MeshBuildingBlockIO `json:"outputs"`
}
