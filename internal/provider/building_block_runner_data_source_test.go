package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	testconfig "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	xknownvalue "github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccBuildingBlockRunnerDataSource(t *testing.T) {
	t.Parallel()

	t.Run("by_uuid", func(t *testing.T) {
		runnerConfig, runnerAddr, _ := testconfig.BuildingBlockRunnerWifAndWorkspace(t)
		runnerConfig = runnerConfig.WithFirstBlock(testconfig.Descend("spec", "public_key")(testconfig.SetString(runnerPublicKey)))
		dataSourceConfig, dataSourceAddr := testconfig.BuildingBlockRunnerDataSource(t, runnerAddr)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: dataSourceConfig.Join(runnerConfig).String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My GCP WIF Runner")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("implementation_type"), knownvalue.StringExact("TERRAFORM")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("restriction"), knownvalue.StringExact("PRIVATE")),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("workload_identity_federation"), xknownvalue.MapExact(map[string]knownvalue.Check{
							"issuer":           knownvalue.StringExact("https://oidc.example.com"),
							"subject_template": knownvalue.StringExact("system:serviceaccount:namespace:workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"),
							"gcp": xknownvalue.MapExact(map[string]knownvalue.Check{
								"audience":   knownvalue.StringExact("gcp-workload-identity-provider:namespace"),
								"token_path": knownvalue.StringExact("/var/run/secrets/workload-identity/token"),
							}),
							"aws":   knownvalue.Null(),
							"azure": knownvalue.Null(),
						})),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact(client.MeshObjectKind.BuildingBlockRunner)),
					},
				},
			},
		})
	})

	t.Run("shared_runner_by_default", func(t *testing.T) {
		config, dataSourceAddr := testconfig.SharedBuildingBlockRunnerDataSource(t)

		// Only the shape is asserted: the shared runner's display name and federation values come from
		// the meshStack under test, not from this configuration.
		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), knownvalue.StringExact(client.SharedBuildingBlockRunnerUuid)),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("ref").AtMapKey("uuid"), knownvalue.StringExact(client.SharedBuildingBlockRunnerUuid)),
					},
				},
			},
		})
	})
}
