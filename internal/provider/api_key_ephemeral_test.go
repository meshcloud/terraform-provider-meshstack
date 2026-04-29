package provider

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-testing/echoprovider"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

func TestApiKeyEphemeralResourceLifecycle(t *testing.T) {
	t.Parallel()

	mockClient := clientmock.NewMock()

	expiryDate := time.Now().UTC().Add(30 * time.Second).Format(time.RFC3339)
	config := fmt.Sprintf(`
ephemeral "meshstack_api_key" "example" {
  workspace_identifier = "workspace-1"
  name                 = "temporary-api-key"
  authorities          = ["workspace.read", "project.read"]
  expiry_date          = %q
}

provider "echo" {
  data = ephemeral.meshstack_api_key.example
}

resource "echo" "snapshot" {}
`, expiryDate)

	providerFactories := ProviderFactoriesForTest(func(provider *MeshStackProvider) {
		provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
			return mockClient.AsClient(), nil
		}
	})
	providerFactories["echo"] = echoprovider.NewProviderServer()

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("echo.snapshot", "data.workspace_identifier", "workspace-1"),
					resource.TestCheckResourceAttr("echo.snapshot", "data.name", "temporary-api-key"),
				),
			},
		},
	})

	require.GreaterOrEqual(t, mockClient.ApiKey.CreateCalls, 1)
	require.GreaterOrEqual(t, mockClient.ApiKey.UpdateCalls, 1)
	require.GreaterOrEqual(t, mockClient.ApiKey.DeleteCalls, 1)
	require.Empty(t, mockClient.ApiKey.Store.Values())
}
