package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/meshcloud/meshstack-cli/client"
	clientTypes "github.com/meshcloud/meshstack-cli/client/types"

	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

// A service instance is created by the marketplace, and no Terraform resource creates one, so an
// acceptance run has no instance whose id a config could name. Both subtests below therefore seed
// the mock store and skip in acceptance mode.
func TestServiceInstanceDataSource(t *testing.T) {
	t.Parallel()

	t.Run("reads a marketplace instance by its id", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("no Terraform resource creates a service instance, so a real meshStack has none to read")
		}

		instanceId := "test-instance-id"
		config := testconfig.DataSource{Name: "service_instance"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "instance_id")(testconfig.SetString(instanceId)))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
				},
			},
		}, SeedingMock(func(mock clientmock.Client) {
			mock.ServiceInstance.Store.Set(instanceId, serviceInstance(instanceId, "Test Service Instance", nil))
		}))
	})

	t.Run("reads the parameters the instance was ordered with", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("no Terraform resource creates a service instance, so a real meshStack has none to read")
		}

		instanceId := "test-instance-id"
		config := testconfig.DataSource{Name: "service_instance"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "instance_id")(testconfig.SetString(instanceId)))

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "metadata.instance_id", instanceId),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.display_name", "Test Service Instance"),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.string_param", `"value"`),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.number_param", "42"),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.float_param", "3.14"),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.boolean_param", "true"),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.object_param", `{"count":10,"nested_key":"nested_value"}`),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.array_param", `["item1","item2",123]`),
						resource.TestCheckResourceAttr("data.meshstack_service_instance.example", "spec.parameters.null_param", "null"),
					),
				},
			},
		}, SeedingMock(func(mock clientmock.Client) {
			mock.ServiceInstance.Store.Set(instanceId, serviceInstance(instanceId, "Test Service Instance", map[string]clientTypes.Any{
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
			}))
		}))
	})
}

// serviceInstance is the shape the marketplace stores, with only the id, the display name and the
// parameters left for a test to choose.
func serviceInstance(instanceId, displayName string, parameters map[string]clientTypes.Any) *client.MeshServiceInstance {
	if parameters == nil {
		parameters = map[string]clientTypes.Any{}
	}
	return &client.MeshServiceInstance{
		Metadata: client.MeshServiceInstanceMetadata{
			InstanceId:            instanceId,
			OwnedByWorkspace:      "test-workspace",
			OwnedByProject:        "test-project",
			MarketplaceIdentifier: "test-marketplace",
		},
		Spec: client.MeshServiceInstanceSpec{
			Creator:     "test-user",
			DisplayName: displayName,
			PlanId:      "test-plan",
			ServiceId:   "test-service",
			Parameters:  parameters,
		},
	}
}
