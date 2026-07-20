package provider

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/zclconf/go-cty/cty"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/testconfig"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

func updateBBDDescription(t *testing.T, config testconfig.Config, newDescription string) testconfig.Config {
	t.Helper()
	return config.WithFirstBlock(
		testconfig.Descend("spec", "description")(testconfig.SetString(newDescription)),
	)
}

func releaseBBDVersion(t *testing.T, config testconfig.Config) testconfig.Config {
	t.Helper()
	return config.WithFirstBlock(
		testconfig.Descend("version_spec", "draft")(testconfig.SetValue(cty.False)),
	)
}

func TestAccBuildingBlockDefinition(t *testing.T) {
	t.Parallel()

	var (
		versionStateDraft    = client.MeshBuildingBlockDefinitionVersionStateDraft
		versionStateReleased = client.MeshBuildingBlockDefinitionVersionStateReleased
	)

	expectedVersion := func(number int64, state enum.Entry[client.MeshBuildingBlockDefinitionVersionState]) knownvalue.Check {
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":         xknownvalue.NotEmptyString(),
			"number":       knownvalue.Int64Exact(number),
			"state":        knownvalue.StringExact(state.String()),
			"content_hash": xknownvalue.NotEmptyString(),
			"kind":         knownvalue.StringExact(client.MeshObjectKind.BuildingBlockDefinitionVersion),
		})
	}

	const bbdDescription = "An example building block definition"

	t.Run("01_terraform", func(t *testing.T) {
		config, addr := testconfig.BBDTerraform(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataFull()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecFull(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("01_terraform", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Change secret input name (remove/add operation on inputs map)
				{
					Config: func() string {
						u := config.WithFirstBlock(
							testconfig.Descend("version_spec", "inputs", "SOMETHING_VERY_SECRET")(testconfig.RenameKey("SOMETHING_VERY_SECRET_RENAMED")))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("02_github_workflows", func(t *testing.T) {
		config, addr := testconfig.BBDWithIntegration(t, "02_github_workflows")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("02_github_workflows", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("03_manual", func(t *testing.T) {
		config, addr := testconfig.BBDManual(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Update spec (description change, no new version)
				{
					Config: updateBBDDescription(t, config, "An updated building block definition").String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal("An updated building block definition")),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Release (draft=false)
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateReleased)})),
					},
				},
				// Step 4: New draft (draft=true again, description changed)
				{
					Config: updateBBDDescription(t, config, "An updated building block definition").String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 2)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
					},
				},
				// Step 5: Release the new draft (draft=false)
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 2)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateReleased),
						})),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("04_azure_devops_pipeline", func(t *testing.T) {
		config, addr := testconfig.BBDWithIntegration(t, "04_azure_devops_pipeline")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("04_azure_devops_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
			},
		})
	})

	t.Run("05_gitlab_pipeline", func(t *testing.T) {
		config, addr := testconfig.BBDGitlabPipeline(t)
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Plan-only (ensure tf plan works before apply)
				{
					Config:             config.String(),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				// Step 2: Create
				{
					Config: config.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("05_gitlab_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Import
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr.String(),
				},
				// Step 4: Rotate secret after import
				{
					Config: func() string {
						u := config.WithFirstBlock(
							testconfig.Descend("version_spec", "implementation", "gitlab_pipeline", "pipeline_trigger_token")(
								testconfig.Descend("secret_value")(testconfig.SetString("updated-plaintext-secret")),
								testconfig.Descend("secret_version")(testconfig.SetString("v1")),
							))
						return u.String()
					}(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
			},
		})
	})

	// Regression test for issue #131: releasing a version and then flipping it back to draft
	// together with a version_spec implementation change must NOT alter the already-released
	// version. The backend previously shared the implementation object across versions, so editing
	// the new draft retroactively mutated the released version, making its content_hash change
	// during apply ("Provider produced inconsistent result after apply"). Uses the Terraform
	// implementation because its implementation carries mutable fields (e.g. pre_run_script);
	// the manual implementation could not surface this.
	t.Run("06_release_redraft_implementation_change", func(t *testing.T) {
		config, addr := testconfig.BBDTerraform(t)

		// The released version's content_hash must be identical before and after the new draft
		// is created from it.
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		redraftWithImplChange := config.WithFirstBlock(
			testconfig.Descend("version_spec", "implementation", "terraform", "pre_run_script")(
				testconfig.SetString(`echo "changed for the second version"`),
			),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
					},
				},
				// Step 3: Flip draft false->true AND change the implementation -> new draft v2.
				// The released v1 must stay immutable (same content_hash, no inconsistent result).
				{
					Config: redraftWithImplChange.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
					},
				},
			},
		})
	})

	// Regression test for issues #131 and #176: manual building blocks derive their outputs from inputs
	// on the backend (SINGLE_SELECT/STATIC inputs auto-generate outputs). The config omits version_spec.outputs
	// entirely (it is computed). Releasing and then re-drafting together with an input change must reconcile
	// the derived outputs without "Provider produced inconsistent result after apply", and must not change
	// the already-released version.
	t.Run("07_manual_computed_outputs", func(t *testing.T) {
		config, addr := testconfig.BBDManual(t)
		withInputs := func(inputs string) testconfig.Config {
			return config.WithFirstBlock(testconfig.Descend("version_spec", "inputs")(testconfig.SetRawExpr("%s", inputs)))
		}

		// approval (BOOLEAN) mirrors to a BOOLEAN output; region (SINGLE_SELECT) mirrors to a STRING output.
		base := withInputs(`{
      approval = { display_name = "Approval", type = "BOOLEAN", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT" }
      region   = { display_name = "Region", type = "SINGLE_SELECT", assignment_type = "USER_INPUT", selectable_values = ["eu", "us"] }
    }`)
		redraft := withInputs(`{
      approval = { display_name = "Approval", type = "BOOLEAN", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT" }
      region   = { display_name = "Region", type = "SINGLE_SELECT", assignment_type = "USER_INPUT", selectable_values = ["eu", "us"] }
      ticket   = { display_name = "Ticket", type = "STRING", assignment_type = "STATIC", argument = jsonencode("T-1") }
    }`)

		// Derived outputs inherit the backend's positional display_order (input order, 0-based): the manual
		// output derivation assigns position = input index, see meshfed ManualDefinitionVersionService.
		outputCheck := func(displayName, ioType string, displayOrder int64) knownvalue.Check {
			return xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact(displayName),
				"type":            knownvalue.StringExact(ioType),
				"assignment_type": knownvalue.StringExact("NONE"),
				"display_order":   knownvalue.Int64Exact(displayOrder),
			})
		}
		baseOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"approval": outputCheck("Approval", "BOOLEAN", 0),
			"region":   outputCheck("Region", "STRING", 1),
		})
		redraftOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"approval": outputCheck("Approval", "BOOLEAN", 0),
			"region":   outputCheck("Region", "STRING", 1),
			"ticket":   outputCheck("Ticket", "STRING", 2),
		})

		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with outputs omitted -> outputs computed from inputs
				{
					Config: base.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config: releaseBBDVersion(t, base).String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
					},
				},
				// Step 3: Re-draft (draft false->true) AND add an input -> new draft v2 with reconciled outputs.
				{
					Config: redraft.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_spec").AtMapKey("outputs"), redraftOutputs),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
					},
				},
			},
		})
	})

	// Regression test for issue #196: rotating a sensitive input's secret on a released (immutable)
	// version previously failed with an opaque "Failed to determine content hash ... [plaintext]"
	// error, because the planned DTO carries the rotated secret's plaintext and the content hash
	// disallows plaintext keys. Released versions are immutable, so the rotation must be rejected with
	// a clear, actionable error instead.
	t.Run("08_release_secret_rotation_rejected", func(t *testing.T) {
		config, _ := testconfig.BBDTerraform(t)
		sensitiveInputs := func(version, value string) testconfig.Config {
			return config.WithFirstBlock(testconfig.Descend("version_spec", "inputs")(testconfig.SetRawExpr("%s", fmt.Sprintf(`{
      CONNECTOR_SECRET = {
        display_name    = "Connector Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value   = %q
            secret_version = %q
          }
        }
      }
    }`, value, version))))
		}
		base := sensitiveInputs("v1", "plaintext-secret-v1")
		rotated := sensitiveInputs("v2", "plaintext-secret-v2")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a sensitive input
				{Config: base.String()},
				// Step 2: Release v1 (now immutable)
				{Config: releaseBBDVersion(t, base).String()},
				// Step 3: Rotate the secret on the released version -> clear error, not an opaque hash failure
				{
					Config:      releaseBBDVersion(t, rotated).String(),
					ExpectError: regexp.MustCompile("Updating a version_spec in non-draft state is not allowed"),
				},
			},
		})
	})

	// Regression test for issue #196: the released-version secret-rotation guard must key off an actual
	// secret change, not the presence of a JSON key literally named "plaintext". An input named "plaintext"
	// (or any STATIC argument whose JSON carries a "plaintext" key) is user data, not a secret. Mutating a
	// non-version_spec field (the description) on a released version that contains such an input must NOT be
	// rejected as a secret rotation.
	t.Run("09_released_plaintext_named_input_is_not_a_secret", func(t *testing.T) {
		config, addr := testconfig.BBDTerraform(t)
		withPlaintextInput := func(c testconfig.Config) testconfig.Config {
			return c.WithFirstBlock(testconfig.Descend("version_spec", "inputs")(testconfig.SetRawExpr("%s", `{
      plaintext = { display_name = "Plaintext", type = "STRING", assignment_type = "STATIC", argument = jsonencode("hello") }
    }`)))
		}
		base := withPlaintextInput(config)
		released := releaseBBDVersion(t, base)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a non-sensitive input named "plaintext"
				{Config: base.String()},
				// Step 2: Release v1 (now immutable)
				{Config: released.String()},
				// Step 3: Change only the description on the released version. version_spec is unchanged and
				// carries no secret, so this must succeed - the old "plaintext" key match wrongly rejected it.
				{
					Config: updateBBDDescription(t, released, "updated description, version_spec untouched").String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("spec").AtMapKey("description"), knownvalue.StringExact("updated description, version_spec untouched")),
					},
				},
			},
		})
	})

	// Positive-path companion to issue #196: rotating a secret on a released version is rejected (subtest 08),
	// but the documented workaround - flip version_spec.draft false->true AND rotate the secret in the same
	// step - must succeed, creating a new draft version that carries the rotated secret while the released
	// version stays immutable. Guards the ModifyPlan rejection from over-triggering on the draft-flip path.
	t.Run("10_redraft_with_secret_rotation_allowed", func(t *testing.T) {
		config, addr := testconfig.BBDTerraform(t)
		sensitiveInputs := func(version, value string) testconfig.Config {
			return config.WithFirstBlock(testconfig.Descend("version_spec", "inputs")(testconfig.SetRawExpr("%s", fmt.Sprintf(`{
      CONNECTOR_SECRET = {
        display_name    = "Connector Secret"
        type            = "STRING"
        assignment_type = "STATIC"
        sensitive = {
          argument = {
            secret_value   = %q
            secret_version = %q
          }
        }
      }
    }`, value, version))))
		}
		base := sensitiveInputs("v1", "plaintext-secret-v1")
		// base defaults to draft=true; after releasing v1 this same config (draft=true) flips back to draft
		// while also rotating the secret to v2 -> a new draft version v2.
		redraftRotated := sensitiveInputs("v2", "plaintext-secret-v2")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a sensitive input
				{Config: base.String()},
				// Step 2: Release v1 (now immutable)
				{Config: releaseBBDVersion(t, base).String()},
				// Step 3: Flip back to draft AND rotate the secret in one step -> new draft v2, v1 untouched
				{
					Config: redraftRotated.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
					},
				},
			},
		})
	})

	// Regression test (companion to #131) for the integration reference of CI implementations: github
	// (like gitlab/azure) carries an integration_ref besides the implementation. Releasing a github
	// version and then re-drafting it (draft false->true) while switching to a different integration must
	// create a new draft v2 pointing at the new integration, while the already-released v1 keeps its
	// original integration and stays immutable (stable content_hash, no "inconsistent result after
	// apply"). The backend keeps this safe by deep-copying the implementation binding when deriving the
	// draft, so changing the draft's integration reference does not retroactively repoint the released one.
	t.Run("11_github_release_redraft_integration_change", func(t *testing.T) {
		config, addr, integrationBAddr := testconfig.BBDGithubTwoIntegrations(t)

		// The released version's content_hash (which includes integration_ref) must stay identical
		// across the re-draft, proving the released version was not retroactively repointed.
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		// Sanity check that the test actually switches the integration: the draft's integration_ref uuid
		// must differ between draft v1 (integration A) and draft v2 (integration B).
		integrationSwitched := statecheck.CompareValue(compare.ValuesDiffer())
		integrationUuidPath := tfjsonpath.New("version_spec").AtMapKey("implementation").
			AtMapKey("github_workflows").AtMapKey("integration_ref").AtMapKey("uuid")

		// base defaults to draft=true with integration A; this same config (draft=true) flips back to
		// draft while switching to integration B -> a new draft version v2.
		redraftWithIntegrationChange := config.WithFirstBlock(
			testconfig.Descend("version_spec", "implementation", "github_workflows", "integration_ref")(
				testconfig.SetAddr(integrationBAddr, "ref"),
			),
		)

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with integration A
				{
					Config: config.String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						integrationSwitched.AddStateValue(addr.String(), integrationUuidPath),
					},
				},
				// Step 2: Release v1 (now immutable)
				{
					Config: releaseBBDVersion(t, config).String(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
					},
				},
				// Step 3: Flip draft false->true AND switch integration A->B -> new draft v2.
				// Released v1 must keep integration A and stay immutable (same content_hash).
				{
					Config: redraftWithIntegrationChange.String(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr.String(), plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr.String(), tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
						releasedHashStable.AddStateValue(addr.String(), releasedHashPath),
						integrationSwitched.AddStateValue(addr.String(), integrationUuidPath),
					},
				},
			},
		})
	})
}

