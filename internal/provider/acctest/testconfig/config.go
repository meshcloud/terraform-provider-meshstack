// Package testconfig provides a fluent API for building and modifying HCL configurations
// in Terraform provider acceptance tests.
package testconfig

import (
	"bytes"
	"fmt"
	"sort"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"
	"github.com/zclconf/go-cty/cty"
)

// Config wraps a parsed *hclwrite.File. All Config methods return a new Config — the receiver is never mutated.
// Call .String() only at the test step boundary (i.e. when assigning to resource.TestStep.Config).
type Config struct {
	internal *hclwrite.File
}

// NewConfig parses HCL bytes into a Config. Fails the test if parsing fails.
func NewConfig(t *testing.T, src []byte) Config {
	t.Helper()
	file, diags := hclwrite.ParseConfig(src, "", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "failed to parse HCL config (%d bytes): %s", len(src), diags.Error())
	require.NotNil(t, file, "parsed HCL file is nil — check the source bytes")
	return Config{file}
}

func clone(c Config) Config {
	file, diags := hclwrite.ParseConfig(c.internal.Bytes(), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil {
		panic(fmt.Sprintf("internal clone failed (this is a bug): %s", diags.Error()))
	}
	return Config{file}
}

// String renders the config to HCL text.
func (c Config) String() string {
	return string(c.internal.Bytes())
}

// Join combines multiple configs by appending all blocks from others into a new Config.
// The receiver is not modified.
func (c Config) Join(others ...Config) Config {
	result := clone(c)
	for _, other := range others {
		cp := clone(other)
		for _, block := range cp.internal.Body().Blocks() {
			cp.internal.Body().RemoveBlock(block)
			result.internal.Body().AppendNewline()
			result.internal.Body().AppendBlock(block)
		}
	}
	return result
}

// WithFirstBlock applies modifiers to the first block of a cloned Config and returns the new Config.
// The receiver is not modified.
func (c Config) WithFirstBlock(t *testing.T, modifiers ...ExpressionModifier) Config {
	t.Helper()
	result := clone(c)
	blocks := result.internal.Body().Blocks()
	require.NotEmpty(t, blocks, "WithFirstBlock: config has no blocks — did you load the right .tf file?")
	for _, modifier := range modifiers {
		modifier(t, Block{blocks[0]})
	}
	return result
}

// Block wraps a *hclwrite.Block for use as an Expression.
type Block struct {
	internal *hclwrite.Block
}

func (b Block) traverse(key string) Expression {
	return attributeExpression{key, b.internal.Body()}
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
func (b Block) RenameKey(t *testing.T, newName string) {
	t.Helper()
	labels := b.internal.Labels()
	require.NotEmpty(t, labels, "RenameKey: block %q has no labels to rename", b.internal.Type())
	labels[len(labels)-1] = newName
	b.internal.SetLabels(labels)
}

// Expression is the common interface for traversable HCL nodes (blocks and attributes).
type Expression interface {
	Get() hclwrite.Tokens
	Set(tokens hclwrite.Tokens)
	RenameKey(t *testing.T, newName string)
}

// ExpressionModifier is a function that modifies an Expression in the context of a test.
type ExpressionModifier func(t *testing.T, e Expression)

// SetCty returns an ExpressionModifier that sets the expression to a cty value.
func SetCty(val cty.Value) ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		e.Set(hclwrite.TokensForValue(val))
	}
}

// SetString returns an ExpressionModifier that sets the expression to a string value.
func SetString(s string) ExpressionModifier {
	return SetCty(cty.StringVal(s))
}

// SetRawExpr returns an ExpressionModifier that sets the expression to a raw HCL expression string.
func SetRawExpr(expr string) ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		e.Set(parseExprTokens(t, expr))
	}
}

// RenameKey returns an ExpressionModifier that renames the traversed attribute or block label.
func RenameKey(newName string) ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		e.RenameKey(t, newName)
	}
}

