package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnstoredTags(t *testing.T) {
	t.Run("accepts stored tags and ignores keys it did not write", func(t *testing.T) {
		written := map[string][]string{"env": {"prod"}, "empty": {}}
		stored := map[string][]string{"env": {"prod"}, "restricted": {"default"}}
		assert.Empty(t, unstoredTags(written, stored))
	})

	t.Run("reports a restricted tag meshStack kept", func(t *testing.T) {
		written := map[string][]string{"restricted": {"new"}, "env": {"prod"}}
		stored := map[string][]string{"restricted": {"old"}, "env": {"prod"}}
		assert.Equal(t, []string{"restricted"}, unstoredTags(written, stored))
	})

	t.Run("reports missing keys in sorted order", func(t *testing.T) {
		assert.Equal(t, []string{"a", "b"}, unstoredTags(map[string][]string{"b": {"x"}, "a": {"y"}}, nil))
	})
}
