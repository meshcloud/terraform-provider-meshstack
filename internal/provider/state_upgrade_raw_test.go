package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteRawState(t *testing.T) {
	t.Run("moves an attribute and keeps other values byte for byte", func(t *testing.T) {
		upgraded, err := rewriteRawState(
			[]byte(`{"metadata":{"workspace_identifier":"ws","key":"k"},"spec":{"values":["a"]},"quota":12345678901234567890}`),
			moveStateAttribute([]string{"metadata", "workspace_identifier"}, []string{"workspace"}),
			moveStateAttribute([]string{"metadata", "key"}, []string{"key"}),
			moveStateAttribute([]string{"spec", "values"}, []string{"values"}),
			removeStateAttribute("metadata"),
			removeStateAttribute("spec"),
		)
		require.NoError(t, err)
		assert.JSONEq(t, `{"workspace":"ws","key":"k","values":["a"],"quota":12345678901234567890}`, string(upgraded))
	})

	t.Run("builds a project ref from two attributes", func(t *testing.T) {
		upgraded, err := rewriteRawState(
			[]byte(`{"metadata":{"workspace_identifier":"ws","project_identifier":"prj"}}`),
			projectRefFromState([]string{"metadata", "workspace_identifier"}, []string{"metadata", "project_identifier"}, []string{"project_ref"}),
			removeStateAttribute("metadata"),
		)
		require.NoError(t, err)
		assert.JSONEq(t, `{"project_ref":{"kind":"meshProject","owned_by_workspace":"ws","name":"prj"}}`, string(upgraded))
	})

	t.Run("adds a kind to an object and leaves null alone", func(t *testing.T) {
		upgraded, err := rewriteRawState(
			[]byte(`{"target_ref":{"name":"prj","owned_by_workspace":"ws"},"other_ref":null}`),
			addKindToState([]string{"target_ref"}, "meshProject"),
			addKindToState([]string{"other_ref"}, "meshProject"),
			addKindToState([]string{"absent_ref"}, "meshProject"),
		)
		require.NoError(t, err)
		assert.JSONEq(t, `{"target_ref":{"kind":"meshProject","name":"prj","owned_by_workspace":"ws"},"other_ref":null}`, string(upgraded))
	})

	t.Run("leaves a missing attribute alone", func(t *testing.T) {
		upgraded, err := rewriteRawState([]byte(`{"a":1}`), moveStateAttribute([]string{"metadata", "x"}, []string{"y"}))
		require.NoError(t, err)
		assert.JSONEq(t, `{"a":1}`, string(upgraded))
	})
}
