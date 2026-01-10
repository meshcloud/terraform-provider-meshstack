package examples

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
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

func (r Resource) String() string {
	return readEmbeddedFile(resources, resourcePrefix, r.Name, r.Suffix)
}

type DataSource struct {
	// Name is required, Suffix is optional (useful when multiple example files are present).
	Name, Suffix string
}

func (d DataSource) String() string {
	return readEmbeddedFile(dataSources, dataSourcePrefix, d.Name, d.Suffix)
}

type embeddedPrefix string

const (
	dataSourcePrefix embeddedPrefix = "data-source"
	resourcePrefix   embeddedPrefix = "resource"
)

func readEmbeddedFile(fsys fs.ReadFileFS, prefix embeddedPrefix, name, fileSuffix string) string {
	filePath := path.Join(string(prefix)+"s", "meshstack_"+name, fmt.Sprintf("%s%s.tf", prefix, fileSuffix))
	content, err := fsys.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("cannot open embedded file '%s': %s", filePath, err))
	}
	return string(content)
}
