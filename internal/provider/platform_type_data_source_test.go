package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccPlatformTypeDataSource(t *testing.T) {
	suffix := acctest.RandString(8)
	vars := platformTypeStepVars(suffix)
	dataSourceAddress := "data.meshstack_platform_type.example"
	platformTypeName := "TEST-PT-" + strings.ToUpper(suffix)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/platform_type", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/platform_type", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata"), checkPlatformTypeMetadata(platformTypeName, "test-ws-"+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("display_name"), knownvalue.StringExact("My Custom Platform "+suffix)),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("ref"), checkPlatformTypeRef(platformTypeName)),
				},
			},
		},
	})
}
