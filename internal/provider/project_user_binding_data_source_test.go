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

func TestAccProjectUserBindingDataSource(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user 'user@meshcloud.io' in local meshStack")
	}

	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "project_user_binding", 1),
		examples.Resource.TestStepConfig(t, "project_user_binding", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectUserBindingDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(projectBindingName)),
					statecheck.ExpectKnownValue(projectUserBindingDataSourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
				},
			},
		},
	})
}
