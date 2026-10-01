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

const projectDataSourceAddr = "data.meshstack_project.example"

func TestAccProjectDataSource(t *testing.T) {
	// The data source reads back the project the resource example creates, so the step applies both:
	// the project resource's step 1 with its prerequisites, and the data source variant pointing at it.
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "project", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	suffix := acctest.RandString(8)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: NewVariablesWithSuffix(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-proj-"+suffix)),
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
				},
			},
		},
	})
}
