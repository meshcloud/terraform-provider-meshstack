package examples

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	//go:embed data-sources
	dataSources embed.FS
	//go:embed resources
	resources embed.FS
)

type Example string

const (
	DataSource Example = "data-source"
	Resource   Example = "resource"
)

// Read reads the embedded example Terraform code for the named resource/datasource,
// If file name parts are empty or only one is provided, the parts are prefixed with the Example value as string:
// The empty case reflects the simple case of
// examples/resources/meshstack_<name>/resource.tf or
// examples/data-sources/meshstack_<name>/data-source.tf, respectively.
// The case of one part adds a suffix to the .tf files.
func (e Example) Read(t *testing.T, name string, fileNameParts ...string) []byte {
	t.Helper()
	var fsys fs.ReadFileFS
	switch e {
	case DataSource:
		fsys = dataSources
	case Resource:
		fsys = resources
	default:
		t.Fatalf("unknown example type: %s", e)
	}
	switch len(fileNameParts) {
	case 0:
		fileNameParts = []string{string(e)}
	case 1:
		fileNameParts = []string{string(e), fileNameParts[0]}
	default:
		// do nothing for custom filePrefix+fileSuffixes... (such as test-support*.tf files),
		// just join them below.
	}
	filePath := path.Join(
		string(e)+"s",
		"meshstack_"+name,
		strings.Join(append(fileNameParts, ".tf"), ""),
	)
	content, err := fsys.ReadFile(filePath)
	require.NoErrorf(t, err, "cannot read examples config file %s:", filePath)
	return content
}

// TestStepConfig assembles the HCL for one acceptance test step: the step's own variant of the
// example (<resource|data-source>-test-<index>.tf) followed by the named test-support files
// (test-support_<name>.tf) holding the prerequisites that variant references.
//
// One file per step keeps every applied config readable as plain HCL and diffable against the
// documented example; the index is the link between the file and the step that uses it.
func (e Example) TestStepConfig(t *testing.T, name string, index int, supportNames ...string) string {
	t.Helper()
	parts := []string{string(e.Read(t, name, fmt.Sprintf("-test-%d", index)))}
	for _, supportName := range supportNames {
		parts = append(parts, string(e.Read(t, name, "test-support", "_"+supportName)))
	}
	return strings.Join(parts, "\n")
}
