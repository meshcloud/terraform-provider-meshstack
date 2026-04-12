package provider

import (
	_ "embed"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
)

func TestAccTagDefinitionsDataSource(t *testing.T) {
	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: testconfig.DataSource{Name: "tag_definitions"}.Config(t).String(),
			},
		},
	})
}
