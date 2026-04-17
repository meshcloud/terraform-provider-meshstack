package testconfig

import (
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestConfigImmutability(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  name = "original"
}
`))
	modified := c.WithFirstBlock(Descend("name")(SetString("modified")))

	assert.Contains(t, c.String(), `"original"`)
	assert.Contains(t, modified.String(), `"modified"`)
}

func TestConfigExtractIdentifierResource(t *testing.T) {
	c := newConfig(t, []byte(`resource "meshstack_workspace" "my_ws" {
  name = "test"
}
`))
	var addr Traversal
	c.WithFirstBlock(ExtractAddress(&addr))
	assert.Equal(t, Traversal{"meshstack_workspace", "my_ws"}, addr)
}

func TestConfigExtractIdentifierDataSource(t *testing.T) {
	c := newConfig(t, []byte(`data "meshstack_service_instance" "example" {
  metadata = {
    instance_id = "my-id"
  }
}
`))
	var addr Traversal
	c.WithFirstBlock(ExtractAddress(&addr))
	assert.Equal(t, Traversal{"data.meshstack_service_instance", "example"}, addr)
}

func TestTraverseSetString(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  metadata = {
    name = "old-name"
    tags = "keep-me"
  }
}
`))
	c = c.WithFirstBlock(Descend("metadata", "name")(SetString("new-name")))

	assert.Contains(t, c.String(), `"new-name"`)
	assert.Contains(t, c.String(), `tags = "keep-me"`)
	assert.NotContains(t, c.String(), `"old-name"`)
}

func TestTraverseTreeNesting(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {
    display_name              = "My Project"
    payment_method_identifier = "my-payment"
    tags = {
      "tag-key" = ["some-value"]
    }
  }
}
`))
	c = c.WithFirstBlock(
		Descend("spec")(
			Descend("display_name")(SetString("Updated")),
			Descend("tags", "tag-key")(SetString("new-value")),
		),
	)

	result := c.String()
	assert.Contains(t, result, `"Updated"`)
	assert.Contains(t, result, `"new-value"`)
}

func TestTraverseQuotedKeys(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  inputs = {
    normal = {
      type = "STRING"
    }
    "some-file.yaml" = {
      type  = "FILE"
      value = "old"
    }
  }
}
`))

	c = c.WithFirstBlock(
		Descend("inputs", "some-file.yaml", "value")(SetString("new")),
		Descend("inputs", "normal", "type")(SetString("INTEGER")),
	)

	assert.Contains(t, c.String(), `"new"`)
	assert.Contains(t, c.String(), `"INTEGER"`)
}

func TestTraverseSetRawExpr(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }
}
`))
	c = c.WithFirstBlock(Descend("metadata", "owned_by_workspace")(SetRawExpr("meshstack_workspace.example.metadata.name")))

	assert.Contains(t, c.String(), `meshstack_workspace.example.metadata.name`)
	assert.NotContains(t, c.String(), `"my-workspace"`)
}

func TestTraverseRenameKey(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {
    old_name = "value"
  }
}
`))
	c = c.WithFirstBlock(Descend("spec", "old_name")(RenameKey("new_name")))

	assert.Contains(t, c.String(), `new_name`)
	assert.NotContains(t, c.String(), `old_name`)
}

func TestTraverseSetValue(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {
    draft = true
  }
}
`))
	c = c.WithFirstBlock(Descend("spec", "draft")(SetValue(cty.False)))
	assert.Contains(t, c.String(), `draft = false`)
}

func TestTraverseDeepNesting(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  version_spec = {
    implementation = {
      gitlab_pipeline = {
        token = "old-token"
      }
    }
  }
}
`))
	c = c.WithFirstBlock(Descend("version_spec", "implementation", "gitlab_pipeline", "token")(SetString("new-token")))

	assert.Contains(t, c.String(), `"new-token"`)
	assert.NotContains(t, c.String(), `"old-token"`)
}

