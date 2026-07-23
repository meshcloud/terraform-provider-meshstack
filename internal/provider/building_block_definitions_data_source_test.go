package provider

import (
	"encoding/json"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
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

		buildingBlockDefinitionConfig, buildingBlockDefinitionAddr := testconfig.BBDManual(t)

		// AS the BBDManual above already creates an "example" workspace, we need to rename the other workspace
		// and extract its correct address as well.
		// This other workspace will hold the API key used to list the BBD cross
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

		supportConfig := buildingBlockDefinitionConfig.WithFirstBlock(
			testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
		).Join(
			otherWorkspaceConfig,
			apiKeyConfig,
		)

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

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		s := supportConfig.String()
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: s,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(buildingBlockDefinitionAddr.String(), tfjsonpath.New("version_latest_release").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
				},
			},
			{
				Config: config.String(),
				ConfigVariables: tfconfig.Variables{
					// variables are hard-coded in test support file
					"apikey_client_id":     &apiKeyClientId,
					"apikey_client_secret": &apiKeyClientSecret,
				},
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
	return json.Marshal(*l)
}