// checkBBDMetadataFull checks metadata for the 01_terraform example (tags with 2 entries).
func checkBBDMetadataFull() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"tags":               knownvalue.MapSizeExact(2),
	})
}

// checkBBDMetadataMinimal checks metadata for examples without tags.
func checkBBDMetadataMinimal() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": xknownvalue.NotEmptyString(),
		"tags":               knownvalue.MapSizeExact(0),
	})
}

// checkBBDSpecFull checks spec for the 01_terraform example (all optional attributes set).
func checkBBDSpecFull(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "Example Building Block") {
				return fmt.Errorf("expected %s to start with %s", v, "Example Building Block")
			}
			return nil
		}),
		"symbol": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "data:image/png;base64,") {
				return fmt.Errorf("value does not start with %s", "data:image/png;base64,")
			}
			return nil
		}),
		"description":       knownvalue.StringExact(expectedDescription),
		"readme":            xknownvalue.NotEmptyString(),
		"support_url":       knownvalue.StringExact("https://support.example.com/building-blocks"),
		"documentation_url": knownvalue.StringExact("https://docs.example.com/building-blocks"),
		"target_type":       knownvalue.StringExact("TENANT_LEVEL"),
		"supported_platforms": knownvalue.SetExact([]knownvalue.Check{
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AZURE"),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AWS"),
			}),
		}),
		"run_transparency":          knownvalue.Bool(true),
		"use_in_landing_zones_only": knownvalue.Bool(true),
		"notification_subscribers": knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("email:ops@example.com"),
		}),
	})
}

