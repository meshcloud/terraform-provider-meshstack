package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccProjectUserBindingDataSource(t *testing.T) {
	testAccProjectBindingDataSource(t, "user", seededUsername, expectUserEnabled)
}

func TestAccProjectGroupBindingDataSource(t *testing.T) {
	testAccProjectBindingDataSource(t, "group", seededGroupName)
}

func testAccProjectBindingDataSource(t *testing.T, subjectKind, subjectName string, subjectChecks ...subjectStateCheck) {
	t.Helper()
	vars := hclStepVars{"suffix": acctest.RandString(8), "subject_kind": subjectKind, "subject_name": subjectName}
	bindingAddress := "meshstack_project_" + subjectKind + "_binding.test"
	dataSourceAddress := "data.meshstack_project_" + subjectKind + "_binding.example"

	expectFound := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("role_ref").AtMapKey("name"), knownvalue.StringExact("Project Reader")),
		statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("expiry_date"), knownvalue.StringExact("2099-12-31")),
		statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("expired"), knownvalue.Bool(false)),
	}
	expectFound = append(expectFound, subjectStateChecks(dataSourceAddress, subjectChecks)...)
	for _, attribute := range []string{"metadata", "target_ref", "subject"} {
		expectFound = append(expectFound, statecheck.CompareValuePairs(
			dataSourceAddress, tfjsonpath.New(attribute), bindingAddress, tfjsonpath.New(attribute), compare.ValuesSame()))
	}

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: hclStep(t, "data-sources/project_binding", 1, vars)},
			{Config: hclStep(t, "data-sources/project_binding", 2, vars), ConfigStateChecks: expectFound},
			{Config: hclStep(t, "data-sources/project_binding", 3, vars), ConfigStateChecks: expectFound},
			{
				Config:      hclStep(t, "data-sources/project_binding", 4, vars),
				ExpectError: expectErrorWords("binding", "matches"),
			},
		},
	})
}
