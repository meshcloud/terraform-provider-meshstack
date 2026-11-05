package provider

import (
	"encoding/json/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDefinitionsDataSource(t *testing.T) {
	t.Run("simple state check", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		dataSourceAddress := "data.meshstack_building_block_definitions.example"

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_definitions/simple_state_check", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/building_block_definitions/simple_state_check", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("workspace_identifier"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":               xknownvalue.NotEmptyString(),
								"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
							}),
							"spec": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"display_name": knownvalue.StringExact("Example Building Block"),
								"target_type":  knownvalue.StringExact("WORKSPACE_LEVEL"),
							}),
							"ref": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"kind": knownvalue.StringExact("meshBuildingBlockDefinition"),
								"uuid": xknownvalue.NotEmptyString(),
							}),
							"version_latest": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":         xknownvalue.NotEmptyString(),
								"number":       knownvalue.Int64Exact(1),
								"state":        knownvalue.StringExact("DRAFT"),
								"content_hash": xknownvalue.NotEmptyString(),
							}),
							"versions": ascendingVersionsExact(
								knownvalue.ObjectPartial(map[string]knownvalue.Check{
									"uuid":         xknownvalue.NotEmptyString(),
									"number":       knownvalue.Int64Exact(1),
									"state":        knownvalue.StringExact("DRAFT"),
									"content_hash": xknownvalue.NotEmptyString(),
								}),
							),
							"version_latest_release": knownvalue.Null(),
						}),
					})),
				},
			},
		}})
	})

	t.Run("cross-workspace listing", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		buildingBlockDefinitionAddr := "meshstack_building_block_definition.example_03_manual"
		dataSourceAddress := "data.meshstack_building_block_definitions.example"
		captureReaderKey, readerKey := otherWorkspaceReader()

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/building_block_definitions/cross_workspace_listing", 1, vars),
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionAddr, tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
				}, captureReaderKey...),
			},
			{
				Config:          hclStep(t, "data-sources/building_block_definitions/cross_workspace_listing", 2, vars),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectPartial(func() map[string]knownvalue.Check {
							versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":         xknownvalue.NotEmptyString(),
								"state":        knownvalue.StringExact("RELEASED"),
								"content_hash": knownvalue.Null(),
							})
							return map[string]knownvalue.Check{
								"version_latest":         versionRef,
								"version_latest_release": versionRef,
								"versions":               ascendingVersionsExact(versionRef),
							}
						}()),
					})),
				},
			},
		}})
	})

	t.Run("workload identity of each version", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		bbdAddr := "meshstack_building_block_definition.example_01_terraform"
		dataSourceAddress := "data.meshstack_building_block_definitions.example"

		listedPath := tfjsonpath.New("building_block_definitions").AtSliceIndex(0)
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				// The workspace also holds the definition's dependency, which the filter leaves out.
				Config: hclStep(t, "data-sources/building_block_definitions/workload_identity_of_each_version", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("building_block_definitions"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceAddress, listedPath.AtMapKey("version_latest").AtMapKey("workload_identity_federation"),
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"issuer":  knownvalue.StringExact("https://oidc.example.com"),
							"subject": resolvedSubject("system:serviceaccount:namespace:workspace."),
						})),
					statecheck.CompareValuePairs(
						bbdAddr, tfjsonpath.New("versions").AtMapKey("v1").AtMapKey("workload_identity_federation"),
						dataSourceAddress, listedPath.AtMapKey("versions").AtMapKey("v1").AtMapKey("workload_identity_federation"),
						compare.ValuesSame(),
					),
				},
			},
		}})
	})

	t.Run("cross-workspace workload identity", func(t *testing.T) {
		if IsMockClientTest() {
			t.Skip("cross-workspace test requires real permission boundaries")
		}

		vars := hclStepVars{"suffix": acctest.RandString(8)}
		bbdAddr := "meshstack_building_block_definition.example_01_terraform"
		dataSourceAddress := "data.meshstack_building_block_definitions.example"
		captureReaderKey, readerKey := otherWorkspaceReader()

		listedPath := tfjsonpath.New("building_block_definitions").AtSliceIndex(0)
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				// Another workspace only sees released versions, through the redacted status of the definition.
				Config:            hclStep(t, "data-sources/building_block_definitions/cross_workspace_workload_identity", 1, vars),
				ConfigStateChecks: captureReaderKey,
			},
			{
				Config:          hclStep(t, "data-sources/building_block_definitions/cross_workspace_workload_identity", 2, vars),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, listedPath.AtMapKey("version_latest_release").AtMapKey("content_hash"), knownvalue.Null()),
					statecheck.CompareValuePairs(
						bbdAddr, tfjsonpath.New("version_latest_release").AtMapKey("workload_identity_federation"),
						dataSourceAddress, listedPath.AtMapKey("version_latest_release").AtMapKey("workload_identity_federation"),
						compare.ValuesSame(),
					),
					statecheck.ExpectKnownValue(dataSourceAddress, listedPath.AtMapKey("version_latest_release").AtMapKey("workload_identity_federation").AtMapKey("subject"),
						resolvedSubject("system:serviceaccount:namespace:workspace.")),
				},
			},
		}})
	})
}

