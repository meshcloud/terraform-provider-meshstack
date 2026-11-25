package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccUserDataSource(t *testing.T) {
	vars := hclStepVars{"seeded_username": seededUsername}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/user", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_user.by_name", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(seededUsername)),
					statecheck.ExpectKnownValue("data.meshstack_user.by_name", tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue("data.meshstack_user.by_name", tfjsonpath.New("ref"), knownvalue.ObjectExact(map[string]knownvalue.Check{
						"kind": knownvalue.StringExact(client.MeshObjectKind.User),
						"name": knownvalue.StringExact(seededUsername),
					})),
					statecheck.ExpectKnownValue("data.meshstack_user.by_name", tfjsonpath.New("spec").AtMapKey("email"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue("data.meshstack_user.by_name", tfjsonpath.New("status").AtMapKey("lifecycle").AtMapKey("state"), knownvalue.StringExact("ACTIVE")),
					statecheck.CompareValuePairs(
						"data.meshstack_user.by_name", tfjsonpath.New("metadata").AtMapKey("uuid"),
						"data.meshstack_user.by_email", tfjsonpath.New("metadata").AtMapKey("uuid"),
						compare.ValuesSame(),
					),
				},
			},
		},
	})
}

func TestAccUsersDataSource(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8), "seeded_username": seededUsername}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/users", 1, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownOutputValue("seeded_user_listed", knownvalue.Bool(true)),
				},
			},
		},
	})
}
