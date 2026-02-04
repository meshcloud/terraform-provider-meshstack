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
	"github.com/meshcloud/terraform-provider-meshstack/internal/clientmock"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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

type ResourceTestCaseModifier func(t *testing.T, testCase *resource.TestCase)

type ResourceTestCaseModifiers []ResourceTestCaseModifier

func (m ResourceTestCaseModifiers) ApplyAndTest(t *testing.T, testCase resource.TestCase) {
	t.Helper()
	for _, modifier := range m {
		modifier(t, &testCase)
	}
	if testCase.ProtoV6ProviderFactories == nil {
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest()
	}
	resource.Test(t, testCase)
}

func SetupMockClient(additions ...func(t *testing.T, testCase *resource.TestCase, mockClient clientmock.Client)) ResourceTestCaseModifier {
	return func(t *testing.T, testCase *resource.TestCase) {
		t.Helper()
		mockClient := clientmock.NewMock()
		testCase.IsUnitTest = true
		testCase.ProtoV6ProviderFactories = ProviderFactoriesForTest(func(provider *MeshStackProvider) {
			provider.clientFactory = func(ctx context.Context, data MeshStackProviderModel, providerVersion string) (client.Client, diag.Diagnostics) {
				return mockClient.AsClient(), nil
			}
		})
		for _, addition := range additions {
			addition(t, testCase, mockClient)
		}
	}
}

func DefaultTestPreCheck(t *testing.T) {
	t.Helper()
	endpoint := os.Getenv(envKeyMeshstackEndpoint)
	require.Truef(t, strings.HasPrefix(endpoint, "http://localhost"),
		"Env %s='%s' does not start with http://localhost, only locally running meshStacks should be used for tests", envKeyMeshstackEndpoint, endpoint)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiKey), "Env %s empty, please set before running", envKeyMeshstackApiKey)
	require.NotEmptyf(t, os.Getenv(envKeyMeshstackApiSecret), "Env %s empty, please set before running", envKeyMeshstackApiSecret)
}
