package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestServiceInstanceDataSource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runServiceInstanceDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()

		// Pre-populate the mock store with a service instance
		instanceId := "test-instance-id"
		mockClient.ServiceInstance.Store[instanceId] = &client.MeshServiceInstance{
			ApiVersion: "v1",
			Kind:       "meshServiceInstance",
			Metadata: client.MeshServiceInstanceMetadata{
				InstanceId:            instanceId,
				OwnedByWorkspace:      "test-workspace",
				OwnedByProject:        "test-project",
				MarketplaceIdentifier: "test-marketplace",
			},
			Spec: client.MeshServiceInstanceSpec{
				Creator:     "test-user",
				DisplayName: "Test Service Instance",
				PlanId:      "test-plan",
				ServiceId:   "test-service",
				Parameters:  map[string]clientTypes.Any{},
			},
		}

		testCase.Steps[0].PostApplyFunc = func() {
			// Verify the service instance was read from the store
			assert.Len(t, mockClient.ServiceInstance.Store, 1)
			instance, exists := mockClient.ServiceInstance.Store[instanceId]
			require.True(t, exists)
			assert.Equal(t, "Test Service Instance", instance.Spec.DisplayName)
			assert.Equal(t, "test-workspace", instance.Metadata.OwnedByWorkspace)
		}
	}))
}

func TestServiceInstanceDataSourceWithParameters(t *testing.T) {
	// Test that service instance parameters with different types (number, string, bool, object, array)
	// are properly converted to JSON strings
	runServiceInstanceDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()

		// Pre-populate the mock store with a service instance containing various parameter types
		instanceId := "test-instance-id"
		mockClient.ServiceInstance.Store[instanceId] = &client.MeshServiceInstance{
			ApiVersion: "v1",
			Kind:       "meshServiceInstance",
			Metadata: client.MeshServiceInstanceMetadata{
				InstanceId:            instanceId,
				OwnedByWorkspace:      "test-workspace",
				OwnedByProject:        "test-project",
				MarketplaceIdentifier: "test-marketplace",
			},
			Spec: client.MeshServiceInstanceSpec{
				Creator:     "test-user",
				DisplayName: "Test Service Instance",
				PlanId:      "test-plan",
				ServiceId:   "test-service",
				Parameters: map[string]clientTypes.Any{
					"string_param":  "value",
					"number_param":  42,
					"float_param":   3.14,
					"boolean_param": true,
					"object_param": map[string]any{
						"nested_key": "nested_value",
						"count":      10,
					},
					"array_param": []any{"item1", "item2", 123},
					"null_param":  nil,
				},
			},
		}

		testCase.Steps[0].Check = resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "metadata.instance_id", instanceId),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.display_name", "Test Service Instance"),
			// Verify parameters are present and properly converted to JSON
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.string_param", `"value"`),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.number_param", "42"),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.float_param", "3.14"),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.boolean_param", "true"),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.object_param", `{"count":10,"nested_key":"nested_value"}`),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.array_param", `["item1","item2",123]`),
			resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.null_param", "null"),
		)
	}))
}

func runServiceInstanceDataSourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()

	config := examples.DataSource{Name: "service_instance"}.Config().
		ReplaceAll(`instance_id = "my-service-instance-id"`, `instance_id = "test-instance-id"`)

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
