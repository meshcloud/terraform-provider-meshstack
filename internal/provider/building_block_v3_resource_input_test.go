package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseStringInputToClientValue(t *testing.T) {
	t.Run("keeps plain string value", func(t *testing.T) {
		result := parseStringInputToClientValue("my-secret")

		require.Nil(t, result.Sensitive)
		require.Equal(t, "my-secret", result.Value)
	})

	t.Run("decodes json number", func(t *testing.T) {
		result := parseStringInputToClientValue(`16`)

		require.Nil(t, result.Sensitive)
		typed, ok := result.Value.(float64)
		require.True(t, ok)
		require.InDelta(t, 16.0, typed, 0.0)
	})

	t.Run("keeps object-like values as non-sensitive value", func(t *testing.T) {
		result := parseStringInputToClientValue(`{"plaintext":"my-secret"}`)

		require.Nil(t, result.Sensitive)
		typed, ok := result.Value.(map[string]any)
		require.True(t, ok)
		require.Equal(t, "my-secret", typed["plaintext"])
	})
}
