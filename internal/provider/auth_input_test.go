package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/auth/method"
	"github.com/meshcloud/meshstack-cli/pkg/diags"
	"github.com/meshcloud/meshstack-cli/pkg/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderInputNamesTheMethodItsAttributesImply pins the one thing this adapter decides on its
// own. A secret never travels in auth.Values, so pkg/auth learns that the block carries a complete
// credential from the method named here, and only then asks for the secret itself.
func TestProviderInputNamesTheMethodItsAttributesImply(t *testing.T) {
	tests := []struct {
		name string
		data MeshStackProviderModel
		want method.Method
	}{
		{name: "nothing set leaves the method to pkg/auth", data: MeshStackProviderModel{}},
		{
			name: "an api token is the manual method",
			data: MeshStackProviderModel{ApiToken: types.StringValue("a-token")},
			want: method.Manual,
		},
		{
			name: "a key without its secret is not a complete credential",
			data: MeshStackProviderModel{ApiKey: types.StringValue("an-id")},
		},
		{
			name: "a key with its secret is the apiKey method",
			data: MeshStackProviderModel{ApiKey: types.StringValue("an-id"), ApiSecret: types.StringValue("a-secret")},
			want: method.ApiKey,
		},
		{
			name: "an api token wins over a key, because pkg/auth ranks it first too",
			data: MeshStackProviderModel{
				ApiKey:    types.StringValue("an-id"),
				ApiSecret: types.StringValue("a-secret"),
				ApiToken:  types.StringValue("a-token"),
			},
			want: method.Manual,
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			input := &providerInput{data: testCase.data}
			assert.Equal(t, testCase.want, input.Explicit().Method)
		})
	}
}

func TestProviderInputCarriesTheBlockThroughToValues(t *testing.T) {
	input := &providerInput{data: MeshStackProviderModel{
		Endpoint:  types.StringValue("https://api.example.com"),
		Profile:   types.StringValue("dev"),
		Workspace: types.StringValue("my-workspace"),
		ApiKey:    types.StringValue("an-id"),
	}}
	assert.Equal(t, auth.Values{
		Profile:   "dev",
		Endpoint:  "https://api.example.com",
		Workspace: workspace.Name("my-workspace"),
		ApiKey:    "an-id",
	}, input.Explicit())
}

// TestApiKeySecretFallsBackToTheEnvironment pins the reason the fallback exists: pkg/auth has
// already decided the environment supplies the credential, so returning only the attribute would
// leave a CI job with a key it cannot mint from.
func TestApiKeySecretFallsBackToTheEnvironment(t *testing.T) {
	t.Setenv("MESHSTACK_API_SECRET", "from-the-environment")
	t.Setenv("MESHSTACK_API_TOKEN", "token-from-the-environment")

	t.Run("the attribute wins", func(t *testing.T) {
		input := &providerInput{data: MeshStackProviderModel{
			ApiSecret: types.StringValue("from-the-block"),
			ApiToken:  types.StringValue("token-from-the-block"),
		}}
		secret, err := input.ApiKeySecret(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "from-the-block", secret)
		token, err := input.ApiToken(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "token-from-the-block", token)
	})

	t.Run("an empty attribute falls back", func(t *testing.T) {
		input := &providerInput{}
		secret, err := input.ApiKeySecret(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "from-the-environment", secret)
		token, err := input.ApiToken(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "token-from-the-environment", token)
	})
}

// TestTheProviderHasNoBrowser is the guarantee the whole pkg/oidc split exists for, expressed as a
// test as well as a depguard rule: a terraform plan must never open one.
func TestTheProviderHasNoBrowser(t *testing.T) {
	assert.Nil(t, (&providerInput{}).Browser())
}

// TestProblemKeepsItsOwnSummary pins the reason problemDiagnostics inspects the error at all: a
// diags.Problem has already split the failure into a summary and an actionable paragraph, and
// flattening it into the caller's summary would put the paragraph where nobody reads it.
func TestProblemKeepsItsOwnSummary(t *testing.T) {
	fatal := problemDiagnostics("ignored", diags.Errorf("no workspace", "name one with the workspace attribute."))
	require.Len(t, fatal, 1)
	assert.Equal(t, diag.SeverityError, fatal[0].Severity())
	assert.Equal(t, "no workspace", fatal[0].Summary(), "a Problem keeps its own summary rather than the caller's")

	// An error that is not a Problem has no summary of its own, so the caller's is used.
	plain := problemDiagnostics("Failed to create meshStack client.", assert.AnError)
	require.Len(t, plain, 1)
	assert.Equal(t, "Failed to create meshStack client.", plain[0].Summary())
}
