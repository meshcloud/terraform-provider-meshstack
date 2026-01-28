package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccTagDefinitionsDataSource(t *testing.T) {
	// this very minimal test is already useful as it runs a request against the API to receive tag definitions
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		PreCheck:                 func() { testAccPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: examples.DataSource{Name: "tag_definitions"}.String(),
			},
		},
	})
}
