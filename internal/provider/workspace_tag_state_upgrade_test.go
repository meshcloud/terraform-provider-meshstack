package provider

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
)

// As provider v0.24.3 up to v0.28.2 wrote it.
var (
	//go:embed testdata/workspace_tag/state-v0.json
	workspaceTagStateV0 string
	//go:embed testdata/workspace_tags/state-v0.json
	workspaceTagsStateV0 string
)

func TestWorkspaceTagUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		upgraded := UpgradeRawResourceState(t, NewWorkspaceTagResource(), 0, workspaceTagStateV0)
		assert.JSONEq(t, `{"workspace":"my-workspace","key":"cost-center","values":["12345","67890"]}`, upgraded)
	})
}

func TestWorkspaceTagsUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		upgraded := UpgradeRawResourceState(t, NewWorkspaceTagsResource(), 0, workspaceTagsStateV0)
		assert.JSONEq(t, `{"workspace":"my-workspace","tags":{"cost-center":["12345"],"empty":[]}}`, upgraded)
	})
}
