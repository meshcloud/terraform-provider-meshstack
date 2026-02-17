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

func TestServiceInstancesDataSource(t *testing.T) {
	// Run acceptance tests as unit tests with mock
	runServiceInstancesDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()

		// Pre-populate the mock store with service instances
		mockClient.ServiceInstance.Store["instance-1"] = &client.MeshServiceInstance{
			ApiVersion: "v1",
			Kind:       "meshServiceInstance",
			Metadata: client.MeshServiceInstanceMetadata{
				InstanceId:            "instance-1",
				OwnedByWorkspace:      "test-workspace",
				OwnedByProject:        "test-project",
				MarketplaceIdentifier: "test-marketplace",
			},
			Spec: client.MeshServiceInstanceSpec{
				Creator:     "test-user",
				DisplayName: "First Instance",
				PlanId:      "test-plan",
				ServiceId:   "test-service",
				Parameters:  map[string]clientTypes.Any{},
			},
		}

		mockClient.ServiceInstance.Store["instance-2"] = &client.MeshServiceInstance{
			ApiVersion: "v1",
			Kind:       "meshServiceInstance",
			Metadata: client.MeshServiceInstanceMetadata{
				InstanceId:            "instance-2",
				OwnedByWorkspace:      "test-workspace",
				OwnedByProject:        "test-project",
				MarketplaceIdentifier: "test-marketplace",
			},
			Spec: client.MeshServiceInstanceSpec{
				Creator:     "test-user",
				DisplayName: "Second Instance",
				PlanId:      "test-plan",
				ServiceId:   "test-service",
				Parameters:  map[string]clientTypes.Any{},
			},
		}

		testCase.Steps[0].PostApplyFunc = func() {
			// Verify the service instances are in the store
			assert.Len(t, mockClient.ServiceInstance.Store, 2)
			instance1, exists := mockClient.ServiceInstance.Store["instance-1"]
			require.True(t, exists)
			assert.Equal(t, "First Instance", instance1.Spec.DisplayName)
		}
	}))
}

func TestServiceInstancesDataSourceWithParameters(t *testing.T) {
	// Test that service instance parameters with different types are properly converted to JSON strings
	runServiceInstancesDataSourceTestCase(t, SetupMockClient(func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client) {
		t.Helper()

		// Pre-populate the mock store with service instances containing various parameter types
		mockClient.ServiceInstance.Store["instance-1"] = &client.MeshServiceInstance{
			ApiVersion: "v1",
			Kind:       "meshServiceInstance",
			Metadata: client.MeshServiceInstanceMetadata{
				InstanceId:            "instance-1",
				OwnedByWorkspace:      "test-workspace",
				OwnedByProject:        "test-project",
				MarketplaceIdentifier: "test-marketplace",
			},
			Spec: client.MeshServiceInstanceSpec{
				Creator:     "test-user",
				DisplayName: "Instance with Parameters",
				PlanId:      "test-plan",
				ServiceId:   "test-service",
				Parameters: map[string]clientTypes.Any{
					"string_param": "value",
					"number_param": 42,
					"bool_param":   true,
					"object_param": map[string]any{
						"key": "value",
					},
				},
			},
		}

		testCase.Steps[0].Check = resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.#", "1"),
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.metadata.instance_id", "instance-1"),
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.display_name", "Instance with Parameters"),
			// Verify parameters are properly converted to JSON
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.string_param", `"value"`),
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.number_param", "42"),
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.bool_param", "true"),
			resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.object_param", `{"key":"value"}`),
		)
	}))
}

func runServiceInstancesDataSourceTestCase(t *testing.T, modifiers ...ResourceTestCaseModifier) {
	t.Helper()

	config := examples.DataSource{Name: "service_instances"}.Config()

	testCase := resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
			},
		},
	}

	ResourceTestCaseModifiers(modifiers).ApplyAndTest(t, testCase)
}
