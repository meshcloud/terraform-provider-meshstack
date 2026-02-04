package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
	var resourceAddress, dataSourceAddress, platformName examples.Identifier

	config := examples.DataSource{Name: "platform"}.Config().
		Join(PlatformResourceConfigForTest(&resourceAddress, &platformName)).
		SingleResourceAddress(&dataSourceAddress).
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
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
