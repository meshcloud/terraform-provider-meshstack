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

func TestAccWorkspaceDataSource(t *testing.T) {
	config := examples.JoinTestStepConfigs(
		examples.DataSource.TestStepConfig(t, "workspace", 1),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)

	suffix := acctest.RandString(8)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: SuffixVariables(suffix),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(workspaceDataSourceAddr, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(workspaceDataSourceAddr, tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceDataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("test-ws-"+suffix)),
					statecheck.ExpectKnownValue(workspaceDataSourceAddr, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),
				},
			},
		},
	})
}
