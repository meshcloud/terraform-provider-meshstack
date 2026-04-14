package provider

import (
	_ "embed"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccPlatformDataSource(t *testing.T) {
	runPlatformDataSourceTestCase(t)
}

func TestPlatformDataSource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runPlatformDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.Platform.Store, 1)
		}
	}))
}

func runPlatformDataSourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	var resourceAddress, platformName examples.Identifier

	config := examples.DataSource{Name: "platform"}.Config().
		Join(PlatformResourceConfigForTest(&resourceAddress, &platformName)).
		ReplaceAll(`uuid = "d32951fc-6589-412f-b8bd-50c78fe2cb79"`, resourceAddress.Format(`uuid = %s.metadata.uuid`)).
		ReplaceAll(
			`data "meshstack_platform" "example" {`,
			resourceAddress.Format(`data "meshstack_platform" "example" {
  depends_on = [%s]
`),
		)

	testCase := resource.TestCase{
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
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
