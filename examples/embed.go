package examples

import (
	"embed"
	"fmt"
	"io/fs"
	"iter"
	"path"
	"regexp"
	"strings"
)

var (
	//go:embed data-sources
	dataSources embed.FS
	//go:embed resources
	resources embed.FS
)

type Resource struct {
	// Name is required, Suffix is optional (useful when multiple example files are present).
	Name, Suffix string
}

func (r Resource) Config() Config {
	return readConfigFromTfFile(resources, resourcePrefix, resourcePrefix, r.Name, r.Suffix)
}

func (r Resource) TestSupportConfig(suffix string) Config {
	return readConfigFromTfFile(resources, resourcePrefix, testSupportPrefix, r.Name, r.Suffix+suffix)
}

func (r Resource) All() iter.Seq[Resource] {
	return func(yield func(Resource) bool) {
		for suffix := range listTfFileSuffixesInDir(resources, resourcePrefix, r.Name) {
			if !yield(Resource{Name: r.Name, Suffix: suffix}) {
				return
			}
		}
	}
}

type DataSource struct {
	// Name is required, Suffix is optional (useful when multiple example files are present).
	Name, Suffix string
}

func (d DataSource) Config() Config {
	return readConfigFromTfFile(dataSources, dataSourcePrefix, dataSourcePrefix, d.Name, d.Suffix)
}

type Config string

func (c Config) ReplaceAll(old, replacement string) (modified Config) {
	backup := c
	modified = Config(strings.ReplaceAll(string(c), old, replacement))
	if modified == backup {
		panic(fmt.Sprintf("Replacing '%s' -> '%s' in example config had no effect", old, replacement))
	}
	return
}

func (c Config) OwnedByAdminWorkspace() Config {
	return c.ReplaceAll(`owned_by_workspace = "my-workspace"`, `owned_by_workspace = "managed-customer"`)
}

func (c Config) SingleResourceAddress(out *Identifier) Config {
	// See https://regex101.com/r/f4Wiey/2
	re := regexp.MustCompile(`(?m)^resource "(?P<type>[^"]+)" "(?P<name>[^"]+)"`)
	if matches := re.FindAllStringSubmatch(string(c), -1); len(matches) == 1 {
		*out = Identifier{
			matches[0][re.SubexpIndex("type")],
			matches[0][re.SubexpIndex("name")],
		}
	} else {
		panic(fmt.Sprintf("not exactly one single resource found, but %d", len(matches)))
	}
	return c
}

func (c Config) Join(others ...Config) (joined Config) {
	joined = c
	for _, other := range others {
		joined += "\n\n" + other
	}
	return
}

func (c Config) String() string {
	return string(c)
}

type Identifier []string

func (a Identifier) Join(elems ...string) Identifier {
	return append(a, elems...)
}

func (a Identifier) String() string {
	return strings.Join(a, ".")
}

func (a Identifier) Format(format string, args ...any) string {
	return fmt.Sprintf(format, append([]any{a.String()}, args...)...)
}

type embeddedPrefix string

const (
	dataSourcePrefix  embeddedPrefix = "data-source"
	resourcePrefix    embeddedPrefix = "resource"
	testSupportPrefix embeddedPrefix = "test-support"
)

func listTfFileSuffixesInDir(fsys fs.ReadDirFS, prefix embeddedPrefix, name string) iter.Seq[string] {
	dir := path.Join(string(prefix)+"s", "meshstack_"+name)
	entries, err := fsys.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("error listing dir %s: %s", dir, err.Error()))
	}
	return func(yield func(string) bool) {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), string(prefix)) && strings.HasSuffix(entry.Name(), ".tf") {
				suffix := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), string(prefix)), ".tf")
				if !yield(suffix) {
					return
				}
			}
		}
	}
}

func readConfigFromTfFile(fsys fs.ReadFileFS, prefix, filePrefix embeddedPrefix, name, fileSuffix string) Config {
	filePath := path.Join(string(prefix)+"s", "meshstack_"+name, fmt.Sprintf("%s%s.tf", filePrefix, fileSuffix))
	content, err := fsys.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("cannot read examples config file '%s': %s", filePath, err.Error()))
	}
	return Config(content)
}
