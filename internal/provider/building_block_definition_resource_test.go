package provider

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/client/types/enum"

	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// ascendingVersionsExact checks the versions numbered 1, 2, … in that order.
func ascendingVersionsExact(versions ...knownvalue.Check) knownvalue.Check {
	byKey := make(map[string]knownvalue.Check, len(versions))
	for position, version := range versions {
		byKey[versionKey(int64(position+1))] = version
	}
	return xknownvalue.MapExact(byKey)
}

func TestAccBuildingBlockDefinition(t *testing.T) {
	t.Parallel()

	var (
		versionStateDraft    = client.MeshBuildingBlockDefinitionVersionStateDraft
		versionStateReleased = client.MeshBuildingBlockDefinitionVersionStateReleased
	)

	// The hosted runner registers its identity through the run-controller, which the CI meshStack does not
	// run, so the block is null there and set elsewhere.
	expectedVersion := func(number int64, state enum.Entry[client.MeshBuildingBlockDefinitionVersionState]) knownvalue.Check {
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"uuid":                         xknownvalue.NotEmptyString(),
			"number":                       knownvalue.Int64Exact(number),
			"state":                        knownvalue.StringExact(state.String()),
			"content_hash":                 xknownvalue.NotEmptyString(),
			"kind":                         knownvalue.StringExact(client.MeshObjectKind.BuildingBlockDefinitionVersion),
			"workload_identity_federation": xknownvalue.Any(),
		})
	}

	const bbdDescription = "An example building block definition"

	t.Run("01_terraform", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_01_terraform"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/01_terraform", 1, vars),
				},
				{
					Config: hclStep(t, "resources/building_block_definition/01_terraform", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataFull(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecFull(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "01_terraform", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_definition/01_terraform", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr,
				},
			},
		})
	})

	t.Run("02_github_workflows", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_02_github_workflows"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/02_github_workflows", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "02_github_workflows", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr,
				},
			},
		})
	})

	t.Run("03_manual", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_03_manual"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create
				{
					Config: hclStep(t, "resources/building_block_definition/03_manual", 1, vars),

					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "03_manual", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Update spec (description change, no new version)
				{
					Config: hclStep(t, "resources/building_block_definition/03_manual", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal("An updated building block definition")),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Release (draft=false)
				{
					Config: hclStep(t, "resources/building_block_definition/03_manual", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "03_manual", versionStateReleased, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateReleased))),
					},
				},
				// Step 4: New draft (draft=true again, description changed)
				{
					Config: hclStep(t, "resources/building_block_definition/03_manual", 4, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "03_manual", versionStateDraft, 2)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						)),
					},
				},
				// Step 5: Release the new draft (draft=false)
				{
					Config: hclStep(t, "resources/building_block_definition/03_manual", 5, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "03_manual", versionStateReleased, 2)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateReleased),
						)),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr,
				},
			},
		})
	})

	t.Run("04_azure_devops_pipeline", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_04_azure_devops_pipeline"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/04_azure_devops_pipeline", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "04_azure_devops_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: addr,
				},
			},
		})
	})

	t.Run("05_gitlab_pipeline", func(t *testing.T) {
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_05_gitlab_pipeline"
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Plan-only (ensure tf plan works before apply)
				{
					Config:             hclStep(t, "resources/building_block_definition/05_gitlab_pipeline", 1, vars),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				// Step 2: Create
				{
					Config: hclStep(t, "resources/building_block_definition/05_gitlab_pipeline", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec(suffix, "05_gitlab_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
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
					ResourceName: addr,
				},
				// Step 4: Rotate secret after import
				{
					Config: hclStep(t, "resources/building_block_definition/05_gitlab_pipeline", 4, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal(suffix)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(expectedVersion(1, versionStateDraft))),
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
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_01_terraform"

		// The released version's content_hash must be identical before and after the new draft
		// is created from it.
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1
				{
					Config: hclStep(t, "resources/building_block_definition/06_release_redraft_implementation_change", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config: hclStep(t, "resources/building_block_definition/06_release_redraft_implementation_change", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
				// Step 3: Flip draft false->true AND change the implementation -> new draft v2.
				// The released v1 must stay immutable (same content_hash, no inconsistent result).
				{
					Config: hclStep(t, "resources/building_block_definition/06_release_redraft_implementation_change", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						)),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
			},
		})
	})

	// Regression test for issues #131 and #176: manual building blocks derive their outputs from inputs
	// on the backend (SINGLE_SELECT/STATIC inputs auto-generate outputs). This test declares no output
	// overrides (version_spec.outputs = {}), so all outputs are derived and pruned. Releasing and then
	// re-drafting together with an input change must reconcile the derived outputs without "Provider produced
	// inconsistent result after apply", and must not change the already-released version.
	t.Run("07_manual_computed_outputs", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_03_manual"

		// Outputs are omitted, so every derived output is a non-override (assignment NONE, display_name = the
		// input's) and prunes away: the tracked subset is the empty map, across the input change too.
		baseOutputs := knownvalue.MapSizeExact(0)
		redraftOutputs := knownvalue.MapSizeExact(0)

		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with outputs omitted -> outputs computed from inputs
				{
					Config: hclStep(t, "resources/building_block_definition/07_manual_computed_outputs", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config: hclStep(t, "resources/building_block_definition/07_manual_computed_outputs", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
				// Step 3: Re-draft (draft false->true) AND add an input -> new draft v2 with reconciled outputs.
				{
					Config: hclStep(t, "resources/building_block_definition/07_manual_computed_outputs", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), redraftOutputs),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
			},
		})
	})

	// Regression test for declaring outputs on a manual building block as a SPARSE OVERRIDE (issues #131,
	// #176, #240). The backend derives one output per input and returns the full set; the provider tracks only
	// the user's overrides (assignment_type != NONE, or a display_name different from the input's) and prunes
	// the rest. Declaring a subset must create, release and re-draft without "inconsistent result after apply"
	// or a content_hash flip, and state must hold only the tracked keys. A declared output must not set type
	// (always derived) - covered by the validation subtest.
	t.Run("12_manual_declared_outputs", func(t *testing.T) {
		// Three inputs, two overrides: rename approval (display_name differs from the input's) and mark region
		// SUMMARY. ticket is not overridden, so its derived output (NONE, display_name = "Ticket") is pruned,
		// proving the subset semantics. No output sets type (always derived) or display_order (the derived
		// positional order flows: inputs sorted by (display_order, key) -> approval=0, region=1, ticket=2).
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_03_manual"

		expectedOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"approval": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Approval Output"),
				"type":            knownvalue.StringExact("BOOLEAN"),
				"assignment_type": knownvalue.StringExact("NONE"),
				"display_order":   knownvalue.Int64Exact(0),
			}),
			"region": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Region"),
				"type":            knownvalue.StringExact("STRING"),
				"assignment_type": knownvalue.StringExact("SUMMARY"),
				"display_order":   knownvalue.Int64Exact(1),
			}),
		})
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a full declared output set (bug 1: create reconciliation).
				{
					Config: hclStep(t, "resources/building_block_definition/12_manual_declared_outputs", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1 (bug 2: content_hash must not flip during apply).
				{
					Config: hclStep(t, "resources/building_block_definition/12_manual_declared_outputs", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
				// Step 3: Re-draft (draft false->true) with no change -> new draft v2 (bug 2: create new
				// version content_hash must not flip); released v1 stays immutable.
				{
					Config: hclStep(t, "resources/building_block_definition/12_manual_declared_outputs", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
			},
		})
	})

	// Sparse override lifecycle on a draft: flipping, removing and re-adding overrides updates in place and
	// re-prunes. Dropping the approval override - which the stateful backend would otherwise preserve -
	// exercises the full-set send that resets it, and region is renamed. Re-adding approval afterwards covers
	// a new override key on an existing resource, whose backend-derived display_name/type/display_order must
	// be planned unknown rather than null (no prior state for that key).
	t.Run("13_manual_output_override_update", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_03_manual"

		initialOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"approval": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Approval"),
				"type":            knownvalue.StringExact("BOOLEAN"),
				"assignment_type": knownvalue.StringExact("SUMMARY"),
				"display_order":   knownvalue.Int64Exact(0),
			}),
			"region": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Region Output"),
				"type":            knownvalue.StringExact("STRING"),
				"assignment_type": knownvalue.StringExact("NONE"),
				"display_order":   knownvalue.Int64Exact(1),
			}),
		})
		updatedOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"region": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Region Renamed"),
				"type":            knownvalue.StringExact("STRING"),
				"assignment_type": knownvalue.StringExact("NONE"),
				"display_order":   knownvalue.Int64Exact(1),
			}),
		})
		readdedOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"approval": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Approval"),
				"type":            knownvalue.StringExact("BOOLEAN"),
				"assignment_type": knownvalue.StringExact("SUMMARY"),
				"display_order":   knownvalue.Int64Exact(0),
			}),
			"region": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Region Renamed"),
				"type":            knownvalue.StringExact("STRING"),
				"assignment_type": knownvalue.StringExact("NONE"),
				"display_order":   knownvalue.Int64Exact(1),
			}),
		})

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/13_manual_output_override_update", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), initialOutputs),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_definition/13_manual_output_override_update", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), updatedOutputs),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_definition/13_manual_output_override_update", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), readdedOutputs),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
			},
		})
	})

	// Import must materialize exactly the diff-rule subset: with no prior state, the read-back prunes the
	// backend's full one-per-input set to the tracked overrides, so the imported state equals the applied one.
	t.Run("14_manual_output_import", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_03_manual"
		var resourceUuid string

		// Only region is a tracked override (assignment != NONE); approval is derived (NONE, display_name equal
		// to the input's) and pruned. The imported state must reproduce exactly this subset.
		expectedOutputs := knownvalue.MapExact(map[string]knownvalue.Check{
			"region": xknownvalue.MapExact(map[string]knownvalue.Check{
				"display_name":    knownvalue.StringExact("Region"),
				"type":            knownvalue.StringExact("STRING"),
				"assignment_type": knownvalue.StringExact("SUMMARY"),
				"display_order":   knownvalue.Int64Exact(1),
			}),
		})

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/14_manual_output_import", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						xknownvalue.Ref(addr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					// Command-style import (not a plannable import block, which does not support ImportStateVerify).
					ImportState:                          true,
					ImportStateIdFunc:                    func(_ *terraform.State) (string, error) { return resourceUuid, nil },
					ResourceName:                         addr,
					ImportStateVerify:                    true,
					ImportStateVerifyIdentifierAttribute: "metadata.uuid",
				},
			},
		})
	})

	// A template can be set, changed, emptied, and dropped again. Both an empty string and a missing
	// attribute mean the same thing to meshStack - name the building block after the definition - but
	// Terraform tells them apart, so each has to survive a refresh without leaving a diff behind.
	t.Run("15_display_name_template_lifecycle", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_03_manual"
		nameTemplatePath := tfjsonpath.New("spec").AtMapKey("display_name_template")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/15_display_name_template_lifecycle", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, nameTemplatePath, knownvalue.StringExact("Block for {{ region }}")),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_definition/15_display_name_template_lifecycle", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, nameTemplatePath, knownvalue.StringExact("Block for {{ region }} v2")),
					},
				},
				{
					Config: hclStep(t, "resources/building_block_definition/15_display_name_template_lifecycle", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, nameTemplatePath, knownvalue.StringExact("")),
					},
				},
				{
					// Dropping the attribute removes the template, and the refresh has to agree: a state that
					// still held the old value would leave the same diff on every later plan.
					Config: hclStep(t, "resources/building_block_definition/15_display_name_template_lifecycle", 4, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, nameTemplatePath, knownvalue.Null()),
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
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a sensitive input
				{Config: hclStep(t, "resources/building_block_definition/08_release_secret_rotation_rejected", 1, vars)},
				// Step 2: Release v1 (now immutable)
				{Config: hclStep(t, "resources/building_block_definition/08_release_secret_rotation_rejected", 2, vars)},
				// Step 3: Rotate the secret on the released version -> clear error, not an opaque hash failure
				{
					Config:      hclStep(t, "resources/building_block_definition/08_release_secret_rotation_rejected", 3, vars),
					ExpectError: expectErrorWords("version_spec", "non-draft", "allowed"),
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
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_01_terraform"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a non-sensitive input named "plaintext"
				{Config: hclStep(t, "resources/building_block_definition/09_released_plaintext_named_input_is_not_a_secret", 1, vars)},
				// Step 2: Release v1 (now immutable)
				{Config: hclStep(t, "resources/building_block_definition/09_released_plaintext_named_input_is_not_a_secret", 2, vars)},
				// Step 3: Change only the description on the released version. version_spec is unchanged and
				// carries no secret, so this must succeed - the old "plaintext" key match wrongly rejected it.
				{
					Config: hclStep(t, "resources/building_block_definition/09_released_plaintext_named_input_is_not_a_secret", 3, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("description"), knownvalue.StringExact("updated description, version_spec untouched")),
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
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_01_terraform"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a sensitive input
				{Config: hclStep(t, "resources/building_block_definition/10_redraft_with_secret_rotation_allowed", 1, vars)},
				// Step 2: Release v1 (now immutable)
				{Config: hclStep(t, "resources/building_block_definition/10_redraft_with_secret_rotation_allowed", 2, vars)},
				// Step 3: Flip back to draft AND rotate the secret in one step -> new draft v2, v1 untouched
				{
					Config: hclStep(t, "resources/building_block_definition/10_redraft_with_secret_rotation_allowed", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						)),
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
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_02_github_workflows"

		// The released version's content_hash (which includes integration_ref) must stay identical
		// across the re-draft, proving the released version was not retroactively repointed.
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		// Sanity check that the test actually switches the integration: the draft's integration_ref uuid
		// must differ between draft v1 (integration A) and draft v2 (integration B).
		integrationSwitched := statecheck.CompareValue(compare.ValuesDiffer())
		integrationUuidPath := tfjsonpath.New("version_spec").AtMapKey("implementation").
			AtMapKey("github_workflows").AtMapKey("integration_ref").AtMapKey("uuid")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with integration A
				{
					Config: hclStep(t, "resources/building_block_definition/11_github_release_redraft_integration_change", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						integrationSwitched.AddStateValue(addr, integrationUuidPath),
					},
				},
				// Step 2: Release v1 (now immutable)
				{
					Config: hclStep(t, "resources/building_block_definition/11_github_release_redraft_integration_change", 2, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
					},
				},
				// Step 3: Flip draft false->true AND switch integration A->B -> new draft v2.
				// Released v1 must keep integration A and stay immutable (same content_hash).
				{
					Config: hclStep(t, "resources/building_block_definition/11_github_release_redraft_integration_change", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("versions"), ascendingVersionsExact(
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						)),
						releasedHashStable.AddStateValue(addr, releasedHashPath),
						integrationSwitched.AddStateValue(addr, integrationUuidPath),
					},
				},
			},
		})
	})

	t.Run("12_restricted_default_tag", func(t *testing.T) {
		// Backend-materialized default: the mock has no tag-restriction logic, so it can't reproduce
		// BuildingBlockDefinitionTagService injecting a restricted tag's default on create. See the
		// lock-step policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		// One tag the BBD declares (needs a real definition or the backend rejects it) ...
		// ... and a restricted tag with a default the BBD does not declare: the backend injects it on
		// create, so the fix must keep it out of the managed tags instead of crashing / drifting.
		// depends_on forces the tag definitions to be created before the BBD (so the restricted
		// default is actually injected) and destroyed after it. The latter avoids a backend bug:
		// deleting a restricted BBD tag definition while any BBD still exists hits a
		// BuildingBlockDefinitionTag foreign-key violation (500) during the server-side tag recompute.
		suffix := acctest.RandString(8)
		vars := hclStepVars{"suffix": suffix}
		addr := "meshstack_building_block_definition.example_03_manual"

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config: hclStep(t, "resources/building_block_definition/12_restricted_default_tag", 1, vars),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the declared tag remains; the injected restricted default is not tracked.
						statecheck.ExpectKnownValue(addr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							"test-key-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.BuildingBlockDefinition))
	})

	// spec.approval_policies / spec.schedule change together with the version's implementation type in
	// one apply. meshStack validates both policies against the implementation type of the definition's
	// latest version, and rejects an implementation change while the stored policies are incompatible
	// with the new type. Neither direction can be applied with a single definition write, so this asserts
	// the resource orders the writes: it passes through neutral policies, writes the version, then writes
	// the planned policies. The mock client echoes any spec back, so only an acceptance run against
	// meshStack rejects a wrong order.
	t.Run("16_policies_across_implementation_change", func(t *testing.T) {
		// github_workflows has no dry run, so it supports drift reconciliation only with automatic
		// approval, and no approval gate at all. terraform supports every policy.
		vars := hclStepVars{"suffix": acctest.RandString(8)}
		addr := "meshstack_building_block_definition.example_02_github_workflows"

		checkPolicies := func(approvalPolicies map[string]bool, mode enum.Entry[client.MeshBuildingBlockScheduleMode], automaticApproval bool) []statecheck.StateCheck {
			return []statecheck.StateCheck{
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("approval_policies"), checkBBDApprovalPolicies(approvalPolicies)),
				statecheck.ExpectKnownValue(addr, tfjsonpath.New("spec").AtMapKey("schedule"),
					checkBBDSchedule(mode, client.MeshBuildingBlockScheduleFrequencyDaily, automaticApproval)),
			}
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Create with a schedule enabled: the definition is created with neutral policies, and the
				// schedule follows once the initial version carries the github_workflows implementation.
				{
					Config: hclStep(t, "resources/building_block_definition/16_policies_across_implementation_change", 1, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: checkPolicies(nil, client.MeshBuildingBlockScheduleModeDriftReconciliation, true),
				},
				// Tighten across an implementation change: approval gates and drift detection need terraform,
				// so the policies can only be written after the version switched to it.
				{
					Config: hclStep(t, "resources/building_block_definition/16_policies_across_implementation_change", 2, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: checkPolicies(
						map[string]bool{"manual_triggers": true, "version_upgrade": true},
						client.MeshBuildingBlockScheduleModeDriftDetection, false),
				},
				// Relax across an implementation change: github_workflows supports neither the approval gates
				// nor drift detection, so they have to be gone before the version switches back to it.
				{
					Config: hclStep(t, "resources/building_block_definition/16_policies_across_implementation_change", 3, vars),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: checkPolicies(nil, client.MeshBuildingBlockScheduleModeDriftReconciliation, true),
				},
			},
		})
	})

	// A version_spec change on a released version must be rejected before the definition is written. The
	// last step passes the plan-time policy validators, because those read the implementation type the
	// configuration asks for, while meshStack validates the policies against the implementation type of
	// the stored version - which a released version never changes. Writing the definition first would
	// therefore report meshStack's policy rejection rather than the immutability error, and would store
	// an approval gate for a version that stays as it is.
	t.Run("17_released_version_rejected_before_policy_write", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Create draft v1 on manual, which accepts no policy at all.
				{Config: hclStep(t, "resources/building_block_definition/17_released_version_rejected_before_policy_write", 1, vars)},
				// Release v1, which makes its version_spec immutable.
				{Config: hclStep(t, "resources/building_block_definition/17_released_version_rejected_before_policy_write", 2, vars)},
				// Switch the implementation to terraform and enable an approval gate on the released version.
				// terraform supports the gate, so the plan-time validators pass and this reaches Update, where
				// the immutable version_spec is what must be reported.
				// outputs on the manual example are its manual-only sparse overrides, which terraform has no use for.
				{
					Config:      hclStep(t, "resources/building_block_definition/17_released_version_rejected_before_policy_write", 3, vars),
					ExpectError: expectErrorWords("version_spec", "non-draft", "released", "allowed"),
				},
				// The rejected apply must not have written the approval gate. Re-planning the configuration
				// that was last applied refreshes from meshStack, so a stored policy change shows up here as a
				// non-empty plan.
				{Config: hclStep(t, "resources/building_block_definition/17_released_version_rejected_before_policy_write", 4, vars), PlanOnly: true},
			},
		})
	})

	t.Run("18_release_with_change_rejected", func(t *testing.T) {
		vars := hclStepVars{"suffix": acctest.RandString(8)}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{Config: hclStep(t, "resources/building_block_definition/18_release_with_change_rejected", 1, vars)},
				{
					Config:      hclStep(t, "resources/building_block_definition/18_release_with_change_rejected", 2, vars),
					ExpectError: expectErrorWords("release", "draft", "changes"),
				},
				{Config: hclStep(t, "resources/building_block_definition/18_release_with_change_rejected", 3, vars)},
			},
		})
	})
}