// RemoveKey returns an ExpressionModifier that removes the traversed attribute from its parent.
func RemoveKey() ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		a, ok := e.(attributeExpression)
		require.True(t, ok, "RemoveKey requires an attributeExpression, got %T — did you Traverse to a leaf attribute?", e)
		a.Parent.RemoveAttribute(a.Name)
	}
}

// ExtractIdentifier returns an ExpressionModifier that extracts the Terraform resource address
// from a resource or data block into the target Traversal.
func ExtractIdentifier(target *Traversal) ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		b, ok := e.(Block)
		require.True(t, ok, "ExtractIdentifier requires a Block expression, got %T — call it directly on WithFirstBlock, not inside Traverse", e)
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

// Traverse returns a modifier-builder that traverses into the given attribute path steps
// and applies the provided modifiers to the resolved expression.
// t is used for error reporting during traversal.
func Traverse(t *testing.T, steps ...string) func(modifiers ...ExpressionModifier) ExpressionModifier {
	t.Helper()
	return func(modifiers ...ExpressionModifier) ExpressionModifier {
		return func(_ *testing.T, root Expression) {
			traversed := root
			for idx, step := range steps {
				traversable, ok := traversed.(stringTraversable)
				require.True(t, ok,
					"Traverse: expected string-traversable expression at step %q (index %d), got %T — cannot traverse into this node type",
					step, idx, traversed,
				)
				traversed = traversable.traverse(step)
			}
			for _, modifier := range modifiers {
				modifier(t, traversed)
			}
		}
	}
}

// TraverseAttributes returns a modifier-builder that visits every attribute reachable from root,
// including nested object attributes, and applies the provided modifiers to each attribute expression.
func TraverseAttributes(t *testing.T) func(modifiers ...ExpressionModifier) ExpressionModifier {
	t.Helper()
	return func(modifiers ...ExpressionModifier) ExpressionModifier {
		return func(_ *testing.T, root Expression) {
			var visit func(Expression)
			visit = func(expr Expression) {
				names := attributeNames(expr)
				if len(names) == 0 {
					return
				}
				traversable, ok := expr.(stringTraversable)
				if !ok {
					return
				}
				sort.Strings(names)
				for _, name := range names {
					child := traversable.traverse(name)
					for _, modifier := range modifiers {
						modifier(t, child)
					}
					visit(child)
				}
			}
			visit(root)
		}
	}
}

// SetBoolTrueIfFalse changes an attribute expression from `false` to `true` and leaves all other values unchanged.
func SetBoolTrueIfFalse() ExpressionModifier {
	return func(t *testing.T, e Expression) {
		t.Helper()
		if bytes.Equal(bytes.TrimSpace(e.Get().Bytes()), []byte("false")) {
			e.Set(hclwrite.TokensForValue(cty.BoolVal(true)))
		}
	}
}

func attributeNames(expr Expression) []string {
	switch typed := expr.(type) {
	case Block:
		attrs := typed.internal.Body().Attributes()
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		return keys
	case attributeExpression:
		tokens := typed.Get()
		if len(tokens) < 2 || tokens[0].Type != hclsyntax.TokenOBrace || tokens[len(tokens)-1].Type != hclsyntax.TokenCBrace {
			return nil
		}
		attrs := fakeBlock{typed}.Attributes()
		keys := make([]string, 0, len(attrs))
		for k := range attrs {
			keys = append(keys, k)
		}
		return keys
	default:
		return nil
	}
}

// OwnedByWorkspace returns an ExpressionModifier that sets metadata.owned_by_workspace
// to a reference to the given workspace's metadata.name.
func OwnedByWorkspace(t *testing.T, workspaceAddr Traversal) ExpressionModifier {
	t.Helper()
	return Traverse(t, "metadata", "owned_by_workspace")(SetRawExpr(workspaceAddr.Join("metadata", "name").String()))
}

