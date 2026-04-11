package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccBuildingBlockDefinitionsDataSource(t *testing.T) {
	runBuildingBlockDefinitionsDataSourceTestCase(t)
}

func TestBuildingBlockDefinitionsDataSource(t *testing.T) {
	runBuildingBlockDefinitionsDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()

		const (
			workspaceIdentifier = "ws-a"
			definitionAUuid     = "bbd-a"
			definitionBUuid     = "bbd-b"
		)

		mockClient.BuildingBlockDefinition.Store[definitionBUuid] = &client.MeshBuildingBlockDefinition{
			Metadata: client.MeshBuildingBlockDefinitionMetadata{
				Uuid:             ptr.To(definitionBUuid),
				OwnedByWorkspace: workspaceIdentifier,
			},
			Spec: client.MeshBuildingBlockDefinitionSpec{
				DisplayName: "B",
				TargetType:  client.MeshBuildingBlockTypeWorkspaceLevel.Unwrap(),
			},
		}
		mockClient.BuildingBlockDefinition.Store[definitionAUuid] = &client.MeshBuildingBlockDefinition{
			Metadata: client.MeshBuildingBlockDefinitionMetadata{
				Uuid:             ptr.To(definitionAUuid),
				OwnedByWorkspace: workspaceIdentifier,
			},
			Spec: client.MeshBuildingBlockDefinitionSpec{
				DisplayName: "A",
				TargetType:  client.MeshBuildingBlockTypeWorkspaceLevel.Unwrap(),
			},
		}
		mockClient.BuildingBlockDefinition.Store["bbd-out-of-scope"] = &client.MeshBuildingBlockDefinition{
			Metadata: client.MeshBuildingBlockDefinitionMetadata{
				Uuid:             ptr.To("bbd-out-of-scope"),
				OwnedByWorkspace: "ws-b",
			},
			Spec: client.MeshBuildingBlockDefinitionSpec{
				DisplayName: "Ignored",
				TargetType:  client.MeshBuildingBlockTypeWorkspaceLevel.Unwrap(),
			},
		}

		mockClient.BuildingBlockDefinitionVersion.Store["bbd-a-v1"] = &client.MeshBuildingBlockDefinitionVersion{
			Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{Uuid: "bbd-a-v1", OwnedByWorkspace: workspaceIdentifier},
			Spec: client.MeshBuildingBlockDefinitionVersionSpec{
				BuildingBlockDefinitionRef: &client.BuildingBlockDefinitionRef{Kind: "meshBuildingBlockDefinition", Uuid: definitionAUuid},
				VersionNumber:              ptr.To(int64(1)),
				State:                      client.MeshBuildingBlockDefinitionVersionStateReleased.Ptr(),
				DeletionMode:               client.BuildingBlockDeletionModeDelete.Unwrap(),
				Implementation: client.MeshBuildingBlockDefinitionImplementation{
					Manual: &client.MeshBuildingBlockDefinitionManualImplementation{},
				},
			},
		}
		mockClient.BuildingBlockDefinitionVersion.Store["bbd-a-v2"] = &client.MeshBuildingBlockDefinitionVersion{
			Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{Uuid: "bbd-a-v2", OwnedByWorkspace: workspaceIdentifier},
			Spec: client.MeshBuildingBlockDefinitionVersionSpec{
				BuildingBlockDefinitionRef: &client.BuildingBlockDefinitionRef{Kind: "meshBuildingBlockDefinition", Uuid: definitionAUuid},
				VersionNumber:              ptr.To(int64(2)),
				State:                      client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
				DeletionMode:               client.BuildingBlockDeletionModeDelete.Unwrap(),
				Implementation: client.MeshBuildingBlockDefinitionImplementation{
					Manual: &client.MeshBuildingBlockDefinitionManualImplementation{},
				},
			},
		}
		mockClient.BuildingBlockDefinitionVersion.Store["bbd-b-v1"] = &client.MeshBuildingBlockDefinitionVersion{
			Metadata: client.MeshBuildingBlockDefinitionVersionMetadata{Uuid: "bbd-b-v1", OwnedByWorkspace: workspaceIdentifier},
			Spec: client.MeshBuildingBlockDefinitionVersionSpec{
				BuildingBlockDefinitionRef: &client.BuildingBlockDefinitionRef{Kind: "meshBuildingBlockDefinition", Uuid: definitionBUuid},
				VersionNumber:              ptr.To(int64(1)),
				State:                      client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr(),
				DeletionMode:               client.BuildingBlockDeletionModeDelete.Unwrap(),
				Implementation: client.MeshBuildingBlockDefinitionImplementation{
					Manual: &client.MeshBuildingBlockDefinitionManualImplementation{},
				},
			},
		}
	}))
}

func runBuildingBlockDefinitionsDataSourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()

	config := examples.DataSource{Name: "building_block_definitions"}.Config().
		ReplaceAll(`workspace_identifier = "my-workspace"`, `workspace_identifier = "ws-a"`)
	addr := "data.meshstack_building_block_definitions.example"

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("workspace_identifier"), knownvalue.StringExact("ws-a")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), knownvalue.StringExact("bbd-a")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(1).AtMapKey("metadata").AtMapKey("uuid"), knownvalue.StringExact("bbd-b")),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(0).AtMapKey("version_latest").AtMapKey("number"), knownvalue.Int64Exact(2)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(0).AtMapKey("version_latest_release").AtMapKey("number"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(1).AtMapKey("version_latest").AtMapKey("number"), knownvalue.Int64Exact(1)),
					statecheck.ExpectKnownValue(addr, tfjsonpath.New("building_block_definitions").AtSliceIndex(1).AtMapKey("version_latest_release"), knownvalue.Null()),
				},
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
