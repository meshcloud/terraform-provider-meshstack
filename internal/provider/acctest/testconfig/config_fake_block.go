package testconfig

import (
	"fmt"
	"iter"
	"maps"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// fakeBlock wraps an attributeExpression to implement the parent interface, enabling
// [hclwrite.Body] operations (Attributes, SetAttributeRaw, RenameAttribute) on object constructor
// expressions like `{ key = "value" }`.
//
// This is necessary because [hclwrite] distinguishes between top-level block bodies and inline
// object constructor expressions at the token level. A block body is directly accessible via
// [hclwrite.Block.Body], but an attribute whose value is `{ ... }` is just raw tokens —
// [hclwrite] provides no API to manipulate its inner keys.
//
// fakeBlock bridges this gap by stripping the outer braces, parsing the inner tokens as a
// temporary [hclwrite.Body] via [hclwrite.ParseConfig], applying the requested modification,
// then reassembling the tokens with braces restored. Quoted attribute names like `"dashed-key"`
// (invalid as HCL identifiers) are temporarily replaced with sanitized placeholders via
// unquoteAttributeNames before parsing, then restored via requoteAttributeNames.
type fakeBlock struct {
	t        *testing.T
	internal attributeExpression
}

func (f fakeBlock) Attributes() (result map[string]*hclwrite.Attribute) {
	f.withBody(func(body *hclwrite.Body, names quotedNameMap) (modified bool) {
		attributes := body.Attributes()
		result = make(map[string]*hclwrite.Attribute, len(attributes))
		for sanitized, unsanitized := range names.Restore(maps.Keys(attributes)) {
			result[unsanitized] = attributes[sanitized]
		}
		return
	})
	return
}

func (f fakeBlock) SetAttributeRaw(name string, tokens hclwrite.Tokens) (result *hclwrite.Attribute) {
	f.withBody(func(body *hclwrite.Body, names quotedNameMap) (modified bool) {
		result = body.SetAttributeRaw(names.Sanitize(name), tokens)
		return true
	})
	return
}

func (f fakeBlock) RenameAttribute(fromName, toName string) (result bool) {
	f.withBody(func(body *hclwrite.Body, names quotedNameMap) (modified bool) {
		result = body.RenameAttribute(names.Sanitize(fromName), names.Sanitize(toName))
		return true
	})
	return
}

// withBody parses the attribute's tokens as an object body and calls the modifier.
// If the tokens don't form an object ({...}), the modifier is never called.
func (f fakeBlock) withBody(modifier func(body *hclwrite.Body, names quotedNameMap) (modified bool)) {
	tokens := f.internal.Get()
	if len(tokens) < 2 {
		return
	}
	if tokens[0].Type != hclsyntax.TokenOBrace {
		return
	}
	if tokens[len(tokens)-1].Type != hclsyntax.TokenCBrace {
		return
	}
	firstToken := tokens[0]
	lastToken := tokens[len(tokens)-1]
	strippedTokens := tokens[1 : len(tokens)-1]
	unquotedTokens, names := unquoteAttributeNames(strippedTokens)
	file, diags := hclwrite.ParseConfig(unquotedTokens.Bytes(), "", hcl.Pos{})
	if diags.HasErrors() || file == nil {
		f.t.Fatalf("fakeBlock internal error: failed to parse inner tokens: %s", diags.Error())
	}
	strippedBody := file.Body()
	if modifier(strippedBody, names) {
		restoredTokens := strippedBody.BuildTokens(hclwrite.Tokens{firstToken})
		restoredTokens = requoteAttributeNames(restoredTokens, names.Invert())
		restoredTokens = append(restoredTokens, lastToken)
		f.internal.Set(restoredTokens)
	}
}

// quotedNameMap holds mapping of unsanitized (quoted) to sanitized names (valid attribute identifier, unquoted).
type quotedNameMap map[string]string

// Sanitize sanitizes the given name if required, otherwise returns name as-is.
func (m quotedNameMap) Sanitize(name string) string {
	if sanitized, ok := m[name]; ok {
		return sanitized
	}
	return name
}

func (m quotedNameMap) Invert() map[string]string {
	inverted := make(map[string]string, len(m))
	for unsanitized, sanitized := range m {
		inverted[sanitized] = unsanitized
	}
	return inverted
}

func (m quotedNameMap) Restore(names iter.Seq[string]) iter.Seq2[string, string] {
	inverted := m.Invert()
	return func(yield func(string, string) bool) {
		for name := range names {
			mapped := name
			if unsanitized, ok := inverted[name]; ok {
				mapped = unsanitized
			}
			if !yield(name, mapped) {
				return
			}
		}
	}
}

// unquoteAttributeNames replaces quoted attribute name tokens (OQuote+QuotedLit+CQuote before Equal)
// with a single TokenIdent so that hclwrite.ParseConfig accepts them.
// Only replaces at brace depth 0 to avoid mangling keys inside nested object expressions.
func unquoteAttributeNames(tokens hclwrite.Tokens) (hclwrite.Tokens, quotedNameMap) {
	names := quotedNameMap{}
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
			sanitized, exists := names[original]
			if !exists {
				sanitized = fmt.Sprintf("__q%d__", counter)
				counter++
				names[original] = sanitized
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
	return result, names
}

// requoteAttributeNames is the inverse of unquoteAttributeNames: it restores the original quoted
// attribute name syntax ("key" =) from the sanitized identifier tokens (key =) using the mapping.
// This is needed because HCL attribute names like "display-name" contain characters invalid in
// identifiers, so we temporarily sanitize them for hclwrite, then restore the original form.
func requoteAttributeNames(tokens hclwrite.Tokens, inverted map[string]string) hclwrite.Tokens {
	var result hclwrite.Tokens
	for i := 0; i < len(tokens); i++ {
		if tokens[i].Type == hclsyntax.TokenIdent {
			if original, ok := inverted[string(tokens[i].Bytes)]; ok && i+1 < len(tokens) && tokens[i+1].Type == hclsyntax.TokenEqual {
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
