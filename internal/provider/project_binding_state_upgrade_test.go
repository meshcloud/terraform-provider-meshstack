package provider

import (
	_ "embed"
	"testing"

	"github.com/meshcloud/meshstack-cli/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// As provider v0.28.2 wrote it.
//
//go:embed testdata/project_binding/state-v0.json
var projectBindingStateV0 string

func TestProjectBindingUpgradeState(t *testing.T) {
	t.Run("v0_to_v1", func(t *testing.T) {
		binding, ok := NewProjectUserBindingResource().(*bindingResource[client.MeshProjectUserBinding])
		require.True(t, ok)
		upgraded, diags := UpgradeResourceState(t, binding, 0, projectBindingStateV0)
		require.False(t, diags.HasError(), "upgrade produced errors: %s", diags)

		var targetRef projectRef
		upgraded.Attribute("target_ref", &targetRef)
		assert.Equal(t, newProjectRef("my-workspace", "my-project"), targetRef)

		var name, expiryDate string
		upgraded.Attribute("metadata.name", &name)
		upgraded.Attribute("expiry_date", &expiryDate)
		assert.Equal(t, "example-project-user-binding", name)
		assert.Equal(t, "2099-12-31", expiryDate)
	})
}
