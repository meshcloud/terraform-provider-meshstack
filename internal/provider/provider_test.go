package provider

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
)

// ProviderFactoriesForTest are used to instantiate a provider during
// acceptance testing. The factory function will be invoked for every Terraform
// CLI command executed to create a provider server to which the CLI can
// reattach.
func ProviderFactoriesForTest(opts ...providerOption) map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"meshstack": providerserver.NewProtocol6WithError(New("test", opts...)()),
	}
}

// IsMockClientTest returns true when TF_ACC is not set, meaning tests run with a mock client.
func IsMockClientTest() bool {
	return os.Getenv("TF_ACC") == ""
}

// ApplyAndTest runs a TF test case. When TF_ACC is not set, it uses a mock
// client (unit test mode). When TF_ACC is set, it runs against a real meshStack.
// All tests using ApplyAndTest run in parallel.
func ApplyAndTest(t *testing.T, testCase resource.TestCase) {
	t.Helper()

	if IsMockClientTest() {
		mockClient := clientmock.NewMock()
		testCase.IsUnitTest = true
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest(func(provider *MeshStackProvider) {
			provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
				return mockClient.AsClient(), nil
			}
		})
	} else {
		t.Parallel()
		testCase.PreCheck = func() { DefaultTestPreCheck(t) }
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest()
	}

	resource.Test(t, testCase)
}

func DefaultTestPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv(envKeyMeshstackEndpoint)
	require.Truef(t, strings.HasPrefix(endpoint, "http://localhost"),
		"Env %s='%s' does not start with http://localhost, only locally running meshStacks should be used for tests", envKeyMeshstackEndpoint, endpoint)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiKey), "Env %s empty, please set before running", envKeyMeshstackApiKey)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiSecret), "Env %s empty, please set before running", envKeyMeshstackApiSecret)
}
