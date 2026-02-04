package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccProjectsDatasource(t *testing.T) {
	// this very minimal test is already useful as it runs a paginated request against the API to receive projects
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProviderFactoriesForTest(),
		PreCheck:                 func() { DefaultTestPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: examples.DataSource{Name: "projects", Suffix: "_all"}.Config().String(),
			},
			{
				Config: examples.DataSource{Name: "projects", Suffix: "_payment_method"}.Config().String(),
			},
		},
	})
}
