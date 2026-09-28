package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// The documented example applies unchanged — it takes no filters and only reads — so this step uses
// it directly instead of a -test- variant.
func TestAccTagDefinitionsDataSource(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: string(examples.DataSource.Read(t, "tag_definitions")),
			},
		},
	})
}
