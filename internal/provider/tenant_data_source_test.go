package provider

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccTenantDataSource(t *testing.T) {
	// The singular meshstack_tenant data source resolves the tenant via the list endpoint by the real
	// platform identifier, which the mock (storing platform by ref uuid) cannot reproduce.
	if IsMockClientTest() {
		t.Skip("meshstack_tenant data source composite lookup requires a real meshStack")
	}

	suffix := acctest.RandString(8)
	vars := hclStepVars{"suffix": suffix}
	dataSourceAddress := "data.meshstack_tenant.name"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: hclStep(t, "data-sources/tenant", 1, vars),
			},
			{
				Config: hclStep(t, "data-sources/tenant", 2, vars),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("spec").AtMapKey("platform_ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("status").AtMapKey("tenant_name"), knownvalue.StringExact("test-ws-"+suffix+".test-proj-"+suffix+".custom-"+suffix+".global")),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("status").AtMapKey("platform_type_identifier"), knownvalue.StringExact("CUSTOM-PT-"+strings.ToUpper(suffix))),
				},
			},
		},
	})
}
