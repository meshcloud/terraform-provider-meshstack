package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// TagDefinition builds a plain string tag definition config for the given target kind
// (e.g. "meshWorkspace"). The tag key is randomized per run.
func TagDefinition(t *testing.T, targetKind string) (config Config, tagDefinitionAddr Traversal, tagKey string) {
	t.Helper()
	suffix := acctest.RandString(8)
	tagKey = "test-key-" + suffix
	config = Resource{Name: "tag_definition"}.TestSupportConfig(t, "_tag").WithFirstBlock(
		// Rename the block per run so several tag definitions can coexist in one config (e.g. one a
		// builder like ProjectAndWorkspace already contributes plus a test's own).
		RenameKey("tag_"+suffix),
		ExtractAddress(&tagDefinitionAddr),
		Descend("spec", "target_kind")(SetString(targetKind)),
		Descend("spec", "key")(SetString(tagKey)),
	)
	return config, tagDefinitionAddr, tagKey
}

// RestrictedTagDefinitionWithDefault builds a restricted string tag definition with a default value
// for the given target kind. On create, the backend injects this default into every resource of that
// kind, whether or not the caller declares the tag. The tag key is randomized per run.
func RestrictedTagDefinitionWithDefault(t *testing.T, targetKind, defaultValue string) (config Config, tagDefinitionAddr Traversal, tagKey string) {
	t.Helper()
	suffix := acctest.RandString(8)
	tagKey = "test-key-" + suffix
	config = Resource{Name: "tag_definition"}.TestSupportConfig(t, "_restricted-tag").WithFirstBlock(
		RenameKey("restricted_tag_"+suffix),
		ExtractAddress(&tagDefinitionAddr),
		Descend("spec", "target_kind")(SetString(targetKind)),
		Descend("spec", "key")(SetString(tagKey)),
		Descend("spec", "value_type", "string", "default_value")(SetString(defaultValue)),
	)
	return config, tagDefinitionAddr, tagKey
}
