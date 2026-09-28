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
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in
// examples/{resources,data-sources}/meshstack_project_user_binding/*-test-*.tf. The binding name is
// projectBindingName, shared with the group binding example.
const (
	projectUserBindingResourceAddr   = "meshstack_project_user_binding.example"
	projectUserBindingDataSourceAddr = "data.meshstack_project_user_binding.example"
)

func TestAccProjectUserBinding(t *testing.T) {
	if !IsMockClientTest() {
		t.Skip("Skipping: requires user 'user@meshcloud.io' in local meshStack")
	}

	config := examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "project_user_binding", 1),
		examples.Resource.TestStepConfig(t, "project", 1, "prerequisites"),
	)

	vars := SuffixVariables(acctest.RandString(8))

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config:          config,
				ConfigVariables: vars,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(projectUserBindingResourceAddr, plancheck.ResourceActionCreate),
					},
				},
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(projectUserBindingResourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact(projectBindingName)),
					statecheck.ExpectKnownValue(projectUserBindingResourceAddr, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
					statecheck.ExpectKnownValue(projectUserBindingResourceAddr, tfjsonpath.New("target_ref").AtMapKey("name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(projectUserBindingResourceAddr, tfjsonpath.New("subject").AtMapKey("name"), knownvalue.StringExact("user@meshcloud.io")),
				},
			},
			{
				ResourceName:    projectUserBindingResourceAddr,
				ImportState:     true,
				ImportStateId:   projectBindingName,
				ImportStateKind: resource.ImportBlockWithID,
				// Required: the framework re-applies the prior step's config for the import plan.
				ConfigVariables: vars,
			},
		},
	})
}
