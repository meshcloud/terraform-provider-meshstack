package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testStringer struct{}

func (t testStringer) String() string {
	return "test"
}

func Test_convertArgsForLogging(t *testing.T) {
	tests := []struct {
		name string
		args []any
		want map[string]any
	}{
		{"empty", []any{}, map[string]any{}},
		{"one pair", []any{"k", "v"}, map[string]any{"k": "v"}},
		{"two pairs", []any{"k1", "v1", "k2", "v2"}, map[string]any{"k1": "v1", "k2": "v2"}},
		{"only one", []any{"k1"}, map[string]any{"k1": "<missing value>"}},
		{"duplicate keys", []any{"k1", "v1", "k1", "v2", "k1", "v3"}, map[string]any{"k1 <duplicate=0>": "v1", "k1 <duplicate=1>": "v2", "k1 <duplicate=2>": "v3"}},
		{"odd args", []any{"k1", true, "k2"}, map[string]any{"k1": true, "k2": "<missing value>"}},
		{"non-string key", []any{"k1", true, false, "v2"}, map[string]any{"k1": true, "'false'(bool) <non-string key at i=2>": "v2"}},
		{"stringer key", []any{testStringer{}, "v1", "k2", "v2"}, map[string]any{"test": "v1", "k2": "v2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult := convertArgsForLogging(tt.args)
			assert.Equal(t, tt.want, gotResult)
		})
	}
}