type stringTraversable interface {
	traverse(key string) Expression
}

type attributeExpression struct {
	Name   string
	Parent parent
}

type parent interface {
	Attributes() map[string]*hclwrite.Attribute
	SetAttributeRaw(name string, tokens hclwrite.Tokens) *hclwrite.Attribute
	RemoveAttribute(name string) *hclwrite.Attribute
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

func (a attributeExpression) RenameKey(t *testing.T, newName string) {
	t.Helper()
	a.Parent.RenameAttribute(a.Name, newName)
}

func (a attributeExpression) traverse(key string) Expression {
	return attributeExpression{key, fakeBlock{a}}
}

type fakeBlock struct {
	internal attributeExpression
}

// quotedNameMap holds bidirectional mapping between original quoted attribute names and sanitized identifiers.
type quotedNameMap struct {
	toSanitized map[string]string
	toOriginal  map[string]string
}

// unquoteAttributeNames replaces quoted attribute name tokens (OQuote+QuotedLit+CQuote before Equal)
// with a single TokenIdent so that hclwrite.ParseConfig accepts them.
// Only replaces at brace depth 0 to avoid mangling keys inside nested object expressions.
func unquoteAttributeNames(tokens hclwrite.Tokens) (hclwrite.Tokens, quotedNameMap) {
	m := quotedNameMap{toSanitized: make(map[string]string), toOriginal: make(map[string]string)}
	var result hclwrite.Tokens
	counter := 0
	depth := 0
	for i := 0; i < len(tokens); i++ {
		switch tokens[i].Type {
		case hclsyntax.TokenOBrace:
			depth++
		case hclsyntax.TokenCBrace:
			depth--
		}
		if depth == 0 && i+3 < len(tokens) &&
			tokens[i].Type == hclsyntax.TokenOQuote &&
			tokens[i+1].Type == hclsyntax.TokenQuotedLit &&
			tokens[i+2].Type == hclsyntax.TokenCQuote &&
			tokens[i+3].Type == hclsyntax.TokenEqual {
			original := string(tokens[i+1].Bytes)
			sanitized, exists := m.toSanitized[original]
			if !exists {
				sanitized = fmt.Sprintf("__q%d__", counter)
				counter++
				m.toSanitized[original] = sanitized
				m.toOriginal[sanitized] = original
			}
			result = append(result, &hclwrite.Token{
				Type:         hclsyntax.TokenIdent,
				Bytes:        []byte(sanitized),
				SpacesBefore: tokens[i].SpacesBefore,
			})
			i += 2 // skip QuotedLit and CQuote
		} else {
			result = append(result, tokens[i])
		}
	}
	return result, m
}

// requoteAttributeNames restores sanitized TokenIdent tokens back to quoted attribute name tokens.
func requoteAttributeNames(tokens hclwrite.Tokens, m quotedNameMap) hclwrite.Tokens {
	var result hclwrite.Tokens
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenIdent {
			if original, ok := m.toOriginal[string(tokens[i].Bytes)]; ok && i+1 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenEqual {
				result = append(result,
					&hclwrite.Token{Type: hclsyntax.TokenOQuote, Bytes: []byte{'"'}, SpacesBefore: tokens[i].SpacesBefore},
					&hclwrite.Token{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(original)},
					&hclwrite.Token{Type: hclsyntax.TokenCQuote, Bytes: []byte{'"'}},
				)
				continue
			}
		}
		result = append(result, tokens[i])
	}
	return result
}

func (f fakeBlock) Attributes() (result map[string]*hclwrite.Attribute) {
	// withBody requires t; use a no-op panic fallback since Attributes is called during traversal
	// where t is available from the surrounding ExpressionModifier.
	// In practice this is always called from within a modifier that has t.
	f.withBodyPanic(func(body *hclwrite.Body, names quotedNameMap) bool {
		sanitized := body.Attributes()
		result = make(map[string]*hclwrite.Attribute, len(sanitized))
		for k, v := range sanitized {
			if original, ok := names.toOriginal[k]; ok {
				result[original] = v
			} else {
				result[k] = v
			}
		}
		return false
	})
	return
}