// checkBBDSpecMinimal checks spec for examples with only required attributes (workspace-level, no extras).
func checkBBDSpecMinimal(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "Example Building Block") {
				return fmt.Errorf("expected %s to start with %s", v, "Example Building Block")
			}
			return nil
		}),
		"symbol":                    xknownvalue.NotEmptyString(),
		"description":               knownvalue.StringExact(expectedDescription),
		"readme":                    knownvalue.Null(),
		"support_url":               knownvalue.Null(),
		"documentation_url":         knownvalue.Null(),
		"target_type":               knownvalue.StringExact("WORKSPACE_LEVEL"),
		"supported_platforms":       knownvalue.Null(),
		"run_transparency":          knownvalue.Bool(false),
		"use_in_landing_zones_only": knownvalue.Bool(false),
		"notification_subscribers":  knownvalue.SetSizeExact(0),
	})
}

func checkBuildingBlockVersionSpec(exampleSuffix string, expectedState enum.Entry[client.MeshBuildingBlockDefinitionVersionState], expectedNumber int64) knownvalue.Check {
	checkInputs, checkImplementation, checkOutputs := checksForImplementation(exampleSuffix)
	expectedDeletionMode := "DELETE"
	if exampleSuffix == "02_github_workflows" {
		expectedDeletionMode = "PURGE"
	}
	expected := map[string]knownvalue.Check{
		"state":                      knownvalue.StringExact(expectedState.String()),
		"version_number":             knownvalue.Int64Exact(expectedNumber),
		"draft":                      knownvalue.Bool(expectedState == client.MeshBuildingBlockDefinitionVersionStateDraft),
		"only_apply_once_per_tenant": knownvalue.Bool(exampleSuffix == "01_terraform"),
		"deletion_mode":              knownvalue.StringExact(expectedDeletionMode),
		"runner_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"kind": knownvalue.StringExact("meshBuildingBlockRunner"),
			"uuid": knownvalue.StringExact(SharedBuildingBlockRunnerUuid),
		}),
		"dependency_refs": knownvalue.SetSizeExact(0),
		"inputs":          checkInputs,
		"implementation":  checkImplementation,
		"outputs":         checkOutputs,
		"permissions":     knownvalue.SetSizeExact(0),
	}

	if exampleSuffix == "01_terraform" {
		expected["dependency_refs"] = knownvalue.ListExact([]knownvalue.Check{
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshBuildingBlockDefinition"),
				"uuid": xknownvalue.NotEmptyString(),
			}),
		})
		expected["permissions"] = knownvalue.SetExact([]knownvalue.Check{
			knownvalue.StringExact("TENANT_SAVE"),
			knownvalue.StringExact("TENANT_LIST"),
		})
	}
	return xknownvalue.MapExact(expected)
}

