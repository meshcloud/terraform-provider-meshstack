package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// Location builds a location config owned by the given workspace.
func Location(t *testing.T, workspaceAddr Traversal) (config Config, locationAddr Traversal, locationName string) {
	t.Helper()
	locationName = "my-location-" + acctest.RandString(32)
	return Resource{Name: "location"}.Config(t).WithFirstBlock(
		ExtractAddress(&locationAddr),
		OwnedByWorkspace(workspaceAddr),
		Descend("metadata", "name")(SetString(locationName)),
	), locationAddr, locationName
}
