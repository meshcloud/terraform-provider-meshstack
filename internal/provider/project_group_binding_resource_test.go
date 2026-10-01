package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Addresses and the binding name of the blocks in
// examples/{resources,data-sources}/meshstack_project_group_binding/*-test-*.tf.
const (
	projectGroupBindingResourceAddr   = "meshstack_project_group_binding.example"
	projectGroupBindingDataSourceAddr = "data.meshstack_project_group_binding.example"
	projectBindingName                = "this-is-an-example"
)

func TestAccProjectGroupBinding(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user group 'my-user-group' in local meshStack")
	}

	config := examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "project_group_binding", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	suffix := acctest.RandString(8)
	vars := SuffixVariables(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectGroupBindingResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectGroupBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(projectBindingName)),
					statecheck.ExpectKnownValue(projectGroupBindingResourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
					statecheck.ExpectKnownValue(projectGroupBindingResourceAddr, tfjsonpath.New("target_ref").AtMapKey("name"), knownvalue.StringExact("test-proj-"+suffix)),
					statecheck.ExpectKnownValue(projectGroupBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("my-user-group")),
				},
			},
			{
				ResourceName:    projectGroupBindingResourceAddr,
				ImportState:     true,
				ImportStateId:   projectBindingName,
				ImportStateKind: resource.ImportBlockWithID,
				ConfigVariables: vars,
			},
		},
	})
}
