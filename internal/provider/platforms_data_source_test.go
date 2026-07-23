package provider

import (
	"fmt"
	"testing"

	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func TestAccPlatformsDataSource(t *testing.T) {
	t.Parallel()

	// plain listing creates a platform in a fresh workspace and lists it back, running identically in
	// mock and acceptance mode. Filtering by the fresh workspace yields exactly one platform.
	t.Run("plain listing", func(t *testing.T) {
		platformConfig, platformAddr, workspaceAddr := testconfig.CustomPlatformAndWorkspace(t)

		dataSourceAddr := "data.meshstack_platforms.published"
		config := testconfig.DataSource{Name: "platforms"}.Config(t).WithFirstBlock(
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(workspaceAddr, "metadata", "name")),
			testconfig.Descend("depends_on")(testconfig.SetRawExpr("[%s]", platformAddr)),
		).Join(platformConfig)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("identifier"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(dataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("availability").AtMapKey("publication_state"), knownvalue.StringExact("PUBLISHED")),
					},
				},
			},
		})
	})

	// cross-workspace listing proves a consumer workspace's restricted key can list a platform published
	// (RESTRICTED) to it (P_pub, positive, both modes) but NOT a platform it is not entitled to (P_priv,
	// negative). The exactly-one boundary that proves P_priv's exclusion and the config-redaction check
	// are acceptance-only: the mock has no entitlement notion (it applies only plain attribute filters).
	t.Run("cross-workspace listing", func(t *testing.T) {
		operatorWorkspaceConfig, operatorWorkspaceAddr := testconfig.Workspace(t)
		platformConfig, platformAddr, platformTypeAddr := testconfig.CustomPlatform(t, operatorWorkspaceAddr)

		// consumer workspace ("other") holds the restricted api key.
		var consumerWorkspaceAddr testconfig.Traversal
		consumerWorkspaceConfig, _ := testconfig.Workspace(t)
		consumerWorkspaceConfig = consumerWorkspaceConfig.WithFirstBlock(
			testconfig.RenameKey("other"),
			testconfig.ExtractAddress(&consumerWorkspaceAddr),
		)

		// P_pub: RESTRICTED + PUBLISHED, restricted to operator + consumer (proves consumer-specific entitlement).
		platformConfig = platformConfig.WithFirstBlock(
			testconfig.Descend("spec", "availability")(
				testconfig.Descend("restriction")(testconfig.SetString("RESTRICTED")),
				testconfig.Descend("publication_state")(testconfig.SetString("PUBLISHED")),
				testconfig.Descend("restricted_to_workspaces")(testconfig.SetRawExpr("[%s, %s]",
					operatorWorkspaceAddr.Join("metadata", "name"), consumerWorkspaceAddr.Join("metadata", "name"))),
			),
		)

		// P_priv: PRIVATE + UNPUBLISHED and NOT shared with the consumer. It is also owned by the
		// operator (so the owner-scoped filter would match) and reuses P_pub's platform type; the
		// consumer's restricted key must not be able to list it. A distinct resource key avoids a
		// collision with P_pub, which is built from the same example resource.
		var privPlatformAddr testconfig.Traversal
		privPlatformConfig := testconfig.Resource{Name: "platform", Suffix: "_08_custom"}.Config(t).WithFirstBlock(
			testconfig.RenameKey("priv_custom"),
			testconfig.ExtractAddress(&privPlatformAddr),
			testconfig.OwnedByWorkspace(operatorWorkspaceAddr),
			testconfig.Descend("metadata", "name")(testconfig.SetString("priv-"+acctest.RandString(8))),
			testconfig.Descend("spec", "config", "custom", "platform_type_ref")(testconfig.SetAddr(platformTypeAddr, "ref")),
			testconfig.Descend("spec", "availability")(
				testconfig.Descend("restriction")(testconfig.SetString("PRIVATE")),
				testconfig.Descend("publication_state")(testconfig.SetString("UNPUBLISHED")),
				// a PRIVATE platform must list exactly its owner (backend validation); it is still not
				// shared with the consumer, so the consumer's restricted key must not see it.
				testconfig.Descend("restricted_to_workspaces")(testconfig.SetRawExpr("[%s]",
					operatorWorkspaceAddr.Join("metadata", "name"))),
			),
		)

		apiKeyConfig, apiKeyAddr := testconfig.ApiKey(t, consumerWorkspaceAddr)
		apiKeyConfig = apiKeyConfig.WithFirstBlock(
			testconfig.Descend("spec", "permissions")(testconfig.SetRawExpr(`["PLATFORMINSTANCE_LIST"]`)),
		)

		supportConfig := platformConfig.Join(operatorWorkspaceConfig, consumerWorkspaceConfig, apiKeyConfig, privPlatformConfig)

		var dataSourceAddress testconfig.Traversal
		example := testconfig.DataSource{Name: "platforms"}
		config := example.Config(t).WithFirstBlock(
			testconfig.ExtractAddress(&dataSourceAddress),
			testconfig.Descend("owned_by_workspace")(testconfig.SetAddr(operatorWorkspaceAddr, "metadata", "name")),
			// use the restricted consumer api key via the meshstack-other provider alias
			testconfig.Descend("provider")(testconfig.SetRawExpr("meshstack-other")),
		).Join(supportConfig, testconfig.OtherProviderConfig(t))

		// pubPlatformUuid is captured from P_pub in the setup step and asserted to be the (only) platform
		// the consumer lists in the second step, proving P_priv is absent.
		var pubPlatformUuid string
		listChecks := []statecheck.StateCheck{
			// Positive present check (both modes): P_pub is the first (and, in acceptance, only) listed platform.
			statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
				if uuid != pubPlatformUuid {
					return fmt.Errorf("expected first listed platform to be P_pub (uuid %s), got %s", pubPlatformUuid, uuid)
				}
				return nil
			})),
			statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
			statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("availability").AtMapKey("publication_state"), knownvalue.StringExact("PUBLISHED")),
		}
		if !IsMockClientTest() {
			// Decisive negative entitlement assertion: the consumer's restricted key lists exactly P_pub;
			// P_priv (PRIVATE + UNPUBLISHED, not shared to the consumer) is filtered out by the marketplace
			// WHERE-clause. The mock has no entitlement notion (it only applies plain attribute filters, so
			// it would drop P_priv merely on publication_state), so exact-size — the proof of exclusion —
			// and config redaction are acceptance-only. Config redaction: a marketplace consumer receives
			// the platform with spec.config omitted entirely, which the provider surfaces as a null config.
			listChecks = append(listChecks,
				statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms"), knownvalue.ListSizeExact(1)),
				statecheck.ExpectKnownValue(dataSourceAddress.String(), tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("config"), knownvalue.Null()),
			)
		}

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config: supportConfig.String(),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyAddr.String(), tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
					// capture P_pub's uuid for the list assertion (also keeps platformAddr wired into the config graph)
					statecheck.ExpectKnownValue(platformAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
						pubPlatformUuid = uuid
						return nil
					})),
					// referenced so the linter/compiler keep privPlatformAddr (P_priv) wired into the config graph
					statecheck.ExpectKnownValue(privPlatformAddr.String(), tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config: config.String(),
				ConfigVariables: tfconfig.Variables{
					"apikey_client_id":     &apiKeyClientId,
					"apikey_client_secret": &apiKeyClientSecret,
				},
				ConfigStateChecks: listChecks,
			},
		}})
	})
}
