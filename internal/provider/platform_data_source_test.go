package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_azure", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "data-sources/platform/01_azure", 1, vars),
				},
				{
					Config: hclStep(t, "data-sources/platform/01_azure", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.meshstack_platform.example", tfjsonpath.New("identifier"), knownvalue.StringExact("test-platform-"+suffix+".meshcloud-azure-dev")),
						statecheck.ExpectKnownValue("data.meshstack_platform.example", tfjsonpath.New("spec").AtMapKey("access_information"), knownvalue.StringExact("Login via [Azure Portal](https://portal.azure.com) using your corporate credentials.")),
					},
				},
			},
		})
	})

	// A data source reads its spec back from the backend, so every nested reference is read-only.
	// This asserts that `project_role_ref` inside aws_role_mappings — which the backend returns —
	// resolves to its {name, kind} on the data source (the read-only fix that motivated this: the
	// reference used to be wrongly declared Required on the data source schema).
	t.Run("02_aws", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "data-sources/platform/02_aws", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(
							"data.meshstack_platform.example",
							tfjsonpath.New("spec").AtMapKey("config").AtMapKey("aws").
								AtMapKey("replication").AtMapKey("aws_identity_store").AtMapKey("aws_role_mappings"),
							knownvalue.SetExact([]knownvalue.Check{
								xknownvalue.MapExact(map[string]knownvalue.Check{
									"project_role_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
										"name": knownvalue.StringExact("admin"),
										"kind": knownvalue.StringExact("meshProjectRole"),
									}),
									"aws_role":            knownvalue.StringExact("admin"),
									"permission_set_arns": knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("arn:aws:sso:::permissionSet/ssoins-1234567890abcdef/ps-1234567890abcdef")}),
								}),
							}),
						),
					},
				},
			},
		})
	})
}
