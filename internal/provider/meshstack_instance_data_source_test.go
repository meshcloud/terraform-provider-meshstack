package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccInstanceDataSource(t *testing.T) {
	t.Parallel()

	dataSourceAddress := testconfig.Traversal{"data.meshstack_instance", "this"}
	config := testconfig.DataSource{Name: "instance"}.Config(t)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("endpoint"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("version"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("is_four_eyes_enabled"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("admin_workspace_identifier"), knownvalue.StringExact(AdminWorkspaceIdentifier)),
				},
			},
		},
	})
}
