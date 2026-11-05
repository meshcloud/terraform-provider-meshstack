package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccTagDefinitionDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	dataSourceAddr := "data.meshstack_tag_definition.example"

	expectFound := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("meshWorkspace.costCenter-"+suffix)),
		statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("spec").AtMapKey("key"), knownvalue.StringExact("costCenter-"+suffix)),
	}
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: hclStep(t, "data-sources/tag_definition", 1, vars)},
			{Config: hclStep(t, "data-sources/tag_definition", 2, vars), ConfigStateChecks: expectFound},
			{Config: hclStep(t, "data-sources/tag_definition", 3, vars), ConfigStateChecks: expectFound},
			{
				Config:      hclStep(t, "data-sources/tag_definition", 4, vars),
				ExpectError: expectErrorWords("matches", "spec.target_kind", "meshProject"),
			},
		},
	})
}
