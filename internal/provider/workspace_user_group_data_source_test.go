package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccWorkspaceUserGroupDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix, "seeded_username": seededUsername}
	dataSourceAddress := "data.meshstack_workspace_user_group.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/workspace_user_group", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/workspace_user_group", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My User Group")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("members"), knownvalue.SetExact([]knownvalue.Check{knownvalue.StringExact(seededUsername)})),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("status").AtMapKey("disabled_usernames"), knownvalue.SetExact([]knownvalue.Check{})),
				},
			},
		},
	})
}
