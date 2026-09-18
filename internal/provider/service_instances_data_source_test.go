package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	clientTypes "github.com/meshcloud/meshstack-cli/client/types"

	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestServiceInstancesDataSource(t *testing.T) {
	t.Parallel()

	t.Run("lists every marketplace instance", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("no Terraform resource creates a service instance, so a real meshStack has none to list")
		}

		config := testconfig.DataSource{Name: "service_instances"}.Config(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.#", "2"),
					),
				},
			},
		}, SeedingMock(func(mock clientmock.Client) {
			mock.ServiceInstance.Store.Set("instance-1", serviceInstance("instance-1", "First Instance", nil))
			mock.ServiceInstance.Store.Set("instance-2", serviceInstance("instance-2", "Second Instance", nil))
		}))
	})

	t.Run("lists the parameters of an instance", func(t *testing.T) {
		if !IsMockClientTest() {
			t.Skip("no Terraform resource creates a service instance, so a real meshStack has none to list")
		}

		config := testconfig.DataSource{Name: "service_instances"}.Config(t)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					Check: resource.ComposeAggregateTestCheckFunc(
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.#", "1"),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.metadata.instance_id", "instance-1"),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.display_name", "Instance with Parameters"),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.string_param", `"value"`),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.number_param", "42"),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.bool_param", "true"),
						resource.TestCheckResourceAttr("data.meshstack_service_instances.all", "service_instances.0.spec.parameters.object_param", `{"key":"value"}`),
					),
				},
			},
		}, SeedingMock(func(mock clientmock.Client) {
			mock.ServiceInstance.Store.Set("instance-1", serviceInstance("instance-1", "Instance with Parameters", map[string]clientTypes.Any{
				"string_param": "value",
				"number_param": 42,
				"bool_param":   true,
				"object_param": map[string]any{
					"key": "value",
				},
			}))
		}))
	})
}
