package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

// The documented examples apply unchanged here — they only read, and name a workspace the query does
// not need to find — so these steps use them directly instead of a -test- variant.
func TestAccProjectsDatasource(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: string(examples.DataSource.Read(t, "projects", "_all")),
			},
			{
				Config: string(examples.DataSource.Read(t, "projects", "_payment_method")),
			},
		},
	})
}
