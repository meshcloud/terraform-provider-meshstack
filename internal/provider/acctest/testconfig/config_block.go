package testconfig

import (
	"testing"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
)

// Block wraps a *hclwrite.Block for use as an Expression.
type Block struct {
	t        *testing.T
	internal *hclwrite.Block
}

func (b Block) traverse(key string) Expression {
	return attributeExpression{b.t, key, b.internal.Body()}
}

func (b Block) attributes() map[string]*hclwrite.Attribute {
	return b.internal.Body().Attributes()
}

func (b Block) Get() hclwrite.Tokens {
	return b.internal.Body().BuildTokens(nil)
}

func (b Block) Set(tokens hclwrite.Tokens) {
	body := b.internal.Body()
	body.Clear()
	body.AppendUnstructuredTokens(tokens)
}

// RenameKey renames the last label of the block (e.g. the resource name in a resource block).
func (b Block) RenameKey(newName string) {
	labels := b.internal.Labels()
	require.NotEmpty(b.t, labels, "RenameKey: block %q has no labels to rename", b.internal.Type())
	labels[len(labels)-1] = newName
	b.internal.SetLabels(labels)
}
