package provider

import (
	"fmt"
	"regexp"
	"strconv"
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

func TestAccBuildingBlockDefinitionDefaultRunner(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("a test cannot change the default runner of a real meshStack")
	}

	const defaultRunnerIssuer = "https://keycloak.example.com/realms/meshstack"
	var mock clientmock.Client
	var newDefaultUuid string
	seedNewDefault := SeedingMock(func(seeded clientmock.Client) {
		mock = seeded
		runner, err := seeded.BuildingBlockRunner.Create(t.Context(), client.MeshBuildingBlockRunner{
			Metadata: client.MeshBuildingBlockRunnerMetadata{OwnedByWorkspace: "meshcloud"},
			Spec: client.MeshBuildingBlockRunnerSpec{
				DisplayName:        "meshStack Hosted Runner (Keycloak)",
				ImplementationType: client.MeshBuildingBlockRunnerImplementationTypeAll,
				Restriction:        new("PUBLIC"),
				WorkloadIdentityFederation: &client.MeshRunnerWorkloadIdentityFed{
					Issuer:          new(defaultRunnerIssuer),
					SubjectTemplate: new("workspace.{{ workspaceIdentifier }}.buildingblockdefinition.{{ buildingBlockDefinitionUuid }}"),
				},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		newDefaultUuid = *runner.Metadata.Uuid
	})
	makeDefault := func(runnerUuid func() string) func() {
		return func() { mock.BuildingBlockRunner.MakeDefault(runnerUuid()) }
	}
	newDefault := func() string { return newDefaultUuid }
	hosted := func() string { return SharedBuildingBlockRunnerUuid }

	suffix := acctest.RandString(8)
	config := func(draft bool) string {
		return hclStep(t, "resources/building_block_definition/default_runner", 1, hclStepVars{"suffix": suffix, "draft": strconv.FormatBool(draft)})
	}
	bbdAddr := "meshstack_building_block_definition.test"

	isNewDefault := xknownvalue.NotEmptyString(func(actual string) error {
		if actual != newDefaultUuid {
			return fmt.Errorf("expected the new default runner %s, got %s", newDefaultUuid, actual)
		}
		return nil
	})
	runnerUuidPath := tfjsonpath.New("version_spec").AtMapKey("runner_ref").AtMapKey("uuid")
	latestWifPath := tfjsonpath.New("version_latest").AtMapKey("workload_identity_federation")

	ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
		{
			Config: config(true),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(bbdAddr, runnerUuidPath, knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
			}},
		},
		{
			PreConfig: makeDefault(newDefault),
			Config:    config(true),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionUpdate),
				plancheck.ExpectKnownValue(bbdAddr, runnerUuidPath, isNewDefault),
				plancheck.ExpectUnknownValue(bbdAddr, latestWifPath),
			}},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(bbdAddr, latestWifPath.AtMapKey("issuer"), knownvalue.StringExact(defaultRunnerIssuer)),
			},
		},
		{
			PreConfig:   makeDefault(hosted),
			Config:      config(false),
			ExpectError: regexp.MustCompile("Cannot change the runner and release the draft in one apply"),
		},
		{
			Config: config(true),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(bbdAddr, runnerUuidPath, knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
			}},
		},
		{
			Config: config(false),
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(bbdAddr, tfjsonpath.New("version_latest").AtMapKey("state"), knownvalue.StringExact("RELEASED")),
				statecheck.ExpectKnownValue(bbdAddr, runnerUuidPath, knownvalue.StringExact(SharedBuildingBlockRunnerUuid)),
			},
		},
		{
			PreConfig: makeDefault(newDefault),
			Config:    config(false),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectResourceAction(bbdAddr, plancheck.ResourceActionNoop),
			}},
		},
		{
			Config: config(true),
			ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
				plancheck.ExpectKnownValue(bbdAddr, runnerUuidPath, isNewDefault),
			}},
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownValue(bbdAddr, tfjsonpath.New("version_latest").AtMapKey("number"), knownvalue.Int64Exact(2)),
				statecheck.ExpectKnownValue(bbdAddr, runnerUuidPath, isNewDefault),
			},
		},
	}}, seedNewDefault)
}
