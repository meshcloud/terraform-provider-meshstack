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

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks the cross-workspace listing needs. P_pub and P_priv are both owned by the
// operator workspace and share a platform type, so only entitlement separates them.
const (
	platformsDataSourceAddr = "data.meshstack_platforms.published"
	privatePlatformAddr     = "meshstack_platform.priv_custom"
)

func TestAccPlatformsDataSource(t *testing.T) {
	t.Parallel()

	// plain listing creates a platform in a fresh workspace and lists it back, running identically in
	// mock and acceptance mode. Filtering by the fresh workspace yields exactly one platform.
	t.Run("plain listing", func(t *testing.T) {
		config := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "platforms", 1),
			platformStepConfig(t, platformVariants[7]),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          config,
					ConfigVariables: SuffixVariables(acctest.RandString(8)),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms"), knownvalue.ListSizeExact(1)),
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("identifier"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
						statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("availability").AtMapKey("publication_state"), knownvalue.StringExact("PUBLISHED")),
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
		vars := SuffixVariables(acctest.RandString(8))

		// The two platforms, both workspaces, the platform type they share and the consumer's key.
		supportConfig := examples.JoinTestStepConfigs(
			examples.Resource.TestStepConfig(t, "platform", 9),
			examples.Resource.TestStepConfig(t, "platform", 10),
			examples.Resource.TestStepConfig(t, "platform_type", 1),
			examples.Resource.TestStepConfig(t, "api_key", 4),
			examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites", "consumer-workspace"),
		)

		listConfig := examples.JoinTestStepConfigs(
			examples.DataSource.TestStepConfig(t, "platforms", 2),
			supportConfig,
			examples.Resource.TestSupportConfigs(t, "api_key", "other_provider"),
		)

		// pubPlatformUuid is captured from P_pub in the setup step and asserted to be the (only) platform
		// the consumer lists in the second step, proving P_priv is absent.
		var pubPlatformUuid string
		listChecks := []statecheck.StateCheck{
			// Positive present check (both modes): P_pub is the first (and, in acceptance, only) listed platform.
			statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
				if uuid != pubPlatformUuid {
					return fmt.Errorf("expected first listed platform to be P_pub (uuid %s), got %s", pubPlatformUuid, uuid)
				}
				return nil
			})),
			statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("kind"), knownvalue.StringExact("meshPlatform")),
			statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("ref").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
			statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("availability").AtMapKey("publication_state"), knownvalue.StringExact("PUBLISHED")),
		}
		if !IsMockClientTest() {
			// Decisive negative entitlement assertion: the consumer's restricted key lists exactly P_pub;
			// P_priv (PRIVATE + UNPUBLISHED, not shared to the consumer) is filtered out by the marketplace
			// WHERE-clause. The mock has no entitlement notion (it only applies plain attribute filters, so
			// it would drop P_priv merely on publication_state), so exact-size — the proof of exclusion —
			// and config redaction are acceptance-only. Config redaction: a marketplace consumer receives
			// the platform with spec.config omitted entirely, which the provider surfaces as a null config.
			listChecks = append(listChecks,
				statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms"), knownvalue.ListSizeExact(1)),
				statecheck.ExpectKnownValue(platformsDataSourceAddr, tfjsonpath.New("platforms").AtSliceIndex(0).AtMapKey("spec").AtMapKey("config"), knownvalue.Null()),
			)
		}

		var apiKeyClientId, apiKeyClientSecret lazyVariable
		listVars := tfconfig.Variables{
			"apikey_client_id":     &apiKeyClientId,
			"apikey_client_secret": &apiKeyClientSecret,
		}
		for name, value := range vars {
			listVars[name] = value
		}

		ApplyAndTest(t, resource.TestCase{Steps: []resource.TestStep{
			{
				Config:          supportConfig,
				ConfigVariables: vars,
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_id"), xknownvalue.NotEmptyString(func(clientId string) error {
						apiKeyClientId = lazyVariable(clientId)
						return nil
					})),
					statecheck.ExpectKnownValue(apiKeyResourceAddr, tfjsonpath.New("status").AtMapKey("client_secret"), xknownvalue.NotEmptyString(func(clientSecret string) error {
						apiKeyClientSecret = lazyVariable(clientSecret)
						return nil
					})),
					// capture P_pub's uuid for the list assertion
					statecheck.ExpectKnownValue(platformVariants[7].addr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString(func(uuid string) error {
						pubPlatformUuid = uuid
						return nil
					})),
					statecheck.ExpectKnownValue(privatePlatformAddr, tfjsonpath.New("metadata").AtMapKey("uuid"), xknownvalue.NotEmptyString()),
				},
			},
			{
				Config:            listConfig,
				ConfigVariables:   listVars,
				ConfigStateChecks: listChecks,
			},
		}})
	})
}
