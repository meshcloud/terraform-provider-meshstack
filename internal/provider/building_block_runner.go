package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
)

var (
	SharedBuildingBlockRunnerUuids = map[enum.Entry[client.MeshBuildingBlockImplementationType]]string{
		client.MeshBuildingBlockImplementationTypeManual:              "46b7c17a-61f0-4062-9601-5785e60ce11f",
		client.MeshBuildingBlockImplementationTypeTerraform:           "66ddc814-1e69-4dad-b5f1-3a5bce51c01f",
		client.MeshBuildingBlockImplementationTypeGithubWorkflows:     "dc8c57a1-823f-4e96-8582-0275fa27dc7b",
		client.MeshBuildingBlockImplementationTypeGitlabPipeline:      "f4f4402b-f54d-4ab9-93ae-c07e997041e9",
		client.MeshBuildingBlockImplementationTypeAzureDevOpsPipeline: "05cfa85f-2818-4bdd-b193-620e0187d7de",
	}
)

func getSharedBuildingBlockRunnerRef(implementationType enum.Entry[client.MeshBuildingBlockImplementationType]) *client.BuildingBlockRunnerRef {
	return &client.BuildingBlockRunnerRef{
		Kind: "meshBuildingBlockRunner",
		Uuid: SharedBuildingBlockRunnerUuids[implementationType],
	}
}