func checksForImplementation(exampleSuffix string) (checkInputs, checkImplementation, checkOutputs knownvalue.Check) {
	switch exampleSuffix {
	case "01_terraform":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"environment": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Environment"),
					"type":                   knownvalue.StringExact("SINGLE_SELECT"),
					"assignment_type":        knownvalue.StringExact("USER_INPUT"),
					"is_environment":         knownvalue.Bool(false),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("The target environment"),
					"selectable_values": knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("dev"),
						knownvalue.StringExact("prod"),
						knownvalue.StringExact("staging"),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(1),
				}),
				"resource_name": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Resource Name"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(true),
					"description":                    knownvalue.StringExact("Name of the resource to create"),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.StringExact(`"some-resource-name"`),
					"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
					"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(2),
				}),
				"SOMETHING_VERY_SECRET": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Top Secret"),
					"type":                   knownvalue.StringExact("STRING"),
					"assignment_type":        knownvalue.StringExact("STATIC"),
					"is_environment":         knownvalue.Bool(true),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("Really secret"),
					"sensitive": xknownvalue.MapExact(map[string]knownvalue.Check{
						"argument": xknownvalue.MapExact(map[string]knownvalue.Check{
							"secret_value":   knownvalue.Null(),
							"secret_hash":    xknownvalue.NotEmptyString(),
							"secret_version": xknownvalue.NotEmptyString(),
						}),
						"default_value": knownvalue.Null(),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
				"some-file.yaml": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Some input file"),
					"type":                           knownvalue.StringExact("FILE"),
					"assignment_type":                knownvalue.StringExact("STATIC"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"argument":                       xknownvalue.NotEmptyString(),
					"default_value":                  knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             checkTerraformImplementation(),
			}), xknownvalue.MapExact(map[string]knownvalue.Check{
				"some_output_flag": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("If true, it really worked"),
					"type":            knownvalue.StringExact("BOOLEAN"),
					"assignment_type": knownvalue.StringExact("NONE"),
					"display_order":   knownvalue.Int64Exact(1),
				}),
				"summary": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Summary of work"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("SUMMARY"),
					"display_order":   knownvalue.Int64Exact(2),
				}),
			})
	case "02_github_workflows":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Workflow Reference"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      checkGithubWorkflowsImplementation(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"workflow_run_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Workflow Run URL"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
					"display_order":   knownvalue.Int64Exact(0),
				}),
			})
	case "03_manual":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"approval_required": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Approval Required"),
					"type":                           knownvalue.StringExact("BOOLEAN"),
					"assignment_type":                knownvalue.StringExact("PLATFORM_OPERATOR_MANUAL_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                checkManualImplementation(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"approval_required": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Approval Required"),
					"type":            knownvalue.StringExact("BOOLEAN"),
					"assignment_type": knownvalue.StringExact("NONE"),
					"display_order":   knownvalue.Int64Exact(0),
				}),
			})
	case "04_azure_devops_pipeline":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_config": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Pipeline Configuration"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       knownvalue.Null(),
				"azure_devops_pipeline": checkAzureDevopsPipelineImplementation(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_run_id": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Pipeline Run ID"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("NONE"),
					"display_order":   knownvalue.Int64Exact(0),
				}),
			})
	case "05_gitlab_pipeline":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"deployment_env": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Deployment Environment"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"manual":                knownvalue.Null(),
				"github_workflows":      knownvalue.Null(),
				"gitlab_pipeline":       checkGitlabPipelineImplementation(),
				"azure_devops_pipeline": knownvalue.Null(),
				"terraform":             knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"pipeline_web_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Pipeline URL"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
					"display_order":   knownvalue.Int64Exact(0),
				}),
			})
	default:
		panic(fmt.Sprintf("unknown example suffix: %s", exampleSuffix))
	}
}

func checkTerraformImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"terraform_version":              knownvalue.StringExact("1.9.0"),
		"repository_url":                 knownvalue.StringExact("https://github.com/example/building-block.git"),
		"async":                          knownvalue.Bool(true),
		"repository_path":                knownvalue.StringExact("terraform/modules/example"),
		"ref_name":                       knownvalue.StringExact("v1.0.0"),
		"use_mesh_http_backend_fallback": knownvalue.Bool(true),
		"ssh_known_host": xknownvalue.MapExact(map[string]knownvalue.Check{
			"host":      knownvalue.StringExact("github.com"),
			"key_type":  knownvalue.StringExact("ssh-rsa"),
			"key_value": xknownvalue.NotEmptyString(),
		}),
		"ssh_private_key": xknownvalue.MapExact(map[string]knownvalue.Check{
			"secret_value":   knownvalue.Null(),
			"secret_hash":    xknownvalue.NotEmptyString(),
			"secret_version": xknownvalue.NotEmptyString(),
		}),
		"pre_run_script": knownvalue.StringExact(`echo "hello world"`),
	})
}

func checkManualImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{})
}

func checkGitlabPipelineImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"project_id": knownvalue.StringExact("12345678"),
		"ref_name":   knownvalue.StringExact("main"),
		"pipeline_trigger_token": xknownvalue.MapExact(map[string]knownvalue.Check{
			"secret_value":   knownvalue.Null(),
			"secret_hash":    xknownvalue.NotEmptyString(),
			"secret_version": xknownvalue.NotEmptyString(),
		}),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkGithubWorkflowsImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"repository":            knownvalue.StringExact("example/building-block"),
		"branch":                knownvalue.StringExact("main"),
		"apply_workflow":        knownvalue.StringExact("apply.yml"),
		"destroy_workflow":      knownvalue.Null(),
		"async":                 knownvalue.Bool(true),
		"omit_run_object_input": knownvalue.Bool(true),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func checkAzureDevopsPipelineImplementation() knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"project":     knownvalue.StringExact("MyProject"),
		"pipeline_id": knownvalue.StringExact("42"),
		"ref_name":    knownvalue.StringExact("refs/heads/main"),
		"async":       knownvalue.Bool(false),
		"integration_ref": xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid": xknownvalue.NotEmptyString(),
			"kind": knownvalue.StringExact("meshIntegration"),
		}),
	})
}

