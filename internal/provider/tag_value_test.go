package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithTagValue(t *testing.T) {
	t.Run("appends a missing value and leaves the input alone", func(t *testing.T) {
		tags := map[string][]string{"teams": {"a"}, "other": {"x"}}
		updated, changed := withTagValue(tags, "teams", "b")
		assert.True(t, changed)
		assert.Equal(t, map[string][]string{"teams": {"a", "b"}, "other": {"x"}}, updated)
		assert.Equal(t, map[string][]string{"teams": {"a"}, "other": {"x"}}, tags)
	})

	t.Run("adds the key when the tag is absent", func(t *testing.T) {
		updated, changed := withTagValue(nil, "teams", "a")
		assert.True(t, changed)
		assert.Equal(t, map[string][]string{"teams": {"a"}}, updated)
	})

	t.Run("reports no change for a present value", func(t *testing.T) {
		_, changed := withTagValue(map[string][]string{"teams": {"a", "b"}}, "teams", "b")
		assert.False(t, changed)
	})
}

func TestWithoutTagValue(t *testing.T) {
	t.Run("removes only the value and keeps the order", func(t *testing.T) {
		tags := map[string][]string{"teams": {"a", "b", "c"}}
		updated, changed := withoutTagValue(tags, "teams", "b")
		assert.True(t, changed)
		assert.Equal(t, map[string][]string{"teams": {"a", "c"}}, updated)
		assert.Equal(t, map[string][]string{"teams": {"a", "b", "c"}}, tags)
	})

	t.Run("drops the key with its last value", func(t *testing.T) {
		updated, changed := withoutTagValue(map[string][]string{"teams": {"a"}, "other": {"x"}}, "teams", "a")
		assert.True(t, changed)
		assert.Equal(t, map[string][]string{"other": {"x"}}, updated)
	})

	t.Run("reports no change for a missing value", func(t *testing.T) {
		_, changed := withoutTagValue(map[string][]string{"teams": {"a"}}, "teams", "b")
		assert.False(t, changed)
	})
}

func TestParseTagValueImportID(t *testing.T) {
	tests := []struct {
		id          string
		identifiers int
		wantIds     []string
		wantKey     string
		wantValue   string
		wantOk      bool
	}{
		{"ws.teams=a", 1, []string{"ws"}, "teams", "a", true},
		{"ws.prj.teams=a", 2, []string{"ws", "prj"}, "teams", "a", true},
		{"ws.my.key=a=b.c", 1, []string{"ws"}, "my.key", "a=b.c", true},
		{"ws.teams", 1, nil, "", "", false},
		{"ws.teams=", 1, nil, "", "", false},
		{"ws.=a", 1, nil, "", "", false},
		{".teams=a", 1, nil, "", "", false},
		{"ws.teams=a", 2, nil, "", "", false},
		{"ws..teams=a", 2, nil, "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			ids, key, value, ok := parseTagValueImportID(tt.id, tt.identifiers)
			assert.Equal(t, tt.wantOk, ok)
			assert.Equal(t, tt.wantIds, ids)
			assert.Equal(t, tt.wantKey, key)
			assert.Equal(t, tt.wantValue, value)
		})
	}
}
