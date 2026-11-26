package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
)

func TestAccWorkspaceUserGroupsDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	onlyTheCreatedGroup := knownvalue.ListExact([]knownvalue.Check{
		knownvalue.ObjectPartial(map[string]knownvalue.Check{
			"ref": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"kind":               knownvalue.StringExact(client.MeshObjectKind.WorkspaceUserGroup),
				"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
				"name":               knownvalue.StringExact("group-" + suffix),
			}),
			"spec": knownvalue.ObjectPartial(map[string]knownvalue.Check{
				"display_name": knownvalue.StringExact("Group " + suffix),
				"egid":         knownvalue.StringExact("egid-" + suffix),
			}),
		}),
	})
	groups := tfjsonpath.New("workspace_user_groups")

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/workspace_user_groups", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_workspace_user_groups.by_egid", groups, onlyTheCreatedGroup),
					statecheck.ExpectKnownValue("data.meshstack_workspace_user_groups.by_display_name_in_other_case", groups, onlyTheCreatedGroup),
					statecheck.ExpectKnownValue("data.meshstack_workspace_user_groups.developers", groups, knownvalue.ListExact([]knownvalue.Check{})),
				},
			},
		},
	})
}
