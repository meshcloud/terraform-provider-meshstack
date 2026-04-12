package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildTagDefinitionConfig(t *testing.T, targetKind string) (config Config, tagDefinitionAddr Traversal) {
	t.Helper()
	keySuffix := acctest.RandString(8)
	tagKey := "test-key-" + keySuffix
	config = Resource{Name: "tag_definition"}.Config(t).WithFirstBlock(t,
		RenameKey("example_"+keySuffix),
		ExtractIdentifier(&tagDefinitionAddr),
		Traverse(t, "spec")(
			Traverse(t, "target_kind")(SetString(targetKind)),
			Traverse(t, "key")(SetString(tagKey)),
			Traverse(t, "value_type", "email")(RenameKey("string")),
		),
	)
	return config, tagDefinitionAddr
}
