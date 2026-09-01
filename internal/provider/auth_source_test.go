package provider

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/credential"
	"github.com/meshcloud/meshstack-cli/pkg/diags"
	"github.com/meshcloud/meshstack-cli/pkg/meshstack"
	"github.com/meshcloud/meshstack-cli/pkg/profile"
	"github.com/meshcloud/meshstack-cli/pkg/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecret has the shape the CLI's credential check expects: 32 alphanumerics, no
// whitespace, not a UUID.
const testSecret = "abcdef0123456789abcdef0123456789"

// isolate points the resolution at an empty configuration directory and clears every
// MESHSTACK_* variable, so that a developer's own .env cannot decide what these tests see.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv(profile.ConfigDir.EnvKey, t.TempDir())
	for _, key := range []string{
		meshstack.Endpoint.EnvKey, meshstack.Workspace.EnvKey, profile.Name.EnvKey,
		credential.ApiKeyId.EnvKey, credential.ApiSecret.EnvKey, credential.ApiToken.EnvKey,
	} {
		t.Setenv(key, "")
	}
}

// A setting is identified by its EnvKey, so there is no second identifier to keep in step.
// Describe naming the attribute is what makes an origin readable.
func TestBlockSourceAnswersEverySettingItsAttributesCarry(t *testing.T) {
	block := blockSource{data: MeshStackProviderModel{
		Endpoint:  types.StringValue("https://api.example.com"),
		Profile:   types.StringValue("dev"),
		Workspace: types.StringValue("my-workspace"),
		ApiKey:    types.StringValue("an-id"),
		ApiSecret: types.StringValue("a-secret"),
		ApiToken:  types.StringValue("a-token"),
	}}

	tests := []struct {
		key       string
		want      string
		attribute string
	}{
		{key: meshstack.Endpoint.EnvKey, want: "https://api.example.com", attribute: "provider block endpoint"},
		{key: profile.Name.EnvKey, want: "dev", attribute: "provider block profile"},
		{key: meshstack.Workspace.EnvKey, want: "my-workspace", attribute: "provider block workspace"},
		{key: credential.ApiKeyId.EnvKey, want: "an-id", attribute: "provider block apikey"},
		{key: credential.ApiSecret.EnvKey, want: "a-secret", attribute: "provider block apisecret"},
		{key: credential.ApiToken.EnvKey, want: "a-token", attribute: "provider block apitoken"},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			value, ok := block.Lookup(test.key)
			require.True(t, ok)
			assert.Equal(t, test.want, value)
			assert.Equal(t, test.attribute, block.Describe(test.key))
		})
	}

	value, ok := blockSource{}.Lookup(meshstack.Endpoint.EnvKey)
	assert.False(t, ok, "an unset attribute must not silence the environment below it")
	assert.Empty(t, value)
}

// The block naming an api key with the secret in the environment is the normal
// non-interactive setup for this provider, not an edge case: the block has no apisecret in
// version control, and the environment carries no id.
func TestApiKeyInTheBlockPairsWithTheSecretInTheEnvironment(t *testing.T) {
	isolate(t)
	t.Setenv(meshstack.Endpoint.EnvKey, "https://api.example.com")
	t.Setenv(credential.ApiSecret.EnvKey, testSecret)

	session, err := auth.ResolveSession(t.Context(), auth.ResolveSessionOptions{
		Settings: blockSource{data: MeshStackProviderModel{ApiKey: types.StringValue("block-key")}},
	})

	require.NoError(t, err)
	assert.Equal(t, credential.MethodApiKey, session.Method())
	assert.Contains(t, session.Origins(), setting.Origin{
		Key: credential.ApiKeyId.EnvKey, Source: "provider block apikey",
	})
	assert.Contains(t, session.Origins(), setting.Origin{
		Key: credential.ApiSecret.EnvKey, Source: credential.ApiSecret.EnvKey,
	})
}

// A stale MESHSTACK_API_KEY beside the block's own apikey means the environment's secret
// belongs to a different id, so it is not borrowed. The refusal has to name the losing id, or
// it is worse than the 401 it replaces.
func TestAStaleApiKeyInTheEnvironmentDoesNotLendItsSecretToTheBlock(t *testing.T) {
	isolate(t)
	t.Setenv(meshstack.Endpoint.EnvKey, "https://api.example.com")
	t.Setenv(credential.ApiKeyId.EnvKey, "stale-key")
	t.Setenv(credential.ApiSecret.EnvKey, testSecret)

	_, err := auth.ResolveSession(t.Context(), auth.ResolveSessionOptions{
		Settings: blockSource{data: MeshStackProviderModel{ApiKey: types.StringValue("block-key")}},
	})

	require.ErrorIs(t, err, auth.ErrNoApiSecret)
	problem, ok := errors.AsType[diags.Problem](err)
	require.True(t, ok)
	assert.Contains(t, problem.Detail(), "provider block apikey")
	assert.Contains(t, problem.Detail(), "stale-key")
	assert.Contains(t, problem.Detail(), credential.ApiKeyId.EnvKey)
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