func TestTraverseQuotedKeysPreservedAtParentLevel(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  version_spec = {
    inputs = {
      normal = {
        type = "STRING"
      }
      "dashed-key" = {
        type  = "FILE"
        value = "old"
      }
    }
    other = "keep"
  }
}
`))
	c = c.WithFirstBlock(Descend("version_spec", "inputs", "dashed-key", "value")(SetString("new")))

	result := c.String()
	assert.Contains(t, result, `"dashed-key"`)
	assert.Contains(t, result, `"new"`)
	assert.NotContains(t, result, `"old"`)
	assert.Contains(t, result, `other = "keep"`)
}

func TestWithFirstBlockOnJoinedConfig(t *testing.T) {
	primary := newConfig(t, []byte(`resource "meshstack_workspace" "ws" {
  metadata = {
    name = "old-ws"
  }
}
`))
	support := newConfig(t, []byte(`resource "meshstack_tag_definition" "tag" {
  spec = {
    key = "my-tag"
  }
}
`))
	joined := primary.Join(support)
	var addr Traversal
	joined = joined.WithFirstBlock(
		ExtractAddress(&addr),
		Descend("metadata", "name")(SetString("new-ws")),
	)

	assert.Equal(t, Traversal{"meshstack_workspace", "ws"}, addr)
	assert.Contains(t, joined.String(), `"new-ws"`)
	assert.Contains(t, joined.String(), `meshstack_tag_definition`)
}

func TestConfigJoin(t *testing.T) {
	a := newConfig(t, []byte(`resource "a" "x" {}`))
	b := newConfig(t, []byte(`resource "b" "y" {}`))
	joined := a.Join(b)
	assert.Contains(t, joined.String(), `resource "a" "x"`)
	assert.Contains(t, joined.String(), `resource "b" "y"`)
}

// TestWalkAttributesSetBoolTrue is a slightly artificial test: it sets every visited leaf to true,
// which is not meaningful in practice. However it verifies that WalkAttributes correctly descends
// into a nested tree structure and invokes the modifier on every leaf attribute.
func TestWalkAttributesSetBoolTrue(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  enabled = false
  spec = {
    active = false
    nested = {
      deep = "replace-me"
    }
  }
  name = "keep"
}
`))
	c = c.WithFirstBlock(WalkAttributes()(func(t *testing.T, e Expression) {
		t.Helper()
		// only set leaf (non-object) values to true, so nesting is preserved for the walk
		if !strings.Contains(string(e.Get().Bytes()), "{") {
			e.Set(hclwrite.TokensForValue(cty.True))
		}
	}))
	assert.Contains(t, c.String(), `enabled = true`)
	assert.Contains(t, c.String(), `active = true`)
	assert.Contains(t, c.String(), `deep = true`)
	assert.Contains(t, c.String(), `name = true`)
}

func TestWalkAttributesNestedObjects(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {
    name = "original"
    nested = {
      deep = "value"
    }
  }
  top = "flat"
}
`))
	var visited []string
	c.WithFirstBlock(WalkAttributes()(func(_ *testing.T, e Expression) {
		visited = append(visited, string(e.Get().Bytes()))
	}))
	assert.Contains(t, visited, ` "original"`)
	assert.Contains(t, visited, ` "value"`)
	assert.Contains(t, visited, ` "flat"`)
}

func TestWalkAttributesStopsAtLeafValues(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  name   = "leaf"
  count  = 42
  active = true
}
`))
	count := 0
	c.WithFirstBlock(WalkAttributes()(func(_ *testing.T, _ Expression) {
		count++
	}))
	// exactly 3 leaf attributes, no recursion into non-objects
	assert.Equal(t, 3, count)
}

func TestWalkAttributesQuotedKeys(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  inputs = {
    "dashed-key" = {
      value = "inner"
    }
    normal = "flat"
  }
}
`))
	var visited []string
	c.WithFirstBlock(WalkAttributes()(func(_ *testing.T, e Expression) {
		visited = append(visited, string(e.Get().Bytes()))
	}))
	assert.Contains(t, visited, ` "inner"`)
	assert.Contains(t, visited, ` "flat"`)
}

func TestWalkAttributesEmptyObject(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {}
  name = "keep"
}
`))
	count := 0
	c.WithFirstBlock(WalkAttributes()(func(_ *testing.T, _ Expression) {
		count++
	}))
	// spec (the empty object itself) + name = 2 visits; no children inside spec
	assert.Equal(t, 2, count)
}

func TestBlockSetReplacesBody(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  name  = "old"
  count = 1
}
`))
	c = c.WithFirstBlock(SetRawExpr(`"replaced"`))

	assert.Contains(t, c.String(), `"replaced"`)
	assert.NotContains(t, c.String(), `name`)
	assert.NotContains(t, c.String(), `count`)
}

func TestBlockRenameKey(t *testing.T) {
	c := newConfig(t, []byte(`resource "meshstack_workspace" "old_name" {
  metadata = {
    name = "ws"
  }
}
`))
	c = c.WithFirstBlock(func(t *testing.T, e Expression) {
		t.Helper()
		e.RenameKey("new_name")
	})

	assert.Contains(t, c.String(), `resource "meshstack_workspace" "new_name"`)
	assert.NotContains(t, c.String(), `old_name`)
}

func TestSetTopLevelValue(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  name   = "old"
  count  = 1
  active = false
}
`))
	c = c.WithFirstBlock(
		Descend("name")(SetString("new")),
		Descend("count")(SetValue(cty.NumberIntVal(42))),
		Descend("active")(SetValue(cty.True)),
	)

	assert.Contains(t, c.String(), `name   = "new"`)
	assert.Contains(t, c.String(), `count  = 42`)
	assert.Contains(t, c.String(), `active = true`)
}

func TestTraverseUpsertsNewAttribute(t *testing.T) {
	c := newConfig(t, []byte(`resource "test" "ex" {
  spec = {
    name = "test"
  }
}
`))
	c = c.WithFirstBlock(Descend("spec", "new_attr")(SetString("hello")))
	assert.Contains(t, c.String(), `new_attr = "hello"`)
}
