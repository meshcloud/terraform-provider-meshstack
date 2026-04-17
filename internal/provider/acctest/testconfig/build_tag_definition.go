package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// TagDefinition builds a tag definition config for the given target kind (e.g. "meshWorkspace").
func TagDefinition(t *testing.T, targetKind string) (config Config, tagDefinitionAddr Traversal, tagKey string) {
	t.Helper()
	keySuffix := acctest.RandString(8)
	tagKey = "test-key-" + keySuffix
	config = Resource{Name: "tag_definition"}.Config(t).WithFirstBlock(
		RenameKey("example_"+keySuffix),
		ExtractAddress(&tagDefinitionAddr),
		Descend("spec")(
			Descend("target_kind")(SetString(targetKind)),
			Descend("key")(SetString(tagKey)),
			Descend("value_type", "email")(RenameKey("string")),
		),
	)
	return config, tagDefinitionAddr, tagKey
}
