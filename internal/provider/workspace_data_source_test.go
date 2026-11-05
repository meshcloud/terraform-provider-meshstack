package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccWorkspaceDataSource(t *testing.T) {
	dataSourceAddress := "data.meshstack_workspace.example"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/workspace", 1, nil),
			},
			{
				Config: hclStep(t, "data-sources/workspace", 2, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref").AtMapKey("kind"), knownvalue.StringExact("meshWorkspace")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),
				},
			},
		},
	})
}
