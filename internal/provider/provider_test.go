package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/examples"
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
	testCase.ExternalProviders = map[string]resource.ExternalProvider{
		"random": {},
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

func KnownValueNotEmptyString(consumers ...func(actualValue string) error) knownvalue.Check {
	return knownvalue.StringFunc(func(v string) error {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("expected non-empty string after trimming whitespace, but is '%s'", v)
		}
		for _, consumer := range consumers {
			err := consumer(v)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func KnownValueRef(resourceAddress examples.Identifier, expectedKind string, uuidOut *string) statecheck.StateCheck {
	return statecheck.ExpectKnownValue(resourceAddress.String(), tfjsonpath.New("ref"), knownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact(expectedKind),
		"uuid": KnownValueNotEmptyString(func(actualValue string) error {
			if *uuidOut == "" {
				*uuidOut = actualValue
			} else if *uuidOut != actualValue {
				return fmt.Errorf("mismatching Resource UUID %s vs. %s, which should never change", *uuidOut, actualValue)
			}
			return nil
		}),
	}))
}
