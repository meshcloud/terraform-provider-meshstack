package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Address of the block in examples/{resources,data-sources}/meshstack_workspace/*-test-*.tf.
const (
	workspaceResourceAddr   = "meshstack_workspace.example"
	workspaceDataSourceAddr = "data.meshstack_workspace.example"
)

func TestAccWorkspace(t *testing.T) {
	t.Run("superset_tag_reconciliation", func(t *testing.T) {
		// The backend returns a tag superset (an empty-list entry for every defined tag property, even
		// undeclared ones); the mock has no tag-schema logic and can't reproduce it. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend returning an entry for every defined tag property")
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites", "undeclared-tag"),
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the single declared tag remains; the undeclared property's empty-list
						// superset entry was reconciled away.
						statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	suffix := acctest.RandString(8)
	vars := SuffixVariables(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceResourceAddr, plancheck.ResourceActionCreate),
						// `kind` is the single constant value, so it is known already at plan time;
						// only the identifier is computed on create.
						plancheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
						plancheck.ExpectUnknownValue(workspaceResourceAddr, tfjsonpath.New("ref").AtMapKey("name")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Metadata
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("metadata").AtMapKey("created_on"), xknownvalue.NotEmptyString()),

					// Spec
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),

					// Ref
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
				},
			},
			{
				Config:          examples.Resource.TestStepConfig(t, "workspace", 2, "variables", "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[workspaceResourceAddr]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", workspaceResourceAddr)
					}
					return rs.Primary.Attributes["ref.name"], nil
				},
				ResourceName: workspaceResourceAddr,
			},
		},
	})
}
