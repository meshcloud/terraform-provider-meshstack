package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccProjectsDatasource(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testconfig.DataSource{Name: "projects", Suffix: "_all"}.Config(t).String(),
			},
			{
				Config: testconfig.DataSource{Name: "projects", Suffix: "_payment_method"}.Config(t).String(),
			},
		},
	})
}
