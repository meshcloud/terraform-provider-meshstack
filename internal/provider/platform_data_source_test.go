package provider

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func platformDataSourceConfig(t *testing.T, suffix string) testconfig.Config {
	t.Helper()
	platformConfig, platformAddr := testconfig.PlatformAndWorkspace(t, suffix)
	return testconfig.DataSource{Name: "platform"}.Config(t).
		WithFirstBlock(testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid"))).
		Join(platformConfig)
}

func TestAccPlatformDataSource(t *testing.T) {
	t.Parallel()

	t.Run("01_azure", func(t *testing.T) {
		config := platformDataSourceConfig(t, "_01_azure")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue("data.meshstack_platform.example", tfjsonpath.New("identifier"), knownvalue.StringFunc(func(value string) error {
							parts := strings.SplitN(value, ".", 2)
							if len(parts) != 2 || !strings.HasPrefix(parts[0], "my-platform-") || parts[1] == "" {
								return fmt.Errorf("expected identifier format <platform>.<location>, got %q", value)
							}
							return nil
						})),
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
		config := platformDataSourceConfig(t, "_02_aws")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
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
