package provider

import (
	"context"
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

// TestAccServiceInstancesDataSource tests the service instances data source.
// Service instances are read-only (no TF resource), so unit tests must pre-populate the mock store.
// We use resource.UnitTest directly instead of ApplyAndTest because pre-population requires
// access to the mock client before the test runs.
func TestServiceInstancesDataSource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		mockClient := clientmock.NewMock()
		mockClient.ServiceInstance.Store.Set("instance-1", &client.MeshServiceInstance{
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
		})
		mockClient.ServiceInstance.Store.Set("instance-2", &client.MeshServiceInstance{
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
		})

		config := testconfig.DataSource{Name: "service_instances"}.Config(t)

		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: ProviderFactoriesForTest(func(provider *MeshStackProvider) {
				provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
					return mockClient.AsClient(), nil
				}
			}),
			Steps: []resource.TestStep{
				{
					Config: config.String(),
				},
			},
		})
	})

	t.Run("with_parameters", func(t *testing.T) {
		t.Parallel()

		mockClient := clientmock.NewMock()
		mockClient.ServiceInstance.Store.Set("instance-1", &client.MeshServiceInstance{
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
		})

		config := testconfig.DataSource{Name: "service_instances"}.Config(t)

		resource.UnitTest(t, resource.TestCase{
			ProtoV6ProviderFactories: ProviderFactoriesForTest(func(provider *MeshStackProvider) {
				provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
					return mockClient.AsClient(), nil
				}
			}),
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
		})
	})
}
