package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

const projectDataSourceAddr = "data.meshstack_project.example"

func TestAccProjectDataSource(t *testing.T) {
	// The data source reads back the project the resource example creates, so the step applies both:
	// the project resource's step 1 with its prerequisites, and the data source variant pointing at it.
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "project", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: projectConfigVariables(acctest.RandString(8)),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("owned_by_workspace"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(projectDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Project's Display Name")),
				},
			},
		},
	})
}
