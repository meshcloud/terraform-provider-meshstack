package examples

import (
	"embed"
	"fmt"
	"path"
	"testing"
)

var (
	//go:embed data-sources
	dataSources embed.FS
	//go:embed resources
	resources embed.FS
)

type embeddedPrefix string

// EmbeddedPrefix is the constraint for the typed prefix constants.
type EmbeddedPrefix = embeddedPrefix

const (
	DataSourcePrefix  embeddedPrefix = "data-source"
	ResourcePrefix    embeddedPrefix = "resource"
	TestSupportPrefix embeddedPrefix = "test-support"
)

// ReadTfFile reads a .tf file from the embedded FS for the given prefix and resource name.
// The given prefix selects both the top-level directory (data-sources/resources)
// and the file prefix within that directory.
// Fails the test on error.
func ReadTfFile(t *testing.T, prefix embeddedPrefix, name, fileSuffix string) []byte {
	t.Helper()
	return readTfFile(t, prefix, prefix, name, fileSuffix)
}

// ReadTestSupportTfFile reads a test-support .tf file from the resources embedded FS.
// Test-support files live alongside resource examples (resources/meshstack_<name>/test-support<suffix>.tf).
// Fails the test on error.
func ReadTestSupportTfFile(t *testing.T, name, fileSuffix string) []byte {
	t.Helper()
	return readTfFile(t, ResourcePrefix, TestSupportPrefix, name, fileSuffix)
}

func readTfFile(t *testing.T, dirPrefix, filePrefix embeddedPrefix, name, fileSuffix string) []byte {
	t.Helper()
	fsys := resources
	if dirPrefix == DataSourcePrefix {
		fsys = dataSources
	}
	filePath := path.Join(string(dirPrefix)+"s", "meshstack_"+name, fmt.Sprintf("%s%s.tf", filePrefix, fileSuffix))
	content, err := fsys.ReadFile(filePath)
	if err != nil {
		t.Fatalf("cannot read examples config file %q: %s", filePath, err.Error())
	}
	return content
}
