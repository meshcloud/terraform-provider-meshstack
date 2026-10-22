package provider

import (
	"encoding/json/v2"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/compare"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockDefinitionsDataSource(t *testing.T) {
	t.Run("simple state check", func(t *testing.T) {
		buildingBlockDefinitionConfig, buildingBlockDefinitionAddr := testconfig.BBDManual(t)

		var dataSourceAddress testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_definitions"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockDefinitionAddr, "metadata", "owned_by_workspace")),
		).Join(buildingBlockDefinitionConfig)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("workspace_identifier"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
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

		buildingBlockDefinitionConfig, buildingBlockDefinitionAddr := testconfig.BBDManual(t)
		readerConfig, captureReaderKey, readerKey := otherWorkspaceReader(t)

		supportConfig := buildingBlockDefinitionConfig.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
		).Join(readerConfig)

		var dataSourceAddress testconfig.Traversal
		example := testconfig.DataSource{Name: "building_block_definitions"}
		config := example.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(buildingBlockDefinitionAddr, "metadata", "owned_by_workspace")),
			// provider alias meshstack-other has hardcoded config in _other_provider test support file with restricted apikey
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(
			// keep the support config as is (containing the BBD to be read)
			supportConfig,
			// but make the data source use a different provider using the api key from the other workspace
			// (credentials are passed in as variables in the second step)
			testconfig.OtherProviderConfig(t),
		)

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: supportConfig.String(),
				ConfigStateChecks: append([]statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionAddr.String(), tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
				}, captureReaderKey...),
			},
			{
				Config:          config.String(),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("building_block_definitions"), knownvalue.ListExact([]knownvalue.Check{
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
		bbdConfig, bbdAddr, _, _ := testconfig.BBDTerraformWithWifRunners(t, runnerPublicKey)

		var dataSourceAddress testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_definitions"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(bbdAddr, "metadata", "owned_by_workspace")),
			// The workspace also holds the definition's dependency, which the filter leaves out.
			testconfig.Descend("uuids")(testconfig.SetRawExpr("[%s]", bbdAddr.Join("metadata", "uuid"))),
		).Join(bbdConfig)

		listedPath := tfjsonpath.New("building_block_definitions").AtSliceIndex(0)
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("building_block_definitions"), knownvalue.ListSizeExact(1)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), listedPath.AtMapKey("version_latest").AtMapKey("workload_identity_federation"),
						knownvalue.ObjectPartial(map[string]knownvalue.Check{
							"issuer":  knownvalue.StringExact("https://oidc.example.com"),
							"subject": resolvedSubject("system:serviceaccount:namespace:workspace."),
						})),
					statecheck.CompareValuePairs(
						bbdAddr.String(), tfjsonpath.New("versions").AtMapKey("v1").AtMapKey("workload_identity_federation"),
						dataSourceAddress.String(), listedPath.AtMapKey("versions").AtMapKey("v1").AtMapKey("workload_identity_federation"),
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

		bbdConfig, bbdAddr, _, _ := testconfig.BBDTerraformWithWifRunners(t, runnerPublicKey)
		readerConfig, captureReaderKey, readerKey := otherWorkspaceReader(t)

		// Another workspace only sees released versions, through the redacted status of the definition.
		supportConfig := bbdConfig.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
		).Join(readerConfig)

		var dataSourceAddress testconfig.Traversal
		config := testconfig.DataSource{Name: "building_block_definitions"}.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("workspace_identifier")(testconfig.SetAddr(bbdAddr, "metadata", "owned_by_workspace")),
			testconfig.Descend("uuids")(testconfig.SetRawExpr("[%s]", bbdAddr.Join("metadata", "uuid"))),
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(supportConfig, testconfig.OtherProviderConfig(t))

		listedPath := tfjsonpath.New("building_block_definitions").AtSliceIndex(0)
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:            supportConfig.String(),
				ConfigStateChecks: captureReaderKey,
			},
			{
				Config:          config.String(),
				ConfigVariables: readerKey,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), listedPath.AtMapKey("version_latest_release").AtMapKey("content_hash"), knownvalue.Null()),
					statecheck.CompareValuePairs(
						bbdAddr.String(), tfjsonpath.New("version_latest_release").AtMapKey("workload_identity_federation"),
						dataSourceAddress.String(), listedPath.AtMapKey("version_latest_release").AtMapKey("workload_identity_federation"),
						compare.ValuesSame(),
					),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), listedPath.AtMapKey("version_latest_release").AtMapKey("workload_identity_federation").AtMapKey("subject"),
						resolvedSubject("system:serviceaccount:namespace:workspace.")),
				},
			},
		}})
	})
}

// otherWorkspaceReader builds a second workspace, renamed so it does not clash with the "example" one the
// definition builders create, holding an API key that may only list definitions. It returns that config,
// the state checks that capture the key once it exists, and the variables that hand the key to the
// meshstack-other provider of testconfig.OtherProviderConfig.
func otherWorkspaceReader(t *testing.T) (config testconfig.Config, captureKey []statecheck.StateCheck, keyVariables tfconfig.Variables) {
	t.Helper()
	var otherWorkspaceAddr testconfig.Traversal
	otherWorkspaceConfig, _ := testconfig.Workspace(t)
	otherWorkspaceConfig = otherWorkspaceConfig.WithFirstBlock(
		testconfig.RenameKey("other"),
		testconfig.ExtractAddress(&otherWorkspaceAddr),
	)
	apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, otherWorkspaceAddr)
	apiKeyConfig = apiKeyConfig.WithFirstBlock(
		testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["BUILDINGBLOCKDEFINITION_LIST"]`)),
	)

	clientId, clientSecret := new(lazyVariable), new(lazyVariable)
	capture := func(target *lazyVariable) knownvalue.Check {
		return xknownvalue.NotEmptyString(func(value string) error {
			*target = lazyVariable(value)
			return nil
		})
	}
	captureKey = []statecheck.StateCheck{
		statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), capture(clientId)),
		statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), capture(clientSecret)),
	}
	return otherWorkspaceConfig.Join(apiKeyConfig), captureKey, tfconfig.Variables{
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
