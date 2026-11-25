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
			{
				Config: hclStep(t, "data-sources/workspace", 3, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_workspace.this", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
					statecheck.ExpectKnownValue("data.meshstack_workspace.this", tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Workspace's Display Name")),
				},
			},
		},
	})
}

func TestWorkspaceDataSourceNameWinsOverProviderWorkspace(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("the acceptance run of TestAccWorkspaceDataSource covers the API")
	}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/workspace/provider_workspace", 1, nil),
			},
			{
				Config: hclStep(t, "data-sources/workspace/provider_workspace", 2, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_workspace.example", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
				},
			},
		},
	})
}

func TestWorkspaceDataSourceFromEnvironment(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("t.Setenv cannot run in the parallel acceptance run")
	}
	t.Setenv("MESHSTACK_WORKSPACE", "my-workspace")

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/workspace/environment", 1, nil),
			},
			{
				Config: hclStep(t, "data-sources/workspace/environment", 2, nil),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue("data.meshstack_workspace.this", tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
					statecheck.ExpectKnownValue("data.meshstack_workspace.this", tfjsonpath.New("ref").AtMapKey("name"), knownvalue.StringExact("my-workspace")),
				},
			},
		},
	})
}

func TestWorkspaceDataSourceWithoutWorkspace(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("t.Setenv cannot run in the parallel acceptance run")
	}
	t.Setenv("MESHSTACK_WORKSPACE", "")

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:      hclStep(t, "data-sources/workspace/no_workspace", 1, nil),
				ExpectError: expectErrorWords("No workspace to read", "metadata.name", "workspace", "MESHSTACK_WORKSPACE"),
			},
		},
	})
}