func TestAccBuildingBlockDefinitionSymbolValidation(t *testing.T) {
	// Symbol validation is client-side only; success cases need a real workspace in acceptance mode.
	if !IsMockClientTest() {
		t.Skip("symbol validation is tested with mock client only")
	}

	t.Parallel()

	// symbolConfig wraps a symbol value into a minimal valid BBD config.
	symbolConfig := func(symbol string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec = {
    display_name = "Test"
    description  = "Test"
    symbol       = %q
  }
  version_spec = {
    draft = true
    implementation = { manual = {} }
  }
}`, symbol)
	}

	tests := []struct {
		name        string
		symbol      string
		expectError *regexp.Regexp
	}{
		{
			name:   "https URL",
			symbol: "https://example.com/icon.png",
		},
		{
			name:   "http URL",
			symbol: "http://example.com/icon.png",
		},
		{
			name:        "plain string is rejected",
			symbol:      "not-a-url-or-data-uri",
			expectError: regexp.MustCompile(`Invalid Symbol Format`),
		},
		{
			name:        "disallowed image type is rejected",
			symbol:      "data:image/bmp;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 50))),
			expectError: regexp.MustCompile(`Invalid Symbol Format`),
		},
		{
			name:        "invalid base64 is rejected",
			symbol:      "data:image/png;base64,!!!not-valid-base64!!!",
			expectError: regexp.MustCompile(`Invalid Base64 in Symbol Data URI`),
		},
		{
			name:   "data URI decoded size exactly at 100 KiB limit",
			symbol: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024))),
		},
		{
			name:        "data URI decoded size exceeds 100 KiB limit",
			symbol:      "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024+1))),
			expectError: regexp.MustCompile(`Symbol Image Too Large`),
		},
		{
			name:   "raw (no-padding) base64",
			symbol: "data:image/jpeg;base64," + base64.RawStdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024))),
		},
		{
			name:   "svg+xml image type",
			symbol: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("y", 50))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{
				Config: symbolConfig(tt.symbol),
			}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionManualOutputsValidation(t *testing.T) {
	// Output configuration rules for manual building blocks are validated client-side only.
	if !IsMockClientTest() {
		t.Skip("manual outputs validation is tested with mock client only")
	}

	t.Parallel()

	manualConfig := func(outputs string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft = true
    inputs = {
      tenant = { display_name = "Tenant", type = "STRING", assignment_type = "STATIC", argument = jsonencode("t") }
    }
    implementation = { manual = {} }
    %s
  }
}`, outputs)
	}

	type testCase struct {
		name        string
		outputs     string
		expectError *regexp.Regexp
	}
	tests := []testCase{
		{
			name:        "NONE output rejected for manual",
			outputs:     `outputs = { tenant = { display_name = "Tenant", type = "STRING", assignment_type = "NONE" } }`,
			expectError: regexp.MustCompile(`must have a special assignment_type`),
		},
		{
			// Omitting assignment_type defaults it to NONE, which the backend ignores; it must be
			// rejected the same as an explicit NONE rather than silently accepted.
			name:        "omitted assignment_type rejected for manual",
			outputs:     `outputs = { tenant = { display_name = "Tenant", type = "STRING" } }`,
			expectError: regexp.MustCompile(`must have a special assignment_type`),
		},
		{
			// Regression test for issue #240: an explicit empty map is not the same as omitting outputs.
			// The backend derives one output per input, so `outputs = {}` fails at apply with "provider
			// produced inconsistent result after apply". Reject it at validate time and tell users to omit it.
			name:        "empty outputs map rejected for manual",
			outputs:     `outputs = {}`,
			expectError: regexp.MustCompile(`must be omitted, not an empty map`),
		},
	}

	// Every non-NONE assignment type is accepted on a manual output; loop the enum so a new entry is
	// covered without editing this test.
	for _, assignmentType := range nonNoneOutputAssignmentTypes {
		tests = append(tests, testCase{
			name: fmt.Sprintf("%s output allowed for manual", assignmentType),
			outputs: fmt.Sprintf(
				`outputs = { tenant = { display_name = "Tenant", type = "STRING", assignment_type = %q } }`,
				assignmentType,
			),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: manualConfig(tt.outputs)}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}