// checkBBDMetadataFull checks metadata for the 01_terraform example (tags with 2 entries).
func checkBBDMetadataFull(suffix string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
		"tags": knownvalue.MapExact(map[string]knownvalue.Check{
			"environment-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("dev"), knownvalue.StringExact("prod")}),
			"cost-center-" + suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("cc-123")}),
		}),
	})
}

// checkBBDMetadataMinimal checks metadata for examples without tags.
func checkBBDMetadataMinimal(suffix string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"uuid":               xknownvalue.NotEmptyString(),
		"owned_by_workspace": knownvalue.StringExact("test-ws-" + suffix),
		"tags":               knownvalue.MapSizeExact(0),
	})
}

// checkBBDSpecFull checks spec for the 01_terraform example (all optional attributes set).
func checkBBDSpecFull(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name": knownvalue.StringExact("Example Building Block"),
		"symbol": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "data:image/png;base64,") {
				return fmt.Errorf("value does not start with %s", "data:image/png;base64,")
			}
			return nil
		}),
		"display_name_template": knownvalue.StringExact("Example Building Block {{ resource_name }}"),
		"description":           knownvalue.StringExact(expectedDescription),
		"readme":                knownvalue.StringExact("# Example Building Block\n\nThis is a comprehensive example showcasing all available attributes."),
		"support_url":           knownvalue.StringExact("https://support.example.com/building-blocks"),
		"documentation_url":     knownvalue.StringExact("https://docs.example.com/building-blocks"),
		"target_type":           knownvalue.StringExact("TENANT_LEVEL"),
		"supported_platforms": knownvalue.SetExact([]knownvalue.Check{
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AZURE"),
				"uuid": knownvalue.Null(),
			}),
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"kind": knownvalue.StringExact("meshPlatformType"),
				"name": knownvalue.StringExact("AWS"),
				"uuid": knownvalue.Null(),
			}),
		}),
		"run_transparency":          knownvalue.Bool(true),
		"use_in_landing_zones_only": knownvalue.Bool(true),
		"notification_subscribers": knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("email:ops@example.com"),
		}),
		"approval_policies": checkBBDApprovalPolicies(map[string]bool{
			"version_upgrade": true,
			"manual_triggers": true,
		}),
		"schedule": checkBBDSchedule(
			client.MeshBuildingBlockScheduleModeDriftDetection,
			client.MeshBuildingBlockScheduleFrequencyDaily,
			false,
		),
	})
}

