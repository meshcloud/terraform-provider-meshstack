package testconfig

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

func BuildLocationConfig(t *testing.T, workspaceAddr Traversal) (config Config, locationAddr Traversal) {
	t.Helper()
	locationName := "my-location-" + acctest.RandString(32)
	return Resource{Name: "location"}.Config(t).WithFirstBlock(t,
		ExtractIdentifier(&locationAddr),
		OwnedByWorkspace(t, workspaceAddr),
		Traverse(t, "metadata")(Traverse(t, "name")(SetString(locationName))),
	), locationAddr
}
