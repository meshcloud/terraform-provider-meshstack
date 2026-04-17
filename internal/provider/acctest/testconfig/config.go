// Package testconfig provides a fluent API for building and modifying HCL configurations
// in Terraform provider acceptance tests.
// This works by loading a Resource{...}.Config() or DataSource{...}.Config() and using Config.WithFirstBlock.
// Configuration can be Config.Join and is conventionally joined in such a way that the block to be modified stays on top.
// A simple usage of that fluent API is the Location or Integration builder.
// A more complex usage is the Tenant builder that composes multiple intermediate builders.
package testconfig

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/stretchr/testify/require"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Resource loads an example resource .tf file by resource name.
// Suffix is optional (useful when multiple example files exist for the same resource).
type Resource struct {
	Name, Suffix string
}

// Config loads the resource's example .tf file and returns a Config. Fails the test on error.
func (r Resource) Config(t *testing.T) Config {
	t.Helper()
	return newConfig(t, examples.Resource.Read(t, r.Name, r.Suffix))
}

// TestSupportConfig loads a test-support .tf file for the resource. Fails the test on error.
// The file is looked up as resources/meshstack_<name>/test-support<Suffix><extraSuffix>.tf.
func (r Resource) TestSupportConfig(t *testing.T, extraSuffix string) Config {
	t.Helper()
	return newConfig(t, examples.Resource.Read(t, r.Name, "test-support", r.Suffix, extraSuffix))
}

// DataSource loads an example data-source .tf file by data source name.
// Suffix is optional (useful when multiple example files exist for the same data-source).
type DataSource struct {
	Name, Suffix string
}

// Config loads the data source's example .tf file and returns a Config. Fails the test on error.
func (d DataSource) Config(t *testing.T) Config {
	t.Helper()
	return newConfig(t, examples.DataSource.Read(t, d.Name, d.Suffix))
}

// Config wraps a parsed *hclwrite.File. All Config methods return a new Config — the receiver is never mutated.
// Call [Config.String] only at the test step boundary (i.e. when assigning to [resource.TestStep.Config]).
type Config struct {
	t        *testing.T
	internal *hclwrite.File
}

// newConfig parses HCL bytes into a Config. Fails the test if parsing fails.
func newConfig(t *testing.T, src []byte) Config {
	t.Helper()
	file, diags := hclwrite.ParseConfig(src, "", hcl.Pos{Line: 1, Column: 1})
	require.False(t, diags.HasErrors(), "failed to parse HCL config (%d bytes): %s", len(src), diags.Error())
	require.NotNil(t, file, "parsed HCL file is nil — check the source bytes")
	return Config{internal: file, t: t}
}

func clone(c Config) Config {
	file, diags := hclwrite.ParseConfig(c.internal.Bytes(), "", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() || file == nil {
		panic(fmt.Sprintf("internal clone failed (this is a bug): %s", diags.Error()))
	}
	return Config{c.t, file}
}

// String renders the config to HCL text.
func (c Config) String() string {
	return string(c.internal.Bytes())
}

// Join combines multiple configs by appending all blocks from others into a new Config.
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

// WithFirstBlock applies consumers to the first block of a cloned Config and returns the new Config.
func (c Config) WithFirstBlock(consumers ...ExpressionConsumer) Config {
	c.t.Helper()
	result := clone(c)
	blocks := result.internal.Body().Blocks()
	require.NotEmpty(c.t, blocks, "WithFirstBlock: config has no blocks — did you load the right .tf file?")
	for _, consumer := range consumers {
		consumer(c.t, Block{c.t, blocks[0]})
	}
	return result
}

// Descend returns a ConsumerBuilder that descends into the given attribute path steps
// and applies the provided consumers to the descended Expression.
//
// Flatten single-child chains into one call:
//
//	Descend("spec", "display_name")(SetString("value"))
//
// Nest only when a parent has multiple children to modify:
//
//	Descend("spec")(
//	    Descend("display_name")(SetString("Updated Name")),
//	    Descend("tags")(SetRawExpr(`{(%s) = ["v"]}`, tagAddr)),
//	)
func Descend(steps ...string) ConsumerBuilder {
	return func(consumers ...ExpressionConsumer) ExpressionConsumer {
		return func(t *testing.T, root Expression) {
			t.Helper()
			traversed := root
			for idx, step := range steps {
				traversable, ok := traversed.(stringTraversable)
				require.True(t, ok,
					"Descend: expected string-traversable expression at step %q (index %d), got %T — cannot descend into this node type",
					step, idx, traversed,
				)
				traversed = traversable.traverse(step)
			}
			for _, consumer := range consumers {
				consumer(t, traversed)
			}
		}
	}
}

// WalkAttributes returns a ConsumerBuilder that visits every attribute reachable from root,
// including nested object attributes, and applies the provided consumers to each Expression.
// Recursion stops at leaf values where stringTraversable.attributes() returns nil (non-object values).
func WalkAttributes() ConsumerBuilder {
	return func(consumers ...ExpressionConsumer) ExpressionConsumer {
		return func(t *testing.T, root Expression) {
			t.Helper()
			var visit func(Expression)
			visit = func(expr Expression) {
				if traversable, ok := expr.(stringTraversable); ok {
					for name := range traversable.attributes() {
						child := traversable.traverse(name)
						for _, consumer := range consumers {
							consumer(t, child)
						}
						visit(child)
					}
				}
			}
			visit(root)
		}
	}
}

// ConsumerBuilder is used by Descend and WalkAttributes to apply the given consumers to the current Expression.
type ConsumerBuilder func(...ExpressionConsumer) ExpressionConsumer

// stringTraversable is used by Descend and WalkAttributes above.
type stringTraversable interface {
	traverse(key string) Expression
	attributes() map[string]*hclwrite.Attribute
}