func checkBBDApprovalPolicies(enabled map[string]bool) knownvalue.Check {
	checks := make(map[string]knownvalue.Check, 5)
	for _, gate := range []string{"version_upgrade", "user_input_changes", "manual_triggers", "building_block_creation", "any_input_changes"} {
		checks[gate] = knownvalue.Bool(enabled[gate])
	}
	return xknownvalue.MapExact(checks)
}

func checkBBDSchedule(
	mode enum.Entry[client.MeshBuildingBlockScheduleMode],
	frequency enum.Entry[client.MeshBuildingBlockScheduleFrequency],
	automaticApproval bool,
) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"mode":               knownvalue.StringExact(mode.String()),
		"frequency":          knownvalue.StringExact(frequency.String()),
		"automatic_approval": knownvalue.Bool(automaticApproval),
	})
}

// checkBBDSpecMinimal checks spec for examples with only required attributes (workspace-level, no extras).
func checkBBDSpecMinimal(expectedDescription string) knownvalue.Check {
	return xknownvalue.MapExact(map[string]knownvalue.Check{
		"display_name":              knownvalue.StringExact("Example Building Block"),
		"symbol":                    xknownvalue.NotEmptyString(),
		"display_name_template":     knownvalue.Null(),
		"description":               knownvalue.StringExact(expectedDescription),
		"readme":                    knownvalue.Null(),
		"support_url":               knownvalue.Null(),
		"documentation_url":         knownvalue.Null(),
		"target_type":               knownvalue.StringExact("WORKSPACE_LEVEL"),
		"supported_platforms":       knownvalue.Null(),
		"run_transparency":          knownvalue.Bool(false),
		"use_in_landing_zones_only": knownvalue.Bool(false),
		"notification_subscribers":  knownvalue.SetSizeExact(0),
		// Omitted in the example, so both fall back to the neutral default.
		"approval_policies": checkBBDApprovalPolicies(nil),
		"schedule": checkBBDSchedule(
			client.MeshBuildingBlockScheduleModeDisabled,
			client.MeshBuildingBlockScheduleFrequencyNone,
			false,
		),
	})
}