func otherWorkspaceReader() (captureKey []statecheck.StateCheck, keyVariables tfconfig.Variables) {
	const apiKeyAddress = "meshstack_api_key.reader"
	clientId, clientSecret := new(lazyVariable), new(lazyVariable)
	capture := func(target *lazyVariable) knownvalue.Check {
		return xknownvalue.NotEmptyString(func(value string) error {
			*target = lazyVariable(value)
			return nil
		})
	}
	captureKey = []statecheck.StateCheck{
		statecheck.ExpectKnownValue(apiKeyAddress, tfjsonpath.New("status").AtMapKey("client_id"), capture(clientId)),
		statecheck.ExpectKnownValue(apiKeyAddress, tfjsonpath.New("status").AtMapKey("client_secret"), capture(clientSecret)),
	}
	return captureKey, tfconfig.Variables{
		"apikey_client_id":     clientId,
		"apikey_client_secret": clientSecret,
	}
}

type lazyVariable string

func (l *lazyVariable) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(*l))
}

// Test_buildVersionRefsFromStatus covers the fallback for a definition whose version specs meshStack withholds:
// the version references have to come from the definition status alone.
func Test_buildVersionRefsFromStatus(t *testing.T) {
	const definitionUuid, releasedUuid, draftUuid = "definition-uuid", "released-uuid", "draft-uuid"
	consumableDefinition := func(status *client.MeshBuildingBlockDefinitionStatus) client.MeshBuildingBlockDefinition {
		return client.MeshBuildingBlockDefinition{
			Metadata: client.MeshBuildingBlockDefinitionMetadata{Uuid: new(definitionUuid), OwnedByWorkspace: "consumer-workspace"},
			Spec:     client.MeshBuildingBlockDefinitionSpec{DisplayName: "Consumable Building Block", TargetType: client.MeshBuildingBlockTypeTenantLevel.Unwrap()},
			Status:   status,
		}
	}

	t.Run("derives the version refs from the status", func(t *testing.T) {
		releasedWif := &client.MeshBuildingBlockDefinitionWif{
			Issuer:  "https://oidc.example.com",
			Subject: "system:serviceaccount:namespace:workspace.owner-workspace.buildingblockdefinition." + definitionUuid,
			Gcp:     &client.MeshBuildingBlockDefinitionCloudWif{Audience: "gcp-audience", TokenPath: "/var/run/secrets/workload-identity/token"},
		}
		released := client.MeshBuildingBlockDefinitionStatusVersion{VersionUuid: releasedUuid, VersionNumber: 1, State: client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap(), WorkloadIdentityFederation: releasedWif}
		draft := client.MeshBuildingBlockDefinitionStatusVersion{VersionUuid: draftUuid, VersionNumber: 2, State: client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()}
		definition := consumableDefinition(&client.MeshBuildingBlockDefinitionStatus{
			// Out of order on purpose: the data source sorts by version number.
			Versions:                  []client.MeshBuildingBlockDefinitionStatusVersion{draft, released},
			LatestVersion:             2,
			LatestVersionUuid:         draftUuid,
			LatestReleasedVersion:     new(int64(1)),
			LatestReleasedVersionUuid: new(releasedUuid),
			RedactedForNonOwnerAccess: true,
		})

		var diags diag.Diagnostics
		got := buildVersionRefsFromStatus(&diags, definition)

		require.Empty(t, diags)
		assert.Equal(t, buildingBlockDefinitionDataSourceModel{
			Metadata: buildingBlockDefinitionDataSourceMetadataModel{Uuid: definitionUuid, OwnedByWorkspace: "consumer-workspace"},
			Spec:     buildingBlockDefinitionDataSourceSpecModel{DisplayName: "Consumable Building Block", TargetType: client.MeshBuildingBlockTypeTenantLevel.Unwrap()},
			// The content hash needs the version spec, which is exactly what is withheld.
			Versions: map[string]buildingBlockDefinitionDataSourceVersionRefModel{
				"v1": {Uuid: releasedUuid, Number: 1, State: "RELEASED", WorkloadIdentityFederation: releasedWif},
				"v2": {Uuid: draftUuid, Number: 2, State: "DRAFT"},
			},
			VersionLatest:        buildingBlockDefinitionDataSourceVersionRefModel{Uuid: draftUuid, Number: 2, State: "DRAFT"},
			VersionLatestRelease: &buildingBlockDefinitionDataSourceVersionRefModel{Uuid: releasedUuid, Number: 1, State: "RELEASED", WorkloadIdentityFederation: releasedWif},
			Ref:                  newBuildingBlockDefinitionRef(definitionUuid),
		}, got)
	})

	t.Run("reports a missing status", func(t *testing.T) {
		var diags diag.Diagnostics
		buildVersionRefsFromStatus(&diags, consumableDefinition(nil))

		require.Len(t, diags, 1)
		assert.Equal(t, "Building block definition status missing", diags[0].Summary())
	})
}
