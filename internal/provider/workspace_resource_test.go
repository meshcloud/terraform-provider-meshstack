package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccWorkspace(t *testing.T) {
	t.Run("superset_tag_reconciliation", func(t *testing.T) {
		// The backend returns a tag superset (an empty-list entry for every defined tag property, even
		// undeclared ones); the mock has no tag-schema logic and can't reproduce it. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend returning an entry for every defined tag property")
		}

		config, wsAddr := testconfig.Workspace(t)
		// A second tag definition the workspace does not declare: the backend still returns it as an
		// empty list, so the fix must reconcile it away instead of surfacing it as drift.
		undeclaredTag, _, _ := testconfig.TagDefinition(t, client.MeshObjectKind.Workspace)
		config = config.Join(undeclaredTag)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the single declared tag remains; the undeclared property's empty-list
						// superset entry was reconciled away.
						statecheck.ExpectKnownValue(wsAddr.String(), tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapSizeExact(1)),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		})
	})

	config, resourceAddress := testconfig.Workspace(t)

	updateConfig := config.WithFirstBlock(
		testconfig.Descend("spec", "display_name")(testconfig.SetString("Updated Display Name")))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionCreate),
						// `kind` is the single constant value, so it is known already at plan time;
						// only the identifier is computed on create.
						plancheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
						plancheck.ExpectUnknownValue(resourceAddress.String(), tfjsonpath.New("ref").AtMapKey("name")),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					// Metadata
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("metadata").AtMapKey("created_on"), xknownvalue.NotEmptyString()),

					// Spec
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),

					// Ref
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: updateConfig.String(),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resourceAddress.String(), plancheck.ResourceActionUpdate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("Updated Display Name")),
				},
			},
			{
				ImportState:     true,
				ImportStateKind: resource.ImportBlockWithID,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs := s.RootModule().Resources[resourceAddress.String()]
					if rs == nil {
						return "", fmt.Errorf("resource not found: %s", resourceAddress.String())
					}
					return rs.Primary.Attributes["ref.name"], nil
				},
				ResourceName: resourceAddress.String(),
			},
		},
	})
}
