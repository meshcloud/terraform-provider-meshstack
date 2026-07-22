package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReconcileTags pins the reconciliation the tags fix relies on: the meshObject API returns a
// superset of the caller's tags (an empty-list entry for every schema property plus injected
// restricted-tag defaults), and only the keys already tracked in state must survive.
func TestReconcileTags(t *testing.T) {
	tests := map[string]struct {
		tracked map[string][]string
		apiTags map[string][]string
		want    map[string][]string
	}{
		"drops undeclared superset and injected restricted default": {
			tracked: map[string][]string{"env": {"prod"}},
			apiTags: map[string][]string{
				"env":            {"prod"},
				"cost-center":    {},                   // undeclared property, empty-list superset entry
				"restricted-tag": {"injected-default"}, // server-injected restricted default
			},
			want: map[string][]string{"env": {"prod"}},
		},
		"keeps the API value for a tracked key": {
			tracked: map[string][]string{"env": {"stale"}},
			apiTags: map[string][]string{"env": {"prod"}},
			want:    map[string][]string{"env": {"prod"}},
		},
		"drops a tracked key the API no longer returns": {
			tracked: map[string][]string{"env": {"prod"}, "gone": {"x"}},
			apiTags: map[string][]string{"env": {"prod"}},
			want:    map[string][]string{"env": {"prod"}},
		},
		"nothing tracked yields empty result": {
			// "only restricted": the caller declares no tags, the backend injects a restricted default.
			tracked: map[string][]string{},
			apiTags: map[string][]string{"restricted-tag": {"injected-default"}},
			want:    map[string][]string{},
		},
		"keeps several tracked keys at once": {
			tracked: map[string][]string{"env": {"prod"}, "team": {"platform"}, "cost-center": {"cc-1"}},
			apiTags: map[string][]string{
				"env":            {"prod"},
				"team":           {"platform"},
				"cost-center":    {"cc-1"},
				"unset-property": {},                   // undeclared property, empty-list superset entry
				"restricted-tag": {"injected-default"}, // injected restricted default
			},
			want: map[string][]string{"env": {"prod"}, "team": {"platform"}, "cost-center": {"cc-1"}},
		},
		"keeps a declared restricted tag but drops an injected undeclared one": {
			// A restricted tag the caller is permitted to set and declares is tracked, so it survives;
			// a different restricted tag the backend injects without the caller declaring it is dropped.
			tracked: map[string][]string{"env": {"prod"}, "declared-restricted": {"set-by-caller"}},
			apiTags: map[string][]string{
				"env":                 {"prod"},
				"declared-restricted": {"set-by-caller"},
				"injected-restricted": {"injected-default"},
			},
			want: map[string][]string{"env": {"prod"}, "declared-restricted": {"set-by-caller"}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, reconcileTags(tc.tracked, tc.apiTags))
		})
	}
}
