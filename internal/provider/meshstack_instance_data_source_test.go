package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccInstanceDataSource(t *testing.T) {
	const dataSourceAddress = "data.meshstack_instance.this"

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: string(examples.DataSource.Read(t, "instance")),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("endpoint"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("version"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("enabled_feature_flags"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("metadata"), knownvalue.NotNull()),
					statecheck.ExpectKnownValue(dataSourceAddress, tfjsonpath.New("admin_workspace_identifier"), knownvalue.StringExact(AdminWorkspaceIdentifier)),
				},
			},
		},
	})
}
