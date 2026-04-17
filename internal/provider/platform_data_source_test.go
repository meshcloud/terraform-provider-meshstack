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
)

func TestAccPlatformDataSource(t *testing.T) {
	platformConfig, platformAddr := testconfig.PlatformAndWorkspace(t, "_01_azure")
	config := testconfig.DataSource{Name: "platform"}.Config(t).
		WithFirstBlock(testconfig.Descend("metadata", "uuid")(testconfig.SetAddr(platformAddr, "metadata", "uuid"))).
		Join(platformConfig)

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
}
