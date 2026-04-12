package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformTypeDataSource(t *testing.T) {
	workspaceConfig, workspaceAddr := testconfig.BuildWorkspaceConfig(t)
	platformTypeConfig, platformTypeAddr := testconfig.BuildPlatformTypeConfig(t, workspaceAddr)

	dataSourceAddress := testconfig.Traversal{"data.meshstack_platform_type", "example"}
	dsConfig := testconfig.DataSource{Name: "platform_type"}.Config(t)
	dsConfig = dsConfig.WithFirstBlock(t,
		testconfig.Traverse(t, "metadata")(
			testconfig.Traverse(t, "owned_by_workspace")(testconfig.SetRawExpr(workspaceAddr.Format("%s.metadata.name"))),
			testconfig.Traverse(t, "name")(testconfig.SetRawExpr(platformTypeAddr.Format("%s.metadata.name"))),
		))
	config := dsConfig.Join(workspaceConfig, platformTypeConfig)

	ApplyAndTest(t, resource.TestCase{
		Steps: []resource.TestStep{
			{
				Config: config.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("metadata"), checkPlatformTypeMetadata()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("spec").AtMapKey("display_name"), xknownvalue.NotEmptyString()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("status"), checkPlatformTypeStatus()),
					statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("ref"), checkPlatformTypeRef()),
				},
			},
		},
	})
}
