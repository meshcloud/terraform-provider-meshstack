package provider

import (
	"encoding/json/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in
// examples/{resources,data-sources}/meshstack_building_block_definition{,s}/*-test-*.tf.
const (
	manualBbdAddr                        = "meshstack_building_block_definition.example_03_manual"
	buildingBlockDefinitionsDataSourceAd = "data.meshstack_building_block_definitions.example"
)

// manualBbdStepConfig is the manual building block definition's step with the workspace owning it.
// Index 1 is the draft version the example declares, 2 the released one a cross-workspace listing
// needs.
func manualBbdStepConfig(t *testing.T, index int) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "building_block_definition", index),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

func TestAccBuildingBlockDefinitionsDataSource(t *testing.T) {
	t.Run("simple state check", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block_definitions", 1),
			manualBbdStepConfig(t, 1),
		)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionsDataSourceAd, tfjsonpath.New("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(buildingBlockDefinitionsDataSourceAd, tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"metadata": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":               xknownvalue.NotEmptyString(),
								"owned_by_workspace": xknownvalue.NotEmptyString(),
							}),
							"spec": knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"display_name": xknownvalue.NotEmptyString(),
								"target_type":  xknownvalue.NotEmptyString(),
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
							"versions": knownvalue.ListExact([]knownvalue.Check{
								knownvalue.ObjectPartial(map[string]knownvalue.Check{
									"uuid":         xknownvalue.NotEmptyString(),
									"number":       knownvalue.Int64Exact(1),
									"state":        knownvalue.StringExact("DRAFT"),
									"content_hash": xknownvalue.NotEmptyString(),
								}),
							}),
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

		vars := SuffixVariables(acctest.RandString(8))

		// Step 2 of the definition is the released one: a draft version is not visible to another
		// workspace. The consumer workspace holds the restricted key the listing runs under.
		supportConfig := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "building_block_definition", 2),
			examples.Resource.TestStepConfig(t, "api_key", 6),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites", "consumer-workspace"),
		)

		listConfig := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "building_block_definitions", 2),
			supportConfig,
			examples.Resource.TestSupportConfigs(t, "api_key", "other_provider"),
		)

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		listVars := tfconfig.Variables{
			"apikey_client_id":     &apiKeyClientId,
			"apikey_client_secret": &apiKeyClientSecret,
		}
		for name, value := range vars {
			listVars[name] = value
		}

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:          supportConfig,
				ConfigVariables: vars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(manualBbdAddr, tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
				},
			},
			{
				Config:          listConfig,
				ConfigVariables: listVars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionsDataSourceAd, tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
						knownvalue.ObjectPartial(func() map[string]knownvalue.Check {
							versionRef := knownvalue.ObjectPartial(map[string]knownvalue.Check{
								"uuid":         xknownvalue.NotEmptyString(),
								"state":        knownvalue.StringExact("RELEASED"),
								"content_hash": knownvalue.Null(),
							})
							return map[string]knownvalue.Check{
								"version_latest":         versionRef,
								"version_latest_release": versionRef,
								"versions":               knownvalue.ListExact([]knownvalue.Check{versionRef}),
							}
						}()),
					})),
				},
			},
		}})
	})
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
		released := client.MeshBuildingBlockDefinitionStatusVersion{VersionUuid: releasedUuid, VersionNumber: 1, State: client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap()}
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
			Versions: []buildingBlockDefinitionDataSourceVersionRefModel{
				{Uuid: releasedUuid, Number: 1, State: "RELEASED"},
				{Uuid: draftUuid, Number: 2, State: "DRAFT"},
			},
			VersionLatest:        buildingBlockDefinitionDataSourceVersionRefModel{Uuid: draftUuid, Number: 2, State: "DRAFT"},
			VersionLatestRelease: &buildingBlockDefinitionDataSourceVersionRefModel{Uuid: releasedUuid, Number: 1, State: "RELEASED"},
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
