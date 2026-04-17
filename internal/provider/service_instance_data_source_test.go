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

// TestAccServiceInstanceDataSource tests the service instance data source.
// Service instances are read-only (no TF resource), so unit tests must pre-populate the mock store.
// We use resource.UnitTest directly instead of ApplyAndTest because pre-population requires
// access to the mock client before the test runs.
func TestServiceInstanceDataSource(t *testing.T) {
	t.Parallel()

	t.Run("basic", func(t *testing.T) {
		t.Parallel()

		instanceId := "test-instance-id"
		mockClient := clientmock.NewMock()
		mockClient.ServiceInstance.Store.Set(instanceId, &client.MeshServiceInstance{
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
		})

		config := testconfig.DataSource{Name: "service_instance"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "instance_id")(testconfig.SetString(instanceId)))

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

		instanceId := "test-instance-id"
		mockClient := clientmock.NewMock()
		mockClient.ServiceInstance.Store.Set(instanceId, &client.MeshServiceInstance{
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
		})

		config := testconfig.DataSource{Name: "service_instance"}.Config(t).WithFirstBlock(
			testconfig.Descend("metadata", "instance_id")(testconfig.SetString(instanceId)))

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
		})
	})
}
