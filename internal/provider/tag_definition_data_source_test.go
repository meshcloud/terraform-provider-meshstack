package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccTagDefinitionDataSource(t *testing.T) {
	resourceConfig, tagDefinitionAddr, tagKey := testconfig.TagDefinition(t, "meshProject")
	dataSourceAddr := testconfig.Traversal{"data.meshstack_tag_definition", "example"}
	example := testconfig.DataSource{Name: "tag_definition"}.Config(t)

	byName := example.WithFirstBlock(
		testconfig.Descend("metadata")(testconfig.SetRawExpr("{ name = %s }", tagDefinitionAddr.Join("metadata", "name"))),
	).Join(resourceConfig)
	byTargetKindAndKey := example.WithFirstBlock(
		testconfig.Descend("metadata")(testconfig.RemoveKey()),
		testconfig.Descend("spec")(testconfig.SetRawExpr("{ target_kind = %s, key = %s }",
			tagDefinitionAddr.Join("spec", "target_kind"), tagDefinitionAddr.Join("spec", "key"))),
	).Join(resourceConfig)
	otherTargetKind := example.WithFirstBlock(
		testconfig.Descend("metadata")(testconfig.SetRawExpr("{ name = %s }", tagDefinitionAddr.Join("metadata", "name"))),
		testconfig.Descend("spec")(testconfig.SetRawExpr(`{ target_kind = "meshWorkspace" }`)),
	).Join(resourceConfig)

	expectFound := []statecheck.StateCheck{
		statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("metadata").AtMapKey("name"), knownvalue.StringExact("meshProject."+tagKey)),
		statecheck.ExpectKnownValue(dataSourceAddr.String(), tfjsonpath.New("spec").AtMapKey("key"), knownvalue.StringExact(tagKey)),
	}
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{Config: byName.String(), ConfigStateChecks: expectFound},
			{Config: byTargetKindAndKey.String(), ConfigStateChecks: expectFound},
			{
				Config:      otherTargetKind.String(),
				ExpectError: regexp.MustCompile(`(?s)No tag definition matches.*spec.target_kind\s+=\s+"meshWorkspace"`),
			},
		},
	})
}
