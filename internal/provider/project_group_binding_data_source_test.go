package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccProjectGroupBindingDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user group 'my-user-group' in local meshStack")
	}

	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "project_group_binding", 1),
		examples.Resource.TestStepConfig(t, "project_group_binding", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: NewVariablesWithSuffix(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectGroupBindingDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(projectBindingName)),
					statecheck.ExpectKnownValue(projectGroupBindingDataSourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
				},
			},
		},
	})
}
