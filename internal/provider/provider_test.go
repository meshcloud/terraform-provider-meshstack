package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

// testAccProtoV6ProviderFactories are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
func testAccProtoV6ProviderFactories(opts ...providerOption) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"meshstack": providerserver.NewProtocol6WithError(New("test", opts...)()),
	}
}

func withMockClient(mockClient client.Client) providerOption {
	return func(s *MeshStackProvider) {
		s.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
			return mockClient, nil
		}
	}
}

func testAccPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv(envKeyMeshstackEndpoint)
	require.Truef(t, strings.HasPrefix(endpoint, "http://localhost"),
		"Env %s='%s' does not start with http://localhost, only locally running meshStacks should be used for tests", envKeyMeshstackEndpoint, endpoint)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiKey), "Env %s empty, please set before running", envKeyMeshstackApiKey)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiSecret), "Env %s empty, please set before running", envKeyMeshstackApiSecret)
}
