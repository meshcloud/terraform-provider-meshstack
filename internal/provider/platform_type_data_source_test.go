package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestAccPlatformTypeDataSource(t *testing.T) {
	runPlatformTypeDataSourceTestCase(t)
}

func TestPlatformTypeDataSource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runPlatformTypeDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Len(t, mockClient.PlatformType.Store, 1)
		}
	}))
}

func runPlatformTypeDataSourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	var resourceAddress, dataSourceAddress, platformTypeName examples.Identifier
	var displayName string

	config := examples.DataSource{Name: "platform_type"}.Config().
		OwnedByAdminWorkspace().
		Join(PlatformTypeResourceConfigForTest(&resourceAddress, &platformTypeName, &displayName)).
		SingleResourceAddress(&dataSourceAddress).
		ReplaceAll(`name               = "OPENSHIFT-4"`, platformTypeName.Format(`name = "%s"`)).
		ReplaceAll(
			`data "meshstack_platform_type" "example" {`,
			resourceAddress.Format("data \"meshstack_platform_type\" \"example\" {\n  depends_on = [%s]\n"),
		)

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformTypeMetadata(platformTypeName.String())),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec"), checkPlatformTypeSpec(displayName)),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), checkPlatformTypeRef(platformTypeName.String())),
				},
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
