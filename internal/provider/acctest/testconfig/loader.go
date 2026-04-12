package testconfig

import (
	"fmt"
	"strings"
	"testing"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// Traversal is a sequence of HCL attribute path segments forming a resource address,
// e.g. Traversal{"meshstack_workspace", "my_ws"} → "meshstack_workspace.my_ws".
// It mirrors hcl.Traversal at a string level — just enough for test reference construction.
type Traversal []string

// String joins segments with ".".
func (t Traversal) String() string {
	return strings.Join(t, ".")
}

// Join appends additional segments, returning a new Traversal.
func (t Traversal) Join(elems ...string) Traversal {
	result := make(Traversal, len(t)+len(elems))
	copy(result, t)
	copy(result[len(t):], elems)
	return result
}

// Format interpolates the traversal string into a format string with optional extra args.
// Keep for complex HCL expression fragments such as "[%s.ref]" or "{ uuid = %s.metadata.uuid }".
func (t Traversal) Format(format string, args ...any) string {
	return fmt.Sprintf(format, append([]any{t.String()}, args...)...)
}

// Resource loads an example resource .tf file by resource name.
// Suffix is optional (useful when multiple example files exist for the same resource).
type Resource struct {
	Name, Suffix string
}

// Config loads the resource's example .tf file and returns a Config. Fails the test on error.
func (r Resource) Config(t *testing.T) Config {
	t.Helper()
	return NewConfig(t, examples.ReadTfFile(t, examples.ResourcePrefix, r.Name, r.Suffix))
}

// TestSupportConfig loads a test-support .tf file for the resource. Fails the test on error.
// The file is looked up as resources/meshstack_<name>/test-support<Suffix><suffix>.tf.
func (r Resource) TestSupportConfig(t *testing.T, suffix string) Config {
	t.Helper()
	return NewConfig(t, examples.ReadTestSupportTfFile(t, r.Name, r.Suffix+suffix))
}

// DataSource loads an example data-source .tf file by data source name.
type DataSource struct {
	Name, Suffix string
}

// Config loads the data source's example .tf file and returns a Config. Fails the test on error.
func (d DataSource) Config(t *testing.T) Config {
	t.Helper()
	return NewConfig(t, examples.ReadTfFile(t, examples.DataSourcePrefix, d.Name, d.Suffix))
}
