package testconfig

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// PlatformType builds a platform type config owned by the given workspace.
func PlatformType(t *testing.T, workspaceAddr Traversal) (config Config, platformTypeAddr Traversal) {
	t.Helper()
	platformTypeSuffix := strings.ToUpper(acctest.RandString(8))
	platformTypeName := "CUSTOM-PT-" + platformTypeSuffix
	config = Resource{Name: "platform_type"}.Config(t).WithFirstBlock(
		ExtractAddress(&platformTypeAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(platformTypeName)),
		Descend("spec", "display_name")(SetString("My Custom Platform "+platformTypeSuffix)),
	)
	return config, platformTypeAddr
}
