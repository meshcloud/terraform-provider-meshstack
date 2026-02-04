package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
)

func TestAccIntegrationsDataSource(t *testing.T) {
	// this very minimal test is already useful as it runs a request against the API to receive integrations
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: ProviderFactoriesForTest(),
		PreCheck:                 func() { DefaultTestPreCheck(t) },
		Steps: []resource.TestStep{
			{
				Config: examples.DataSource{Name: "integrations"}.Config().String(),
			},
		},
	})
}
