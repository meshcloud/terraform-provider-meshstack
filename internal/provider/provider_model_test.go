package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The setting keys are literals, because taking them from the meshStack CLI's declarations would
// make the test agree with whatever mapping the model happens to carry.
func Test_MeshStackProviderModel_LookupAnswersOnlyItsOwnSettingKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		settingKey   string
		attributeKey string
		model        MeshStackProviderModel
	}{
		{"MESHSTACK_ENDPOINT", "endpoint", MeshStackProviderModel{Endpoint: types.StringValue("set")}},
		{"MESHSTACK_PROFILE", "profile", MeshStackProviderModel{Profile: types.StringValue("set")}},
		{"MESHSTACK_WORKSPACE", "workspace", MeshStackProviderModel{Workspace: types.StringValue("set")}},
		{"MESHSTACK_API_KEY", "apikey", MeshStackProviderModel{ApiKey: types.StringValue("set")}},
		{"MESHSTACK_API_SECRET", "apisecret", MeshStackProviderModel{ApiSecret: types.StringValue("set")}},
		{"MESHSTACK_API_TOKEN", "apitoken", MeshStackProviderModel{ApiToken: types.StringValue("set")}},
	}
	for _, tt := range tests {
		t.Run(tt.settingKey, func(t *testing.T) {
			t.Parallel()

			value, err := tt.model.Lookup(t.Context(), tt.settingKey)

			require.NoError(t, err)
			assert.Equal(t, "set", value)
			assert.Equal(t, "provider block attribute '"+tt.attributeKey+"'", tt.model.Describe(tt.settingKey))

			for _, other := range tests {
				if other.settingKey == tt.settingKey {
					continue
				}
				otherValue, otherErr := tt.model.Lookup(t.Context(), other.settingKey)

				require.NoError(t, otherErr)
				assert.Emptyf(t, otherValue, "%s answered the %s lookup", tt.attributeKey, other.settingKey)
			}
		})
	}
}

func Test_MeshStackProviderModel_LookupIgnoresASettingTheBlockCannotExpress(t *testing.T) {
	t.Parallel()

	model := MeshStackProviderModel{Endpoint: types.StringValue("set")}

	value, err := model.Lookup(t.Context(), "MESHSTACK_SKIP_VERSION_CHECK")

	require.NoError(t, err)
	assert.Empty(t, value)
	assert.Empty(t, model.Describe("MESHSTACK_SKIP_VERSION_CHECK"))
}

func Test_MeshStackProviderModel_LookupRejectsAnUnknownValue(t *testing.T) {
	t.Parallel()

	model := MeshStackProviderModel{Endpoint: types.StringUnknown()}

	_, err := model.Lookup(t.Context(), "MESHSTACK_ENDPOINT")

	require.Error(t, err)
}

func Test_MeshStackProvider_SchemaDescribesEveryAttributeAndMarksTheSecretsSensitive(t *testing.T) {
	t.Parallel()

	sensitive := map[string]bool{}
	for key, attribute := range ProviderSchemaForTest(t).Attributes {
		sensitive[key] = attribute.IsSensitive()
		assert.NotEmptyf(t, attribute.GetMarkdownDescription(), "attribute %s has no description", key)
	}

	assert.Equal(t, map[string]bool{
		"endpoint":  false,
		"profile":   false,
		"workspace": false,
		"apikey":    false,
		"apisecret": true,
		"apitoken":  true,
	}, sensitive)
}
