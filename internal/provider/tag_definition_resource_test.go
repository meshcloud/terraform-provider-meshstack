package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestTagDefinitionResource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runTagDefinitionResourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()
		testCase.Steps[0].PostApplyFunc = func() {
			assert.Equal(t, []string{"meshProject.example-key"}, mockClient.TagDefinition.Store.SortedKeys())
		}
	}))
}

func runTagDefinitionResourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()
	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: examples.Resource{Name: "tag_definition"}.Config().String(),
			},
		},
	}
	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
