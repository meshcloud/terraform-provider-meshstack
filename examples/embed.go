package examples

import "embed"

//go:embed data-sources resources
var examples embed.FS

// ReadFile reads the embedded example at a path relative to examples/, such as
// resources/meshstack_project/resource.tf.
func ReadFile(filePath string) ([]byte, error) {
	return examples.ReadFile(filePath)
}