func checkBuildingBlockVersionSpec(suffix, exampleSuffix string, expectedState enum.Entry[client.MeshBuildingBlockDefinitionVersionState], expectedNumber int64) knownvalue.Check {
	checkInputs, checkImplementation, checkOutputs := checksForImplementation(suffix, exampleSuffix)
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

func checksForImplementation(suffix, exampleSuffix string) (checkInputs, checkImplementation, checkOutputs knownvalue.Check) {
	switch exampleSuffix {
	case "01_terraform":
		return xknownvalue.MapExact(map[string]knownvalue.Check{
				"environment": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Environment"),
					"type":                   knownvalue.StringExact("SINGLE_SELECT"),
					"assignment_type":        knownvalue.StringExact("USER_INPUT"),
					"is_environment":         knownvalue.Bool(false),
					"updateable_by_consumer": knownvalue.Bool(false),
					"is_optional":            knownvalue.Bool(true),
					"description":            knownvalue.StringExact("The target environment"),
					"selectable_values": knownvalue.ListExact([]knownvalue.Check{
						knownvalue.StringExact("dev"),
						knownvalue.StringExact("prod"),
						knownvalue.StringExact("staging"),
					}),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.StringExact("Name of the resource to create"),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.StringExact(`"some-resource-name"`),
					"value_validation_regex":         knownvalue.StringExact("^[a-z0-9-]+$"),
					"validation_regex_error_message": knownvalue.StringExact("Resource name must contain only lowercase letters, numbers, and hyphens"),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(2),
				}),
				// The backend stores the schema verbatim, so state carries exactly what jsonencode produced.
				"deploy_settings": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Deploy Settings"),
					"type":                           knownvalue.StringExact("JSON"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"json_schema":                    knownvalue.StringExact(`{"properties":{"region":{"enum":["eu-central-1","us-east-1"],"type":"string"},"replicas":{"minimum":1,"type":"integer"}},"required":["region"],"type":"object"}`),
					"condition":                      knownvalue.StringExact("input.environment == 'prod'"),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(3),
				}),
				"SOMETHING_VERY_SECRET": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":           knownvalue.StringExact("Top Secret"),
					"type":                   knownvalue.StringExact("STRING"),
					"assignment_type":        knownvalue.StringExact("STATIC"),
					"is_environment":         knownvalue.Bool(true),
					"updateable_by_consumer": knownvalue.Bool(false),
					"is_optional":            knownvalue.Bool(false),
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
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
				"business_unit": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Business Unit"),
					"type":                           knownvalue.StringExact("CODE"),
					"assignment_type":                knownvalue.StringExact("TAG"),
					"is_environment":                 knownvalue.Bool(false),
					"is_optional":                    knownvalue.Bool(true),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"description":                    knownvalue.StringExact("The business unit tag of the workspace this building block belongs to"),
					"argument":                       knownvalue.StringExact(`"WORKSPACE.business-unit-` + suffix + `"`),
					"default_value":                  knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(4),
				}),
				"some-file.yaml": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Some input file"),
					"type":                           knownvalue.StringExact("FILE"),
					"assignment_type":                knownvalue.StringExact("STATIC"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"argument":                       xknownvalue.NotEmptyString(),
					"default_value":                  knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
					"argument":                       knownvalue.Null(),
					"default_value":                  knownvalue.Null(),
					"sensitive":                      knownvalue.Null(),
					"display_order":                  knownvalue.Int64Exact(0),
				}),
				"resource_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":                   knownvalue.StringExact("Resource URL"),
					"type":                           knownvalue.StringExact("STRING"),
					"assignment_type":                knownvalue.StringExact("USER_INPUT"),
					"is_environment":                 knownvalue.Bool(false),
					"updateable_by_consumer":         knownvalue.Bool(false),
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
			// Manual outputs are a sparse override. The example overrides only resource_url (marked
			// RESOURCE_URL, renamed, positioned), which is tracked; approval_required is derived (assignment
			// NONE, display_name = the input's) and pruned. type is always the input's derived output type.
			xknownvalue.MapExact(map[string]knownvalue.Check{
				"resource_url": xknownvalue.MapExact(map[string]knownvalue.Check{
					"display_name":    knownvalue.StringExact("Provisioned Resource"),
					"type":            knownvalue.StringExact("STRING"),
					"assignment_type": knownvalue.StringExact("RESOURCE_URL"),
					"display_order":   knownvalue.Int64Exact(1),
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
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
					"is_optional":                    knownvalue.Bool(false),
					"description":                    knownvalue.Null(),
					"selectable_values":              knownvalue.Null(),
					"value_validation_regex":         knownvalue.Null(),
					"validation_regex_error_message": knownvalue.Null(),
					"json_schema":                    knownvalue.Null(),
					"condition":                      knownvalue.Null(),
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
		"async":                          knownvalue.Bool(false),
		"repository_path":                knownvalue.StringExact("terraform/modules/example"),
		"ref_name":                       knownvalue.StringExact("v1.0.0"),
		"use_mesh_http_backend_fallback": knownvalue.Bool(true),
		"ssh_known_host": xknownvalue.MapExact(map[string]knownvalue.Check{
			"host":      knownvalue.StringExact("github.com"),
			"key_type":  knownvalue.StringExact("ssh-rsa"),
			"key_value": knownvalue.StringExact("AAAAB3NzaC1yc2EAAAABIwAAAQEAq2A7hRGmdnm9tUDbO9IDSwBK6TbQa+..."),
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
		"repository":            knownvalue.StringExact("building-block"),
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

func TestAccBuildingBlockDefinitionPolicyValidation(t *testing.T) {
	// The policy validators are client-side only, so the mock client is enough and a real backend adds
	// nothing - it would only reject the same configurations later, with its own wording.
	if !IsMockClientTest() {
		t.Skip("policy validation is tested with mock client only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name: "approval policies and drift detection on terraform",
			dir:  "approval_policies_and_drift_detection_on_terraform",
		},
		{
			name: "no policy at all on manual",
			dir:  "no_policy_at_all_on_manual",
		},
		{
			name:        "approval gate on manual is rejected",
			dir:         "approval_gate_on_manual_is_rejected",
			expectError: expectErrorWords("Approvals", "dry"),
		},
		{
			name:        "drift detection on manual is rejected",
			dir:         "drift_detection_on_manual_is_rejected",
			expectError: expectErrorWords("Scheduling", "supported", "manual"),
		},
		{
			name:        "frequency without a schedule is rejected",
			dir:         "frequency_without_a_schedule_is_rejected",
			expectError: expectErrorWords("frequency", "without", "schedule"),
		},
		{
			name:        "schedule without a frequency is rejected",
			dir:         "schedule_without_a_frequency_is_rejected",
			expectError: expectErrorWords("frequency", "required"),
		},
		{
			name:        "automatic approval without reconciliation is rejected",
			dir:         "automatic_approval_without_reconciliation_is_rejected",
			expectError: expectErrorWords("Automatic", "approval", "drift"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      hclStep(t, "resources/building_block_definition/policy_validation/"+test.dir, 1, hclStepVars{"suffix": acctest.RandString(8)}),
					ExpectError: test.expectError,
				}},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionTagInputValidation(t *testing.T) {
	// The tag input validator is client-side only, so the mock client is enough and a real backend adds
	// nothing - it would only reject the same configurations later, with its own wording.
	if !IsMockClientTest() {
		t.Skip("tag input validation is tested with mock client only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name: "a workspace tag on a workspace building block",
			dir:  "a_workspace_tag_on_a_workspace_building_block",
		},
		{
			name: "a project tag on a tenant building block",
			dir:  "a_project_tag_on_a_tenant_building_block",
		},
		{
			// Only the first separator splits the reference, so a tag key may contain a dot itself.
			name: "a tag key containing a dot",
			dir:  "a_tag_key_containing_a_dot",
		},
		{
			name:        "a project tag on a workspace building block is rejected",
			dir:         "a_project_tag_on_a_workspace_building_block_is_rejected",
			expectError: expectErrorWords("tag", "read", "target"),
		},
		{
			name:        "a non-code input type is rejected",
			dir:         "a_non-code_input_type_is_rejected",
			expectError: expectErrorWords("tag", "code"),
		},
		{
			name:        "a sensitive tag input is rejected",
			dir:         "a_sensitive_tag_input_is_rejected",
			expectError: expectErrorWords("tag", "sensitive"),
		},
		{
			name:        "a missing argument is rejected",
			dir:         "a_missing_argument_is_rejected",
			expectError: expectErrorWords("tag", "requires", "argument"),
		},
		{
			name:        "an argument without a tag key is rejected",
			dir:         "an_argument_without_a_tag_key_is_rejected",
			expectError: expectErrorWords("tag", "argument", "target", "key"),
		},
		{
			name:        "an unknown target is rejected",
			dir:         "an_unknown_target_is_rejected",
			expectError: expectErrorWords("tag", "argument", "known", "target"),
		},
		{
			name:        "an argument that is not a string is rejected",
			dir:         "an_argument_that_is_not_a_string_is_rejected",
			expectError: expectErrorWords("tag", "argument", "encoded"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      hclStep(t, "resources/building_block_definition/tag_input_validation/"+test.dir, 1, hclStepVars{"suffix": acctest.RandString(8)}),
					ExpectError: test.expectError,
				}},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionPaymentMethodInputValidation(t *testing.T) {
	// Like the tag input validator, this one is client-side only, so the mock client is enough.
	if !IsMockClientTest() {
		t.Skip("payment method input validation is tested with mock client only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name: "a code input on a workspace building block",
			dir:  "a_code_input_on_a_workspace_building_block",
		},
		{
			name: "the default target type is a workspace building block",
			dir:  "the_default_target_type_is_a_workspace_building_block",
		},
		{
			name:        "a tenant building block is rejected",
			dir:         "a_tenant_building_block_is_rejected",
			expectError: expectErrorWords("Payment", "needs", "workspace"),
		},
		{
			name:        "a non-code input type is rejected",
			dir:         "a_non-code_input_type_is_rejected",
			expectError: expectErrorWords("Payment", "code"),
		},
		{
			name:        "a default value is rejected",
			dir:         "a_default_value_is_rejected",
			expectError: expectErrorWords("Payment", "default_value"),
		},
		{
			name:        "a sensitive input is rejected",
			dir:         "a_sensitive_input_is_rejected",
			expectError: expectErrorWords("Payment", "sensitive"),
		},
		{
			name:        "selectable values are rejected",
			dir:         "selectable_values_are_rejected",
			expectError: expectErrorWords("Payment", "selectable_values"),
		},
		{
			name:        "an optional input is rejected",
			dir:         "an_optional_input_is_rejected",
			expectError: expectErrorWords("Payment", "optional"),
		},
		{
			name:        "a validation regex is rejected",
			dir:         "a_validation_regex_is_rejected",
			expectError: expectErrorWords("Payment", "value_validation_regex"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      hclStep(t, "resources/building_block_definition/payment_method_input_validation/"+test.dir, 1, hclStepVars{"suffix": acctest.RandString(8)}),
					ExpectError: test.expectError,
				}},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionSymbolValidation(t *testing.T) {
	// Symbol validation is client-side only; success cases need a real workspace in acceptance mode.
	if !IsMockClientTest() {
		t.Skip("symbol validation is tested with mock client only")
	}

	t.Parallel()

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
			expectError: expectErrorWords("Invalid", "Symbol", "Format"),
		},
		{
			name:        "disallowed image type is rejected",
			symbol:      "data:image/bmp;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 50))),
			expectError: expectErrorWords("Invalid", "Symbol", "Format"),
		},
		{
			name:        "invalid base64 is rejected",
			symbol:      "data:image/png;base64,!!!not-valid-base64!!!",
			expectError: expectErrorWords("Base64", "Symbol"),
		},
		{
			name:   "data URI decoded size exactly at 100 KiB limit",
			symbol: "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024))),
		},
		{
			name:        "data URI decoded size exceeds 100 KiB limit",
			symbol:      "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte(strings.Repeat("x", 100*1024+1))),
			expectError: expectErrorWords("Symbol", "Large"),
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
				Config: hclStep(t, "resources/building_block_definition/symbol_validation", 1, hclStepVars{"suffix": acctest.RandString(8), "symbol": tt.symbol}),
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

func TestAccBuildingBlockDefinitionSupportedPlatformKinds(t *testing.T) {
	vars := hclStepVars{"suffix": acctest.RandString(8)}
	platformAddr := "meshstack_platform.test"

	platformTypeRef := func(name knownvalue.Check) knownvalue.Check {
		return xknownvalue.MapExact(map[string]knownvalue.Check{
			"kind": knownvalue.StringExact(client.MeshObjectKind.PlatformType),
			"name": name,
			"uuid": knownvalue.Null(),
		})
	}

	var appliedPlatformUuid string
	capturePlatformUuid := statecheck.ExpectKnownValue(
		platformAddr,
		tfjsonpath.New("metadata").AtMapKey("uuid"),
		xknownvalue.NotEmptyString(func(actualValue string) error {
			appliedPlatformUuid = actualValue
			return nil
		}),
	)
	platformRef := xknownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact(client.MeshObjectKind.Platform),
		"name": knownvalue.Null(),
		"uuid": xknownvalue.NotEmptyString(func(actualValue string) error {
			if actualValue != appliedPlatformUuid {
				return fmt.Errorf("expected the platform's uuid %q, got %q", appliedPlatformUuid, actualValue)
			}
			return nil
		}),
	})

	landingZoneRef := xknownvalue.MapExact(map[string]knownvalue.Check{
		"kind": knownvalue.StringExact(client.MeshObjectKind.LandingZone),
		"name": knownvalue.StringExact("test-lz-" + vars["suffix"]),
		"uuid": knownvalue.Null(),
	})

	transitions := [][]knownvalue.Check{
		{platformRef},
		{landingZoneRef},
		{
			platformTypeRef(knownvalue.StringExact("AZURE")),
			platformRef,
			landingZoneRef,
		},
		{
			platformTypeRef(knownvalue.StringExact("AZURE")),
			platformRef,
		},
		{platformTypeRef(knownvalue.StringExact(strings.ToUpper("test-pt-" + vars["suffix"])))},
		{platformRef},
	}

	steps := make([]resource.TestStep, 0, len(transitions))
	for position, expectStored := range transitions {
		steps = append(steps, resource.TestStep{
			Config: hclStep(t, "resources/building_block_definition/supported_platform_kinds", position+1, vars),
			ConfigStateChecks: []statecheck.StateCheck{
				capturePlatformUuid,
				statecheck.ExpectKnownValue(
					"meshstack_building_block_definition.test",
					tfjsonpath.New("spec").AtMapKey("supported_platforms"),
					knownvalue.SetExact(expectStored),
				),
			},
		})
	}

	ApplyAndTest(t, resource.TestCase{Steps: steps})
}

func TestAccBuildingBlockDefinitionSupportedPlatformValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name:        "an unknown kind is rejected",
			dir:         "an_unknown_kind_is_rejected",
			expectError: expectErrorWords("one", "meshPlatformType", "meshPlatform", "meshLandingZone"),
		},
		{
			name:        "a landing zone identified by uuid is rejected",
			dir:         "a_landing_zone_identified_by_uuid_is_rejected",
			expectError: expectErrorWords("Invalid", "Attribute", "Combination"),
		},
		{
			name:        "a platform identified by name is rejected",
			dir:         "a_platform_identified_by_name_is_rejected",
			expectError: expectErrorWords("Invalid", "Attribute", "Combination"),
		},
		{
			name:        "a platform type identified by uuid is rejected",
			dir:         "a_platform_type_identified_by_uuid_is_rejected",
			expectError: expectErrorWords("Invalid", "Attribute", "Combination"),
		},
		{
			name:        "an entry with both identifiers is rejected",
			dir:         "an_entry_with_both_identifiers_is_rejected",
			expectError: expectErrorWords("Invalid", "Attribute", "Combination"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      hclStep(t, "resources/building_block_definition/supported_platform_validation/"+test.dir, 1, hclStepVars{"suffix": acctest.RandString(8)}),
					ExpectError: test.expectError,
				}},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionOptionalInputValidation(t *testing.T) {
	// The rules an optional input has to satisfy are mirrored client-side, so they surface at plan time
	// instead of as a backend 400 during apply.
	if !IsMockClientTest() {
		t.Skip("optional input validation is tested with mock client only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			// A manual building block is carried out by a person, so there is no code to fall back to.
			name:        "optional input rejected on manual implementation",
			dir:         "optional_input_rejected_on_manual_implementation",
			expectError: expectErrorWords("optional", "manual"),
		},
		{
			// A STATIC value is always there, so there is nothing for the implementation to fall back from.
			name:        "optional input rejected for a STATIC assignment type",
			dir:         "optional_input_rejected_for_a_static_assignment_type",
			expectError: expectErrorWords("optional", "assignment_type"),
		},
		{
			name:        "optional BOOLEAN input rejected",
			dir:         "optional_boolean_input_rejected",
			expectError: expectErrorWords("BOOLEAN", "optional"),
		},
		{
			name:        "optional input with default_value rejected",
			dir:         "optional_input_with_default_value_rejected",
			expectError: expectErrorWords("default", "value"),
		},
		{
			name:        "optional input with sensitive default_value rejected",
			dir:         "optional_input_with_sensitive_default_value_rejected",
			expectError: expectErrorWords("default", "value"),
		},
		{
			name: "optional USER_INPUT accepted",
			dir:  "optional_user_input_accepted",
		},
		{
			name: "optional PLATFORM_OPERATOR_MANUAL_INPUT accepted",
			dir:  "optional_platform_operator_manual_input_accepted",
		},
		{
			name: "optional TAG accepted",
			dir:  "optional_tag_accepted",
		},
		{
			// None of the rules applies to an input that is not optional.
			name: "non-optional BOOLEAN with default_value accepted",
			dir:  "non-optional_boolean_with_default_value_accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: hclStep(t, "resources/building_block_definition/optional_input_validation/"+tt.dir, 1, hclStepVars{"suffix": acctest.RandString(8)})}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionJsonSchemaValidation(t *testing.T) {
	// The json_schema/type pairing is a ValidateConfig rule, so it never reaches the API. Running this
	// against a deployed meshStack would also need one that already knows the JSON input type.
	if !IsMockClientTest() {
		t.Skip("json_schema pairing is validated client-side only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name: "JSON input with a schema accepted",
			dir:  "json_input_with_a_schema_accepted",
		},
		{
			name:        "JSON input without a schema rejected",
			dir:         "json_input_without_a_schema_rejected",
			expectError: expectErrorWords("json_schema", "required"),
		},
		{
			name:        "schema on an input of another type rejected",
			dir:         "schema_on_an_input_of_another_type_rejected",
			expectError: expectErrorWords("json_schema", "not", "set"),
		},
		{
			name: "input of another type without a schema accepted",
			dir:  "input_of_another_type_without_a_schema_accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: hclStep(t, "resources/building_block_definition/json_schema_validation/"+tt.dir, 1, hclStepVars{"suffix": acctest.RandString(8)})}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}

func TestAccBuildingBlockDefinitionConditionValidation(t *testing.T) {
	// Which input may carry a condition is mirrored client-side, so it surfaces at plan time instead of as a
	// backend 400 during apply. What a condition may read stays with the backend, which compiles the expression.
	if !IsMockClientTest() {
		t.Skip("condition carrier validation is tested with mock client only")
	}

	t.Parallel()

	tests := []struct {
		name        string
		dir         string
		expectError *regexp.Regexp
	}{
		{
			name:        "condition rejected on manual implementation",
			dir:         "condition_rejected_on_manual_implementation",
			expectError: expectErrorWords("condition", "manual"),
		},
		{
			name:        "condition rejected for a non-user assignment type",
			dir:         "condition_rejected_for_a_non-user_assignment_type",
			expectError: expectErrorWords("condition", "assignment_type"),
		},
		{
			name:        "blank condition rejected",
			dir:         "blank_condition_rejected",
			expectError: expectErrorWords("not", "blank"),
		},
		{
			name:        "overlong condition rejected",
			dir:         "overlong_condition_rejected",
			expectError: expectErrorWords("length", "most", "512"),
		},
		{
			name: "conditional USER_INPUT accepted",
			dir:  "conditional_user_input_accepted",
		},
		{
			name: "conditional PLATFORM_OPERATOR_MANUAL_INPUT accepted",
			dir:  "conditional_platform_operator_manual_input_accepted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: hclStep(t, "resources/building_block_definition/condition_validation/"+tt.dir, 1, hclStepVars{"suffix": acctest.RandString(8)})}
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

	// Two inputs so a partial declaration can be exercised. STRING/STATIC inputs derive STRING outputs
	// (STRING passes through the manual IO type translation unchanged).
	type testCase struct {
		name           string
		dir            string
		assignmentType string
		expectError    *regexp.Regexp
	}
	tests := []testCase{
		{
			// Outputs are keyed by input; a key with no matching input is rejected (mirrors the backend 400).
			name:        "output without a matching input rejected",
			dir:         "output_without_a_matching_input_rejected",
			expectError: expectErrorWords("no", "matching", "input"),
		},
		{
			// type is always derived for manual outputs, so declaring it is rejected.
			name:        "declared type rejected on manual output",
			dir:         "declared_type_rejected_on_manual_output",
			expectError: expectErrorWords("type", "not", "set"),
		},
		{
			// A no-op override (NONE assignment and no display_name) cannot be reconstructed by the diff rule.
			name:        "no-op override rejected (NONE assignment, no display_name)",
			dir:         "no-op_override_rejected_none_assignment_no_display_name",
			expectError: expectErrorWords("no", "effect"),
		},
		{
			name:        "no-op override rejected (empty object)",
			dir:         "no-op_override_rejected_empty_object",
			expectError: expectErrorWords("no", "effect"),
		},
		{
			// display_order alone is not a supported override (deliberately not a membership signal).
			name:        "display_order-only override rejected",
			dir:         "display_order-only_override_rejected",
			expectError: expectErrorWords("no", "effect"),
		},
		{
			// A display_name equal to the input's is not a real override.
			name:        "display_name equal to input rejected as no-op",
			dir:         "display_name_equal_to_input_rejected_as_no-op",
			expectError: expectErrorWords("no", "effect"),
		},
		// Accepted: a sparse subset, an empty map (no overrides), and a display_name override.
		{
			name: "empty outputs map accepted (no overrides)",
			dir:  "empty_outputs_map_accepted_no_overrides",
		},
		{
			name: "subset override via display_name accepted",
			dir:  "subset_override_via_display_name_accepted",
		},
	}

	// Any non-NONE assignment_type is a meaningful subset override and is accepted; loop the whole enum so a
	// new entry is covered without editing this test.
	none := client.MeshBuildingBlockDefinitionOutputAssignmentTypeNone.String()
	for _, assignmentType := range client.MeshBuildingBlockDefinitionOutputAssignmentTypes {
		if assignmentType.String() == none {
			continue
		}
		tests = append(tests, testCase{
			name:           fmt.Sprintf("subset override with %s accepted", assignmentType),
			dir:            "subset_override_with_assignment_type_accepted",
			assignmentType: assignmentType.String(),
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vars := hclStepVars{"suffix": acctest.RandString(8), "assignment_type": tt.assignmentType}
			step := resource.TestStep{Config: hclStep(t, "resources/building_block_definition/manual_outputs_validation/"+tt.dir, 1, vars)}
			if tt.expectError != nil {
				step.ExpectError = tt.expectError
			}
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{step},
			})
		})
	}
}
