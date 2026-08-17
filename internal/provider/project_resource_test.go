package provider

import (
	"fmt"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
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

// Addresses and the tag key prefix of the blocks in
// examples/resources/meshstack_project/resource-test-*.tf and its test-support files.
const (
	projectResourceAddr          = "meshstack_project.example"
	projectWorkspaceResourceAddr = "meshstack_workspace.example"
	projectTagKeyPrefix          = "test-key-project-"
)

func TestAccProject(t *testing.T) {
	t.Run("restricted_default_tag", func(t *testing.T) {
		// Backend-materialized default: the mock has no tag-restriction business logic, so it can't
		// reproduce TagService.determineTags injecting a restricted tag's default on create. See the
		// lock-step policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		suffix := acctest.RandString(8)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          examples.Resource.TestStepConfig(t, "project", 3, "prerequisites", "restricted-tag"),
					ConfigVariables: projectConfigVariables(suffix),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("spec").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							projectTagKeyPrefix + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					// Refresh reads back the injected superset; reconcileTrackedTags must reconcile it
					// away so no drift remains.
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	vars := projectConfigVariables(acctest.RandString(8))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
				},
			},
			{
				Config:          examples.Resource.TestStepConfig(t, "project", 2, "prerequisites"),
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectResourceAddr, plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectResourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name")),
				},
			},
			{
				ResourceName:    projectResourceAddr,
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[projectResourceAddr]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", projectResourceAddr)
					}
					ws := s.RootModule().Resources[projectWorkspaceResourceAddr]
					if ws == nil {
						return "", fmt.Errorf("workspace resource not found: %s", projectWorkspaceResourceAddr)
					}
					return ws.Primary.Attributes["metadata.name"] + "." + rs.Primary.Attributes["metadata.name"], nil
				},
			},
		},
	})
}

func projectConfigVariables(suffix string) tfconfig.Variables {
	return tfconfig.Variables{"suffix": tfconfig.StringVariable(suffix)}
}
