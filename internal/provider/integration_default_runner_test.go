package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccIntegrationDefaultRunner(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("a test cannot change the default runner of a real meshStack")
	}

	var mock clientmock.Client
	var newDefaultUuid string
	seedNewDefault := SeedingMock(func(seeded clientmock.Client) {
		mock = seeded
		runner, err := seeded.BuildingBlockRunner.Create(t.Context(), client.MeshBuildingBlockRunner{
			Metadata: client.MeshBuildingBlockRunnerMetadata{OwnedByWorkspace: "meshcloud"},
			Spec: client.MeshBuildingBlockRunnerSpec{
				DisplayName:        "meshStack Hosted Runner (Keycloak)",
				ImplementationType: string(client.MeshBuildingBlockRunnerImplementationTypeAll),
				Restriction:        new("PUBLIC"),
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		newDefaultUuid = *runner.Metadata.Uuid
	})
	isNewDefault := xknownvalue.NotEmptyString(func(actual string) error {
		if actual != newDefaultUuid {
			return fmt.Errorf("expected the new default runner %s, got %s", newDefaultUuid, actual)
		}
		return nil
	})

	vars := hclStepVars{"suffix": acctest.RandString(8)}
	config := hclStep(t, "resources/integration/default_runner", 1, vars)
	configWithHostedRunnerRef := hclStep(t, "resources/integration/01_github", 2, vars)
	integrationAddr := "meshstack_integration.example_github"
	runnerUuidPath := tfjsonpath.New("spec").AtMapKey("config").AtMapKey("github").AtMapKey("runner_ref").AtMapKey("uuid")

	ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
		{
			PreConfig: func() { mock.BuildingBlockRunner.MakeDefault(newDefaultUuid) },
			Config:    config,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(integrationAddr, runnerUuidPath, isNewDefault),
			}},
		},
		{
			PreConfig: func() { mock.BuildingBlockRunner.MakeDefault(SharedBuildingBlockRunnerUuid) },
			Config:    config,
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(integrationAddr, plancheck.ResourceActionNoop),
			}},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(integrationAddr, runnerUuidPath, isNewDefault),
			},
		},
		{
			Config: configWithHostedRunnerRef,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(integrationAddr, runnerUuidPath, knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
			},
		},
	}}, seedNewDefault)
}
