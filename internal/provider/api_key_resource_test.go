package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestApiKeyResource_Lifecycle(t *testing.T) {
	t.Parallel()

	mockClient := clientmock.NewMock()

	providerFactories := ProviderFactoriesForTest(func(provider *MeshStackProvider) {
		provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
			return mockClient.AsClient(), nil
		}
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "meshstack_api_key" "test" {
  workspace_identifier = "workspace-1"
  name                 = "my-api-key"
  authorities          = ["workspace.read", "project.read"]
  expiry_date          = "2030-12-31"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("meshstack_api_key.test", "uuid"),
					resource.TestCheckResourceAttrSet("meshstack_api_key.test", "token"),
					resource.TestCheckResourceAttr("meshstack_api_key.test", "workspace_identifier", "workspace-1"),
					resource.TestCheckResourceAttr("meshstack_api_key.test", "name", "my-api-key"),
					resource.TestCheckResourceAttrSet("meshstack_api_key.test", "created_on"),
				),
			},
			// Update: change name and expiry_date (does not recreate)
			{
				Config: `
resource "meshstack_api_key" "test" {
  workspace_identifier = "workspace-1"
  name                 = "my-api-key-updated"
  authorities          = ["workspace.read", "project.read"]
  expiry_date          = "2031-06-30"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("meshstack_api_key.test", "name", "my-api-key-updated"),
					resource.TestCheckResourceAttr("meshstack_api_key.test", "expiry_date", "2031-06-30"),
					// Token should still be set from create
					resource.TestCheckResourceAttrSet("meshstack_api_key.test", "token"),
				),
			},
		},
	})

	require.Equal(t, 1, mockClient.ApiKey.CreateCalls)
	require.GreaterOrEqual(t, mockClient.ApiKey.UpdateCalls, 1)
	require.Equal(t, 1, mockClient.ApiKey.DeleteCalls)
	require.Empty(t, mockClient.ApiKey.Store.Values())
}

func TestApiKeyResource_Import(t *testing.T) {
	t.Parallel()

	mockClient := clientmock.NewMock()

	providerFactories := ProviderFactoriesForTest(func(provider *MeshStackProvider) {
		provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
			return mockClient.AsClient(), nil
		}
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			// First create
			{
				Config: `
resource "meshstack_api_key" "test" {
  workspace_identifier = "workspace-1"
  name                 = "import-test-key"
  authorities          = ["workspace.read"]
  expiry_date          = "2030-12-31"
}
`,
			},
			// Import by UUID
			{
				ResourceName: "meshstack_api_key.test",
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["meshstack_api_key.test"]
					if !ok {
						return "", fmt.Errorf("resource not found")
					}
					return rs.Primary.Attributes["uuid"], nil
				},
				ImportStateVerify:                    true,
				ImportStateVerifyIdentifierAttribute: "uuid",
				ImportStateVerifyIgnore:              []string{"token"},
			},
		},
	})
}