func (f fakeBlock) SetAttributeRaw(name string, tokens hclwrite.Tokens) (result *hclwrite.Attribute) {
	f.withBodyPanic(func(body *hclwrite.Body, names quotedNameMap) bool {
		actual := name
		if s, ok := names.toSanitized[name]; ok {
			actual = s
		}
		result = body.SetAttributeRaw(actual, tokens)
		return true
	})
	return
}

func (f fakeBlock) RemoveAttribute(name string) (result *hclwrite.Attribute) {
	f.withBodyPanic(func(body *hclwrite.Body, names quotedNameMap) bool {
		actual := name
		if s, ok := names.toSanitized[name]; ok {
			actual = s
		}
		result = body.RemoveAttribute(actual)
		return result != nil
	})
	return
}

func (f fakeBlock) RenameAttribute(fromName, toName string) (result bool) {
	f.withBodyPanic(func(body *hclwrite.Body, names quotedNameMap) bool {
		actualFrom, actualTo := fromName, toName
		if s, ok := names.toSanitized[fromName]; ok {
			actualFrom = s
		}
		if s, ok := names.toSanitized[toName]; ok {
			actualTo = s
		}
		result = body.RenameAttribute(actualFrom, actualTo)
		return true
	})
	return
}

// withBodyPanic is used by the parent interface methods (Attributes, SetAttributeRaw, etc.)
// which cannot accept t because the parent interface doesn't carry t.
// Token structure errors here indicate a programming bug, not user input errors, so panic is appropriate.
func (f fakeBlock) withBodyPanic(modifier func(body *hclwrite.Body, names quotedNameMap) (modified bool)) {
	tokens := f.internal.Get()
	if len(tokens) < 2 {
		panic(fmt.Sprintf("fakeBlock internal error: expected at least 2 tokens, got %d — attribute is not an object", len(tokens)))
	}
	if tokens[0].Type != hclsyntax.TokenOBrace {
		panic(fmt.Sprintf("fakeBlock internal error: expected TokenOBrace, got %s — attribute is not an object", tokens[0].Type))
	}
	if tokens[len(tokens)-1].Type != hclsyntax.TokenCBrace {
		panic(fmt.Sprintf("fakeBlock internal error: expected TokenCBrace, got %s — attribute is not an object", tokens[len(tokens)-1].Type))
	}
	firstToken := tokens[0]
	lastToken := tokens[len(tokens)-1]
	strippedTokens := tokens[1 : len(tokens)-1]
	unquotedTokens, names := unquoteAttributeNames(strippedTokens)
	file, diags := hclwrite.ParseConfig(unquotedTokens.Bytes(), "", hcl.Pos{})
	if diags.HasErrors() || file == nil {
		panic(fmt.Sprintf("fakeBlock internal error: failed to parse inner tokens: %s", diags.Error()))
	}
	strippedBody := file.Body()
	if modifier(strippedBody, names) {
		restoredTokens := strippedBody.BuildTokens(hclwrite.Tokens{firstToken})
		restoredTokens = requoteAttributeNames(restoredTokens, names)
		restoredTokens = append(restoredTokens, lastToken)
		f.internal.Set(restoredTokens)
	}
}

// parseExprTokens parses a raw HCL expression string into tokens. Fails the test if parsing fails.
func parseExprTokens(t *testing.T, expr string) hclwrite.Tokens {
	t.Helper()
	src := fmt.Sprintf("_x = %s\n", expr)
	file, diags := hclwrite.ParseConfig([]byte(src), "", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "SetRawExpr: failed to parse expression %q as HCL — is it valid HCL syntax? Error: %s", expr, diags.Error())
	return file.Body().GetAttribute("_x").Expr().BuildTokens(nil)
}
