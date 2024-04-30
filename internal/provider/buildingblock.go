package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type String interface {
	string | types.String
}

type Bool interface {
	bool | types.Bool
}

type Int64 interface {
	int64 | types.Int64
}

type IMeshBuildingBlock[S String, B Bool, I Int64] struct {
	ApiVersion S                                   `json:"apiVersion" tfsdk:"api_version"`
	Kind       S                                   `json:"kind" tfsdk:"kind"`
	Metadata   IMeshBuildingBlockMetadata[S, B, I] `json:"metadata" tfsdk:"metadata"`
	Spec       IMeshBuildingBlockSpec[S]           `json:"spec" tfsdk:"spec"`
	Status     IMeshBuildingBlockStatus[S]         `json:"status" tfsdk:"status"`
}

type IMeshBuildingBlockMetadata[S String, B Bool, I Int64] struct {
	Uuid              S `json:"uuid" tfsdk:"uuid"`
	DefinitionUuid    S `json:"definitionUuid" tfsdk:"definition_uuid"`
	DefinitionVersion I `json:"definitionVersion" tfsdk:"definition_version"`
	TenantIdentifier  S `json:"tenantIdentifier" tfsdk:"tenant_identifier"`
	ForcePurge        B `json:"forcePurge" tfsdk:"force_purge"`
	CreatedOn         S `json:"createdOn" tfsdk:"created_on"`
	// FIXME: these should be null when unset but are currently ""
	MarkedForDeletionOn S `json:"markedForDeletionOn" tfsdk:"marked_for_deletion_on"`
	MarkedForDeletionBy S `json:"markedForDeletionBy" tfsdk:"marked_for_deletion_by"`
}

type IMeshBuildingBlockSpec[S String] struct {
	DisplayName          S                             `json:"displayName" tfsdk:"display_name"`
	Inputs               []IMeshBuildingBlockIO[S]     `json:"inputs" tfsdk:"inputs"`
	ParentBuildingBlocks []IMeshBuildingBlockParent[S] `json:"parentBuildingBlocks" tfsdk:"parent_building_blocks"`
}

type IMeshBuildingBlockIO[S String] struct {
	Key       S `json:"key" tfsdk:"key"`
	Value     S `json:"value" tfsdk:"value"`
	ValueType S `json:"valueType" tfsdk:"value_type"`
}

type IMeshBuildingBlockParent[S any] struct {
	BuildingBlockUuid S `json:"buildingBlockUuid" tfsdk:"buildingblock_uuid"`
	DefinitionUuid    S `json:"definitionUuid" tfsdk:"definition_uuid"`
}

type IMeshBuildingBlockStatus[S String] struct {
	Status  S                         `json:"status" tfsdk:"status"`
	Outputs []IMeshBuildingBlockIO[S] `json:"outputs" tfsdk:"outputs"`
}

type MeshBuildingBlock IMeshBuildingBlock[string, bool, int64]
