package testconfig

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zclconf/go-cty/cty"
)

func TestConfigImmutability(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  name = "original"
}
`))
	modified := c.WithFirstBlock(t, Traverse(t, "name")(SetString("modified")))

	assert.Contains(t, c.String(), `"original"`)
	assert.Contains(t, modified.String(), `"modified"`)
}

func TestConfigExtractIdentifierResource(t *testing.T) {
	c := NewConfig(t, []byte(`resource "meshstack_workspace" "my_ws" {
  name = "test"
}
`))
	var addr Traversal
	c.WithFirstBlock(t, ExtractIdentifier(&addr))
	assert.Equal(t, Traversal{"meshstack_workspace", "my_ws"}, addr)
}

func TestConfigExtractIdentifierDataSource(t *testing.T) {
	c := NewConfig(t, []byte(`data "meshstack_service_instance" "example" {
  metadata = {
    instance_id = "my-id"
  }
}
`))
	var addr Traversal
	c.WithFirstBlock(t, ExtractIdentifier(&addr))
	assert.Equal(t, Traversal{"data.meshstack_service_instance", "example"}, addr)
}

func TestTraverseSetString(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  metadata = {
    name = "old-name"
    tags = "keep-me"
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "metadata", "name")(SetString("new-name")))

	assert.Contains(t, c.String(), `"new-name"`)
	assert.Contains(t, c.String(), `tags = "keep-me"`)
	assert.NotContains(t, c.String(), `"old-name"`)
}

func TestTraverseTreeNesting(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    display_name              = "My Project"
    payment_method_identifier = "my-payment"
    tags = {
      "tag-key" = ["some-value"]
    }
  }
}
`))
	c = c.WithFirstBlock(t,
		Traverse(t, "spec")(
			Traverse(t, "display_name")(SetString("Updated")),
			Traverse(t, "payment_method_identifier")(RemoveKey()),
			Traverse(t, "tags", "tag-key")(SetString("new-value")),
		),
	)

	result := c.String()
	assert.Contains(t, result, `"Updated"`)
	assert.NotContains(t, result, `payment_method_identifier`)
	assert.Contains(t, result, `"new-value"`)
}

func TestTraverseQuotedKeys(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
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

	c = c.WithFirstBlock(t,
		Traverse(t, "inputs", "some-file.yaml", "value")(SetString("new")),
		Traverse(t, "inputs", "normal", "type")(SetString("INTEGER")),
	)

	assert.Contains(t, c.String(), `"new"`)
	assert.Contains(t, c.String(), `"INTEGER"`)
}

func TestTraverseSetRawExpr(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  metadata = {
    owned_by_workspace = "my-workspace"
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "metadata", "owned_by_workspace")(SetRawExpr("meshstack_workspace.example.metadata.name")))

	assert.Contains(t, c.String(), `meshstack_workspace.example.metadata.name`)
	assert.NotContains(t, c.String(), `"my-workspace"`)
}

func TestTraverseRenameKey(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    old_name = "value"
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "spec", "old_name")(RenameKey("new_name")))

	assert.Contains(t, c.String(), `new_name`)
	assert.NotContains(t, c.String(), `old_name`)
}

func TestTraverseRemoveKey(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    display_name              = "My Project"
    payment_method_identifier = "my-payment"
    tags = {
      "tag-key" = ["some-value"]
    }
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "spec", "payment_method_identifier")(RemoveKey()))

	assert.Contains(t, c.String(), `display_name`)
	assert.NotContains(t, c.String(), `payment_method_identifier`)
	assert.Contains(t, c.String(), `"tag-key"`)
}

func TestTraverseSetCty(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    draft = true
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "spec", "draft")(SetCty(cty.False)))
	assert.Contains(t, c.String(), `draft = false`)
}

func TestTraverseDeepNesting(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  version_spec = {
    implementation = {
      gitlab_pipeline = {
        token = "old-token"
      }
    }
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "version_spec", "implementation", "gitlab_pipeline", "token")(SetString("new-token")))

	assert.Contains(t, c.String(), `"new-token"`)
	assert.NotContains(t, c.String(), `"old-token"`)
}

func TestTraversePreservesUnchangedAttrs(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    display_name              = "My Project"
    payment_method_identifier = "my-payment"
    some_other                = "value"
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "spec", "payment_method_identifier")(RemoveKey()))

	lines := strings.Split(c.String(), "\n")
	var specLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "display_name") || strings.HasPrefix(trimmed, "some_other") {
			specLines = append(specLines, trimmed)
		}
	}
	assert.Len(t, specLines, 2)
}

func TestTraverseQuotedKeysPreservedAtParentLevel(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
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
	c = c.WithFirstBlock(t, Traverse(t, "version_spec", "inputs", "dashed-key", "value")(SetString("new")))

	result := c.String()
	assert.Contains(t, result, `"dashed-key"`)
	assert.Contains(t, result, `"new"`)
	assert.NotContains(t, result, `"old"`)
	assert.Contains(t, result, `other = "keep"`)
}

func TestWithFirstBlockOnJoinedConfig(t *testing.T) {
	primary := NewConfig(t, []byte(`resource "meshstack_workspace" "ws" {
  metadata = {
    name = "old-ws"
  }
}
`))
	support := NewConfig(t, []byte(`resource "meshstack_tag_definition" "tag" {
  spec = {
    key = "my-tag"
  }
}
`))
	joined := primary.Join(support)
	var addr Traversal
	joined = joined.WithFirstBlock(t,
		ExtractIdentifier(&addr),
		Traverse(t, "metadata", "name")(SetString("new-ws")),
	)

	assert.Equal(t, Traversal{"meshstack_workspace", "ws"}, addr)
	assert.Contains(t, joined.String(), `"new-ws"`)
	assert.Contains(t, joined.String(), `meshstack_tag_definition`)
}

func TestConfigJoin(t *testing.T) {
	a := NewConfig(t, []byte(`resource "a" "x" {}`))
	b := NewConfig(t, []byte(`resource "b" "y" {}`))
	joined := a.Join(b)
	assert.Contains(t, joined.String(), `resource "a" "x"`)
	assert.Contains(t, joined.String(), `resource "b" "y"`)
}

func TestTraverseAttributesSetBoolTrueIfFalse(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  enabled = false
  active  = false
  name    = "keep"
}
`))
	c = c.WithFirstBlock(t, TraverseAttributes(t)(SetBoolTrueIfFalse()))
	assert.Contains(t, c.String(), `enabled = true`)
	assert.Contains(t, c.String(), `active  = true`)
	assert.Contains(t, c.String(), `name    = "keep"`)
}

func TestTraverseUpsertsNewAttribute(t *testing.T) {
	c := NewConfig(t, []byte(`resource "test" "ex" {
  spec = {
    name = "test"
  }
}
`))
	c = c.WithFirstBlock(t, Traverse(t, "spec", "new_attr")(SetString("hello")))
	assert.Contains(t, c.String(), `new_attr = "hello"`)
}
