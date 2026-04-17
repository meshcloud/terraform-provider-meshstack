package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Expression is the common interface for traversable HCL nodes (blocks with attributes and nested object constructors).
type Expression interface {
	Get() hclwrite.Tokens
	Set(tokens hclwrite.Tokens)
	RenameKey(newName string)
}

// ExpressionConsumer is a function that consumes an Expression in the context of a test for extraction or modification.
type ExpressionConsumer func(t *testing.T, e Expression)

// SetValue returns an ExpressionConsumer that sets the expression to a cty value.
// See also SetString for the most common usecase.
func SetValue(val cty.Value) ExpressionConsumer {
	return func(t *testing.T, e Expression) {
		t.Helper()
		e.Set(hclwrite.TokensForValue(val))
	}
}

// SetString returns an ExpressionConsumer that sets the expression to a string value.
func SetString(s string) ExpressionConsumer {
	return SetValue(cty.StringVal(s))
}

// SetAddr sets a traversal identifier (typically a resource address) with additional segments appended.
// See OwnedByWorkspace for a very common example usage.
func SetAddr(addr Traversal, moreSegments ...string) ExpressionConsumer {
	return setRawExprString(append(addr, moreSegments...).String())
}

// SetRawExpr returns an ExpressionConsumer that sets the expression to a raw HCL expression string.
// Important: Use this modifier as a last resort. Always prefer other modifiers such as SetAddr or SetValue.
// The first format should use Go raw `...` expressions to avoid quoting of literal expressions inside if necessary.
func SetRawExpr(format string, args ...any) ExpressionConsumer {
	return setRawExprString(fmt.Sprintf(format, args...))
}

// RenameKey returns an ExpressionConsumer that renames the traversed attribute or block label.
func RenameKey(newName string) ExpressionConsumer {
	return func(t *testing.T, e Expression) {
		t.Helper()
		e.RenameKey(newName)
	}
}

// ExtractAddress returns an ExpressionConsumer that extracts the Terraform resource address
// from a resource or data block into the target Traversal.
func ExtractAddress(target *Traversal) ExpressionConsumer {
	return func(t *testing.T, e Expression) {
		t.Helper()
		b, ok := e.(Block)
		require.True(t, ok, "ExtractIdentifier requires a Block expression, got %T — call it directly on WithFirstBlock, not inside Descend", e)
		labels := b.internal.Labels()
		switch b.internal.Type() {
		case "resource":
			*target = Traversal{labels[0], labels[1]}
		case "data":
			*target = Traversal{"data." + labels[0], labels[1]}
		default:
			require.Fail(t, fmt.Sprintf("ExtractIdentifier only works on resource or data blocks, got block type %q", b.internal.Type()))
		}
	}
}

// OwnedByWorkspace returns an ExpressionConsumer that sets metadata.owned_by_workspace
// to a reference to the given workspace's metadata.name.
func OwnedByWorkspace(workspaceAddr Traversal) ExpressionConsumer {
	return Descend("metadata", "owned_by_workspace")(SetAddr(workspaceAddr, "metadata", "name"))
}

func setRawExprString(expr string) ExpressionConsumer {
	return func(t *testing.T, e Expression) {
		t.Helper()
		src := fmt.Sprintf("_x = %s\n", expr)
		file, diags := hclwrite.ParseConfig([]byte(src), "", hcl.Pos{Line: 1, Column: 1})
		require.Falsef(t, diags.HasErrors(), "SetRawExpr: failed to parse expression %q as HCL — is it valid HCL syntax? Error: %s", expr, diags.Error())
		//goland:noinspection GoDfaErrorMayBeNotNil
		e.Set(file.Body().GetAttribute("_x").Expr().BuildTokens(nil))
	}
}

type attributeExpression struct {
	t      *testing.T
	Name   string
	Parent parent
}

// parent is implemented by *hclwrite.Body and fakeBlock.
type parent interface {
	Attributes() map[string]*hclwrite.Attribute
	SetAttributeRaw(name string, tokens hclwrite.Tokens) *hclwrite.Attribute
	RenameAttribute(fromName, toName string) bool
}

func (a attributeExpression) Get() hclwrite.Tokens {
	attributes := a.Parent.Attributes()
	if attribute, found := attributes[a.Name]; found {
		return attribute.Expr().BuildTokens(nil)
	}
	// If attribute doesn't exist, return empty tokens (upsert semantics — Set will create it)
	return hclwrite.Tokens{}
}

func (a attributeExpression) Set(tokens hclwrite.Tokens) {
	a.Parent.SetAttributeRaw(a.Name, tokens)
}

func (a attributeExpression) RenameKey(newName string) {
	a.Parent.RenameAttribute(a.Name, newName)
}

func (a attributeExpression) traverse(key string) Expression {
	return attributeExpression{a.t, key, fakeBlock{a.t, a}}
}

// attributes returns the child attributes if this expression holds an object value ({...}),
// or nil if it is not an object.
func (a attributeExpression) attributes() map[string]*hclwrite.Attribute {
	return fakeBlock{a.t, a}.Attributes()
}
