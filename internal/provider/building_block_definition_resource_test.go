package provider

import (
	_ "embed"
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/compare"
	tfconfig "github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/meshcloud/meshstack-cli/client"
	"github.com/meshcloud/meshstack-cli/client/types/enum"

	"github.com/meshcloud/terraform-provider-meshstack/examples"
	"github.com/meshcloud/terraform-provider-meshstack/internal/provider/acctest/xknownvalue"
)

// Addresses of the blocks in examples/resources/meshstack_building_block_definition/resource-test-*.tf,
// and the tag-key prefixes its test-support_tags.tf declares.
const (
	terraformBbdAddr   = "meshstack_building_block_definition.example_01_terraform"
	githubBbdAddr      = "meshstack_building_block_definition.example_02_github_workflows"
	manualBbdResAddr   = "meshstack_building_block_definition.example_03_manual"
	azureDevopsBbdAddr = "meshstack_building_block_definition.example_04_azure_devops_pipeline"
	gitlabBbdAddr      = "meshstack_building_block_definition.example_05_gitlab_pipeline"

	bbdTagKeyPrefix = "test-key-bbd-"
)

// The support files each definition variant needs beside its own step file.
var (
	terraformBbdSupports = []string{
		"01_terraform_tag-environment",
		"01_terraform_tag-cost-center",
		"01_terraform_tag-workspace-business-unit",
		"01_terraform_dependency-bbd",
	}
	githubBbdSupports      = []string{"02_github_workflows_integration"}
	azureDevopsBbdSupports = []string{"04_azure_devops_pipeline_integration"}
	gitlabBbdSupports      = []string{"05_gitlab_pipeline_integration"}
)

// bbdStepConfig is a definition's step with the workspace owning it, plus whatever that variant
// references — an integration, the tag definitions, the dependency definition.
func bbdStepConfig(t *testing.T, index int, supports ...string) string {
	t.Helper()
	return examples.JoinTestStepConfigs(
		examples.Resource.TestStepConfig(t, "building_block_definition", index, append([]string{"variables"}, supports...)...),
		examples.Resource.TestStepConfig(t, "workspace", 1, "variables", "prerequisites"),
	)
}

// bbdVars dials the attributes a definition case changes while the rest of the definition stays
// byte-identical. The zero value beyond the suffix is the example as documented: a draft, the
// example's own description, no policies and no name template.
type bbdVars struct {
	suffix              string
	description         string
	draft               *bool
	secretValue         string
	secretVersion       string
	displayNameTemplate *string
	preRunScript        string
	// The policy attributes are objects, so they are typed variables rather than strings.
	approvalPolicies tfconfig.Variable
	schedule         tfconfig.Variable
}

func newBbdVars() bbdVars { return bbdVars{suffix: acctest.RandString(8)} }

func (v bbdVars) variables() tfconfig.Variables {
	vars := SuffixVariables(v.suffix)
	vars["tag_suffix"] = tfconfig.StringVariable(v.suffix)
	if v.description != "" {
		vars["description"] = tfconfig.StringVariable(v.description)
	}
	if v.draft != nil {
		vars["draft"] = tfconfig.BoolVariable(*v.draft)
	}
	if v.secretValue != "" {
		vars["secret_value"] = tfconfig.StringVariable(v.secretValue)
		vars["secret_version"] = tfconfig.StringVariable(v.secretVersion)
	}
	if v.displayNameTemplate != nil {
		vars["display_name_template"] = tfconfig.StringVariable(*v.displayNameTemplate)
	}
	if v.preRunScript != "" {
		vars["pre_run_script"] = tfconfig.StringVariable(v.preRunScript)
	}
	if v.approvalPolicies != nil {
		vars["approval_policies"] = v.approvalPolicies
	}
	if v.schedule != nil {
		vars["schedule"] = v.schedule
	}
	return vars
}

// withPolicies returns v with spec.approval_policies and spec.schedule set. Which of these meshStack
// accepts depends on the implementation the stored version carries, which is what the policy cases
// walk across an implementation swap.
func (v bbdVars) withPolicies(approvalPolicies, schedule tfconfig.Variable) bbdVars {
	v.approvalPolicies, v.schedule = approvalPolicies, schedule
	return v
}

// bbdApprovalPolicies builds the approval_policies object from the gates that are enabled.
func bbdApprovalPolicies(gates ...string) tfconfig.Variable {
	policies := map[string]tfconfig.Variable{}
	for _, gate := range gates {
		policies[gate] = tfconfig.BoolVariable(true)
	}
	return tfconfig.ObjectVariable(policies)
}

// bbdSchedule builds the schedule object; automaticApproval is only meaningful for reconciliation.
func bbdSchedule(mode string, automaticApproval bool) tfconfig.Variable {
	schedule := map[string]tfconfig.Variable{
		"mode":      tfconfig.StringVariable(mode),
		"frequency": tfconfig.StringVariable("DAILY"),
	}
	if automaticApproval {
		schedule["automatic_approval"] = tfconfig.BoolVariable(true)
	}
	return tfconfig.ObjectVariable(schedule)
}

// released returns v with the version released rather than drafted.
func (v bbdVars) released() bbdVars {
	released := false
	v.draft = &released
	return v
}

// withDescription returns v with the definition's description changed, which updates the spec
// without cutting a new version.
func (v bbdVars) withDescription(description string) bbdVars {
	v.description = description
	return v
}

// withSecret returns v with the sensitive STATIC input's secret rotated.
func (v bbdVars) withSecret(version, value string) bbdVars {
	v.secretVersion, v.secretValue = version, value
	return v
}

// withNameTemplate returns v with spec.display_name_template set to the given value; "" is an
// explicitly empty template, which meshStack treats like no template but Terraform does not.
func (v bbdVars) withNameTemplate(template string) bbdVars {
	v.displayNameTemplate = &template
	return v
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
		vars := newBbdVars()
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 1, terraformBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("metadata"), checkBBDMetadataFull()),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("spec"), checkBBDSpecFull(bbdDescription)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("01_terraform", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(terraformBbdAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Change secret input name (remove/add operation on inputs map). Step file 21 is
				// step file 1 with SOMETHING_VERY_SECRET renamed and nothing else touched.
				{
					Config:          bbdStepConfig(t, 21, terraformBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						},
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars.variables(),
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: terraformBbdAddr,
				},
			},
		})
	})

	t.Run("02_github_workflows", func(t *testing.T) {
		vars := newBbdVars()
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 2, githubBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(githubBbdAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("02_github_workflows", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(githubBbdAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars.variables(),
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: githubBbdAddr,
				},
			},
		})
	})

	t.Run("03_manual", func(t *testing.T) {
		vars := newBbdVars()
		updated := vars.withDescription("An updated building block definition")
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(manualBbdResAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 2: Update spec (description change, no new version)
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: updated.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal("An updated building block definition")),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(manualBbdResAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Release (draft=false)
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.released().variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 1)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateReleased)})),
					},
				},
				// Step 4: New draft (draft=true again, description changed)
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: updated.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateDraft, 2)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
					},
				},
				// Step 5: Release the new draft (draft=false)
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.released().variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("03_manual", versionStateReleased, 2)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateReleased),
						})),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars.released().variables(),
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: manualBbdResAddr,
				},
			},
		})
	})

	t.Run("04_azure_devops_pipeline", func(t *testing.T) {
		vars := newBbdVars()
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 5, azureDevopsBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(azureDevopsBbdAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("04_azure_devops_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(azureDevopsBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(azureDevopsBbdAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars.variables(),
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: azureDevopsBbdAddr,
				},
			},
		})
	})

	t.Run("05_gitlab_pipeline", func(t *testing.T) {
		vars := newBbdVars()
		var resourceUuid string

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Plan-only (ensure tf plan works before apply)
				{
					Config:             bbdStepConfig(t, 6, gitlabBbdSupports...),
					ConfigVariables:    vars.variables(),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
				// Step 2: Create
				{
					Config:          bbdStepConfig(t, 6, gitlabBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(gitlabBbdAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("version_spec"), checkBuildingBlockVersionSpec("05_gitlab_pipeline", versionStateDraft, 1)),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("version_latest_release"), knownvalue.Null()),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(gitlabBbdAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				// Step 3: Import
				{
					ImportState:     true,
					ImportStateKind: resource.ImportBlockWithID,
					ConfigVariables: vars.variables(),
					ImportStateIdFunc: func(state *terraform.State) (string, error) {
						return resourceUuid, nil
					},
					ResourceName: gitlabBbdAddr,
				},
				// Step 4: Rotate the pipeline trigger token after import.
				{
					Config:          bbdStepConfig(t, 22, gitlabBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(gitlabBbdAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("metadata"), checkBBDMetadataMinimal()),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("spec"), checkBBDSpecMinimal(bbdDescription)),
						statecheck.ExpectKnownValue(gitlabBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{expectedVersion(1, versionStateDraft)})),
						xknownvalue.Ref(gitlabBbdAddr, "meshBuildingBlockDefinition", &resourceUuid),
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
		vars := newBbdVars()

		// The released version's content_hash must be identical before and after the new draft
		// is created from it.
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1
				{
					Config:          bbdStepConfig(t, 1, terraformBbdSupports...),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config:          bbdStepConfig(t, 1, terraformBbdSupports...),
					ConfigVariables: vars.released().variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(terraformBbdAddr, releasedHashPath),
					},
				},
				// Step 3: Flip draft false->true AND change the implementation -> new draft v2.
				// The released v1 must stay immutable (same content_hash, no inconsistent result).
				{
					Config:          bbdStepConfig(t, 1, terraformBbdSupports...),
					ConfigVariables: vars.withPreRunScript(`echo "changed for the second version"`).variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(terraformBbdAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("versions"), knownvalue.ListExact([]knownvalue.Check{
							expectedVersion(1, versionStateReleased),
							expectedVersion(2, versionStateDraft),
						})),
						releasedHashStable.AddStateValue(terraformBbdAddr, releasedHashPath),
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
		vars := newBbdVars()

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
					Config:          bbdStepConfig(t, 7),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1
				{
					Config:          bbdStepConfig(t, 7),
					ConfigVariables: vars.released().variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), baseOutputs),
						releasedHashStable.AddStateValue(manualBbdResAddr, releasedHashPath),
					},
				},
				// Step 3: Re-draft (draft false->true) AND add an input -> new draft v2 with reconciled outputs.
				// Step file 8 is step file 7 plus the STATIC ticket input.
				{
					Config:          bbdStepConfig(t, 8),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), redraftOutputs),
						releasedHashStable.AddStateValue(manualBbdResAddr, releasedHashPath),
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
		vars := newBbdVars()

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
					Config:          bbdStepConfig(t, 9),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				// Step 2: Release v1 (bug 2: content_hash must not flip during apply).
				{
					Config:          bbdStepConfig(t, 9),
					ConfigVariables: vars.released().variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						releasedHashStable.AddStateValue(manualBbdResAddr, releasedHashPath),
					},
				},
				// Step 3: Re-draft (draft false->true) with no change -> new draft v2 (bug 2: create new
				// version content_hash must not flip); released v1 stays immutable.
				{
					Config:          bbdStepConfig(t, 9),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						releasedHashStable.AddStateValue(manualBbdResAddr, releasedHashPath),
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
		vars := newBbdVars()

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
					Config:          bbdStepConfig(t, 10),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), initialOutputs),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				{
					Config:          bbdStepConfig(t, 11),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), updatedOutputs),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				{
					Config:          bbdStepConfig(t, 12),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), readdedOutputs),
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
			},
		})
	})

	// Import must materialize exactly the diff-rule subset: with no prior state, the read-back prunes the
	// backend's full one-per-input set to the tracked overrides, so the imported state equals the applied one.
	t.Run("14_manual_output_import", func(t *testing.T) {
		vars := newBbdVars()
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
					Config:          bbdStepConfig(t, 13),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("version_spec").AtMapKey("outputs"), expectedOutputs),
						xknownvalue.Ref(manualBbdResAddr, "meshBuildingBlockDefinition", &resourceUuid),
					},
				},
				{
					// Command-style import (not a plannable import block, which does not support ImportStateVerify).
					ImportState:                          true,
					ConfigVariables:                      vars.variables(),
					ImportStateIdFunc:                    func(_ *terraform.State) (string, error) { return resourceUuid, nil },
					ResourceName:                         manualBbdResAddr,
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
		vars := newBbdVars()
		nameTemplatePath := tfjsonpath.New("spec").AtMapKey("display_name_template")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.withNameTemplate("Block for {{ region }}").variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, nameTemplatePath, knownvalue.StringExact("Block for {{ region }}")),
					},
				},
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.withNameTemplate("Block for {{ region }} v2").variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(manualBbdResAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, nameTemplatePath, knownvalue.StringExact("Block for {{ region }} v2")),
					},
				},
				{
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.withNameTemplate("").variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, nameTemplatePath, knownvalue.StringExact("")),
					},
				},
				{
					// Leaving the variable unset drops the attribute, which removes the template, and the
					// refresh has to agree: a state that still held the old value would leave the same diff on
					// every later plan.
					Config:          bbdStepConfig(t, 3),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{
							plancheck.ExpectEmptyPlan(),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(manualBbdResAddr, nameTemplatePath, knownvalue.Null()),
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
		vars := newBbdVars().withSecret("v1", "plaintext-secret-v1")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Create draft v1 with the sensitive input.
				{Config: bbdStepConfig(t, 14, terraformBbdSupports...), ConfigVariables: vars.variables()},
				// Release v1, which makes its version_spec immutable.
				{Config: bbdStepConfig(t, 14, terraformBbdSupports...), ConfigVariables: vars.released().variables()},
				// Rotating the secret on the released version must be rejected, not reported as a
				// content-hash failure.
				{
					Config:          bbdStepConfig(t, 14, terraformBbdSupports...),
					ConfigVariables: vars.released().withSecret("v2", "plaintext-secret-v2").variables(),
					ExpectError:     regexp.MustCompile("Updating a version_spec in non-draft state is not allowed"),
				},
			},
		})
	})

	// A non-sensitive input named "plaintext" is not a secret. The content hash disallows plaintext
	// keys, and a key-name match used to reject an unrelated spec update on a released version.
	t.Run("09_released_plaintext_named_input_is_not_a_secret", func(t *testing.T) {
		vars := newBbdVars()

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Step 1: Create draft v1 with a non-sensitive input named "plaintext"
				{Config: bbdStepConfig(t, 15, terraformBbdSupports...), ConfigVariables: vars.variables()},
				// Step 2: Release v1 (now immutable)
				{Config: bbdStepConfig(t, 15, terraformBbdSupports...), ConfigVariables: vars.released().variables()},
				// Step 3: Change only the description on the released version. version_spec is unchanged and
				// carries no secret, so this must succeed - the old "plaintext" key match wrongly rejected it.
				{
					Config:          bbdStepConfig(t, 15, terraformBbdSupports...),
					ConfigVariables: vars.released().withDescription("updated description, version_spec untouched").variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("spec").AtMapKey("description"),
							knownvalue.StringExact("updated description, version_spec untouched")),
					},
				},
			},
		})
	})

	// The counterpart to 08: re-drafting a released version while rotating the secret is allowed,
	// because the rotation lands on a new draft rather than the immutable released version.
	t.Run("10_redraft_with_secret_rotation_allowed", func(t *testing.T) {
		vars := newBbdVars().withSecret("v1", "plaintext-secret-v1")

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{Config: bbdStepConfig(t, 14, terraformBbdSupports...), ConfigVariables: vars.variables()},
				{Config: bbdStepConfig(t, 14, terraformBbdSupports...), ConfigVariables: vars.released().variables()},
				// Flipping back to draft while rotating the secret cuts a new version v2, which is where
				// the rotated secret lands.
				{
					Config:          bbdStepConfig(t, 14, terraformBbdSupports...),
					ConfigVariables: vars.withSecret("v2", "plaintext-secret-v2").variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(terraformBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
					},
				},
			},
		})
	})

	// Re-drafting a released github definition with a different integration must leave the released
	// version pinned to the first one.
	t.Run("11_github_release_redraft_integration_change", func(t *testing.T) {
		vars := newBbdVars()
		releasedHashStable := statecheck.CompareValue(compare.ValuesSame())
		releasedHashPath := tfjsonpath.New("version_latest_release").AtMapKey("content_hash")

		// Both integrations live in the same support file pair, so every step carries both.
		supports := []string{"02_github_workflows_integration", "02_github_workflows_integration_b"}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 2, supports...),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(1, versionStateDraft)),
					},
				},
				{
					Config:          bbdStepConfig(t, 2, supports...),
					ConfigVariables: vars.released().variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						releasedHashStable.AddStateValue(githubBbdAddr, releasedHashPath),
					},
				},
				// Step file 16 points at integration B; the released v1 still pins A.
				{
					Config:          bbdStepConfig(t, 16, supports...),
					ConfigVariables: vars.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(githubBbdAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: []statecheck.StateCheck{
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest_release"), expectedVersion(1, versionStateReleased)),
						statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("version_latest"), expectedVersion(2, versionStateDraft)),
						releasedHashStable.AddStateValue(githubBbdAddr, releasedHashPath),
					},
				},
			},
		})
	})

	t.Run("12_restricted_default_tag", func(t *testing.T) {
		// Backend-materialized default: the mock has no tag-restriction business logic, so it can't
		// reproduce the backend injecting a restricted tag's default on create. See the lock-step
		// policy in the acceptance-testing skill.
		if IsMockClientTest() {
			t.Skip("relies on the backend injecting a restricted tag's default value on create")
		}

		vars := newBbdVars()

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				{
					Config:          bbdStepConfig(t, 17, "tags"),
					ConfigVariables: vars.variables(),
					ConfigStateChecks: []statecheck.StateCheck{
						// Only the declared tag remains; the injected restricted default is reconciled away.
						statecheck.ExpectKnownValue(manualBbdResAddr, tfjsonpath.New("metadata").AtMapKey("tags"), knownvalue.MapExact(map[string]knownvalue.Check{
							bbdTagKeyPrefix + vars.suffix: knownvalue.ListExact([]knownvalue.Check{knownvalue.StringExact("blue")}),
						})),
					},
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PostApplyPostRefresh: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
					},
				},
			},
		}, TouchesExclusively(client.MeshObjectKind.BuildingBlockDefinition))
	})

	t.Run("16_policies_across_implementation_change", func(t *testing.T) {
		// github_workflows has no dry run, so it supports drift reconciliation only with automatic
		// approval, and no approval gate at all. terraform supports every policy.
		vars := newBbdVars()
		githubPolicies := vars.withPolicies(
			bbdApprovalPolicies(),
			bbdSchedule("DRIFT_RECONCILIATION", true),
		)
		terraformPolicies := vars.withPolicies(
			bbdApprovalPolicies("manual_triggers", "version_upgrade"),
			bbdSchedule("DRIFT_DETECTION", false),
		)

		checkPolicies := func(approvalPolicies map[string]bool, mode enum.Entry[client.MeshBuildingBlockScheduleMode], automaticApproval bool) []statecheck.StateCheck {
			return []statecheck.StateCheck{
				statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("spec").AtMapKey("approval_policies"), checkBBDApprovalPolicies(approvalPolicies)),
				statecheck.ExpectKnownValue(githubBbdAddr, tfjsonpath.New("spec").AtMapKey("schedule"),
					checkBBDSchedule(mode, client.MeshBuildingBlockScheduleFrequencyDaily, automaticApproval)),
			}
		}

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Create with a schedule enabled: the definition is created with neutral policies, and the
				// schedule follows once the initial version carries the github_workflows implementation.
				{
					Config:          bbdStepConfig(t, 18, githubBbdSupports...),
					ConfigVariables: githubPolicies.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(githubBbdAddr, plancheck.ResourceActionCreate),
						},
					},
					ConfigStateChecks: checkPolicies(nil, client.MeshBuildingBlockScheduleModeDriftReconciliation, true),
				},
				// Tighten across an implementation change: approval gates and drift detection need terraform,
				// so the policies can only be written after the version switched to it. Step file 19 is 18
				// with the implementation swapped.
				{
					Config:          bbdStepConfig(t, 19, githubBbdSupports...),
					ConfigVariables: terraformPolicies.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(githubBbdAddr, plancheck.ResourceActionUpdate),
						},
					},
					ConfigStateChecks: checkPolicies(
						map[string]bool{"manual_triggers": true, "version_upgrade": true},
						client.MeshBuildingBlockScheduleModeDriftDetection, false),
				},
				// Relax across an implementation change: github_workflows supports neither the approval gates
				// nor drift detection, so they have to be gone before the version switches back to it.
				{
					Config:          bbdStepConfig(t, 18, githubBbdSupports...),
					ConfigVariables: githubPolicies.variables(),
					ConfigPlanChecks: resource.ConfigPlanChecks{
						PreApply: []plancheck.PlanCheck{
							plancheck.ExpectResourceAction(githubBbdAddr, plancheck.ResourceActionUpdate),
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
		vars := newBbdVars()

		ApplyAndTest(t, resource.TestCase{
			Steps: []resource.TestStep{
				// Create draft v1 on manual, which accepts no policy at all.
				{Config: bbdStepConfig(t, 3), ConfigVariables: vars.variables()},
				// Release v1, which makes its version_spec immutable.
				{Config: bbdStepConfig(t, 3), ConfigVariables: vars.released().variables()},
				// Switch the implementation to terraform and enable an approval gate on the released version.
				// terraform supports the gate, so the plan-time validators pass and this reaches Update, where
				// the immutable version_spec is what must be reported.
				{
					Config: bbdStepConfig(t, 20),
					ConfigVariables: vars.released().withPolicies(
						bbdApprovalPolicies("manual_triggers"),
						nil,
					).variables(),
					ExpectError: regexp.MustCompile(`Updating a version_spec in non-draft \(released\) state is not allowed`),
				},
				// The rejected apply must not have written the approval gate. Re-planning the configuration
				// that was last applied refreshes from meshStack, so a stored policy change shows up here as a
				// non-empty plan.
				{Config: bbdStepConfig(t, 3), ConfigVariables: vars.released().variables(), PlanOnly: true},
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
		"display_name_template": knownvalue.StringExact("Example Building Block {{ resource_name }}"),
		"description":           knownvalue.StringExact(expectedDescription),
		"readme":                xknownvalue.NotEmptyString(),
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
		"display_name": knownvalue.StringFunc(func(v string) error {
			if !strings.HasPrefix(v, "Example Building Block") {
				return fmt.Errorf("expected %s to start with %s", v, "Example Building Block")
			}
			return nil
		}),
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
					"display_name":           knownvalue.StringExact("Business Unit"),
					"type":                   knownvalue.StringExact("CODE"),
					"assignment_type":        knownvalue.StringExact("TAG"),
					"is_environment":         knownvalue.Bool(false),
					"is_optional":            knownvalue.Bool(false),
					"updateable_by_consumer": knownvalue.Bool(false),
					"description":            knownvalue.StringExact("The business unit tag of the workspace this building block belongs to"),
					// The argument is the `<target>.<tagKey>` reference; the key carries a per-run random suffix.
					"argument":                       xknownvalue.NotEmptyString(),
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

func TestAccBuildingBlockDefinitionPolicyValidation(t *testing.T) {
	// The policy validators are client-side only, so the mock client is enough and a real backend adds
	// nothing - it would only reject the same configurations later, with its own wording.
	if !IsMockClientTest() {
		t.Skip("policy validation is tested with mock client only")
	}

	t.Parallel()

	policyConfig := func(implementation, approvalPolicies, schedule string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec = {
    display_name    = "Test"
    description     = "Test"
    approval_policies = %s
    schedule          = %s
  }
  version_spec = {
    draft = true
    implementation = %s
  }
}`, approvalPolicies, schedule, implementation)
	}

	const (
		manualImpl    = `{ manual = {} }`
		terraformImpl = `{ terraform = { terraform_version = "1.9.0", repository_url = "https://example.com/bb.git" } }`
	)

	tests := []struct {
		name             string
		implementation   string
		approvalPolicies string
		schedule         string
		expectError      *regexp.Regexp
	}{
		{
			name:             "approval policies and drift detection on terraform",
			implementation:   terraformImpl,
			approvalPolicies: `{ manual_triggers = true }`,
			schedule:         `{ mode = "DRIFT_DETECTION", frequency = "WEEKLY" }`,
		},
		{
			name:             "no policy at all on manual",
			implementation:   manualImpl,
			approvalPolicies: `{}`,
			schedule:         `{}`,
		},
		{
			name:             "approval gate on manual is rejected",
			implementation:   manualImpl,
			approvalPolicies: `{ manual_triggers = true }`,
			schedule:         `{}`,
			expectError:      regexp.MustCompile(`Approvals require a dry run`),
		},
		{
			name:             "drift detection on manual is rejected",
			implementation:   manualImpl,
			approvalPolicies: `{}`,
			schedule:         `{ mode = "DRIFT_DETECTION", frequency = "DAILY" }`,
			expectError:      regexp.MustCompile(`Scheduling is not supported for manual building blocks`),
		},
		{
			name:             "frequency without a schedule is rejected",
			implementation:   terraformImpl,
			approvalPolicies: `{}`,
			schedule:         `{ mode = "DISABLED", frequency = "DAILY" }`,
			expectError:      regexp.MustCompile(`Schedule frequency without a schedule`),
		},
		{
			name:             "schedule without a frequency is rejected",
			implementation:   terraformImpl,
			approvalPolicies: `{}`,
			schedule:         `{ mode = "DRIFT_DETECTION" }`,
			expectError:      regexp.MustCompile(`Schedule frequency required`),
		},
		{
			name:             "automatic approval without reconciliation is rejected",
			implementation:   terraformImpl,
			approvalPolicies: `{}`,
			schedule:         `{ mode = "DRIFT_DETECTION", frequency = "DAILY", automatic_approval = true }`,
			expectError:      regexp.MustCompile(`Automatic approval without drift reconciliation`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      policyConfig(test.implementation, test.approvalPolicies, test.schedule),
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

	// tagInputConfig wraps the spec attributes that decide which tags are readable, plus the attributes of
	// a single tag input, into a minimal valid BBD config.
	tagInputConfig := func(specAttributes, inputAttributes string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec = {
    display_name = "Test"
    description  = "Test"
    %s
  }
  version_spec = {
    draft          = true
    implementation = { terraform = { terraform_version = "1.9.0", repository_url = "https://example.com/bb.git" } }
    inputs = {
      cost_center = {
        display_name    = "Cost Center"
        assignment_type = "TAG"
        %s
      }
    }
  }
}`, specAttributes, inputAttributes)
	}

	const (
		workspaceLevel = `target_type = "WORKSPACE_LEVEL"`
		tenantLevel    = `target_type         = "TENANT_LEVEL"
    supported_platforms = [{ name = "AZURE" }]`
		codeType = `type = "CODE"`
	)

	tests := []struct {
		name            string
		specAttributes  string
		inputAttributes string
		expectError     *regexp.Regexp
	}{
		{
			name:            "a workspace tag on a workspace building block",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"WORKSPACE.costCenter\")",
		},
		{
			name:            "a project tag on a tenant building block",
			specAttributes:  tenantLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"PROJECT.costCenter\")",
		},
		{
			// Only the first separator splits the reference, so a tag key may contain a dot itself.
			name:            "a tag key containing a dot",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"WORKSPACE.cost.center\")",
		},
		{
			name:            "a project tag on a workspace building block is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"PROJECT.costCenter\")",
			expectError:     regexp.MustCompile(`A tag input cannot read this target`),
		},
		{
			name:            "a non-code input type is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: "type = \"STRING\"\n        argument = jsonencode(\"WORKSPACE.costCenter\")",
			expectError:     regexp.MustCompile(`A tag input must be a code input`),
		},
		{
			name:            "a sensitive tag input is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        sensitive = { argument = { secret_value = \"WORKSPACE.costCenter\" } }",
			expectError:     regexp.MustCompile(`A tag input cannot be sensitive`),
		},
		{
			name:            "a missing argument is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType,
			expectError:     regexp.MustCompile(`A tag input requires an argument naming the tag to read`),
		},
		{
			name:            "an argument without a tag key is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"WORKSPACE\")",
			expectError:     regexp.MustCompile(`A tag input argument must name a target and a tag key`),
		},
		{
			name:            "an unknown target is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode(\"TENANT.costCenter\")",
			expectError:     regexp.MustCompile(`A tag input argument must name a known target`),
		},
		{
			name:            "an argument that is not a string is rejected",
			specAttributes:  workspaceLevel,
			inputAttributes: codeType + "\n        argument = jsonencode([\"WORKSPACE.costCenter\"])",
			expectError:     regexp.MustCompile(`A tag input argument must be an encoded string`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      tagInputConfig(test.specAttributes, test.inputAttributes),
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

func TestAccBuildingBlockDefinitionSupportedPlatformKinds(t *testing.T) {
	vars := SuffixVariables(acctest.RandString(8))
	platformAddr := platformVariants[7].addr
	prerequisites := platformStepConfig(t, platformVariants[7])

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

	platformUuidExpr := platformAddr + ".metadata.uuid"
	platformTypeNameExpr := platformTypeResourceAddr + ".metadata.name"

	transitions := []struct {
		supportedPlatforms string
		expectStored       []knownvalue.Check
	}{
		{
			supportedPlatforms: fmt.Sprintf("[%s.ref]", platformAddr),
			expectStored:       []knownvalue.Check{platformRef},
		},
		{
			supportedPlatforms: fmt.Sprintf(`[{ name = "AZURE" }, { kind = "meshPlatform", uuid = %s }]`, platformUuidExpr),
			expectStored: []knownvalue.Check{
				platformTypeRef(knownvalue.StringExact("AZURE")),
				platformRef,
			},
		},
		{
			supportedPlatforms: fmt.Sprintf(`[{ name = %s }]`, platformTypeNameExpr),
			expectStored:       []knownvalue.Check{platformTypeRef(xknownvalue.NotEmptyString())},
		},
		{
			supportedPlatforms: fmt.Sprintf(`[{ kind = "meshPlatform", uuid = %s }]`, platformUuidExpr),
			expectStored:       []knownvalue.Check{platformRef},
		},
	}

	steps := make([]resource.TestStep, 0, len(transitions))
	for _, transition := range transitions {
		steps = append(steps, resource.TestStep{
			Config: examples.JoinTestStepConfigs(prerequisites, fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = %s }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = %s
  }
  version_spec = {
    draft = true
    implementation = { manual = {} }
  }
}`, workspaceResourceAddr+".metadata.name", transition.supportedPlatforms)),
			ConfigVariables: vars,
			ConfigStateChecks: []statecheck.StateCheck{
				capturePlatformUuid,
				statecheck.ExpectKnownValue(
					"meshstack_building_block_definition.test",
					tfjsonpath.New("spec").AtMapKey("supported_platforms"),
					knownvalue.SetExact(transition.expectStored),
				),
			},
		})
	}

	ApplyAndTest(t, resource.TestCase{Steps: steps})
}

func TestAccBuildingBlockDefinitionSupportedPlatformValidation(t *testing.T) {
	t.Parallel()

	const platformUuid = "3f1c0f2e-1a2b-4c3d-8e9f-0a1b2c3d4e5f"

	supportedPlatformsConfig := func(supportedPlatforms string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec = {
    display_name        = "Test"
    description         = "Test"
    target_type         = "TENANT_LEVEL"
    supported_platforms = %s
  }
  version_spec = {
    draft = true
    implementation = { manual = {} }
  }
}`, supportedPlatforms)
	}

	tests := []struct {
		name               string
		supportedPlatforms string
		expectError        *regexp.Regexp
	}{
		{
			name:               "an unknown kind is rejected",
			supportedPlatforms: fmt.Sprintf(`[{ kind = "meshPlatformInstance", uuid = %q }]`, platformUuid),
			expectError:        regexp.MustCompile(`value must be one of: \["meshPlatformType" "meshPlatform"\]`),
		},
		{
			name:               "a platform identified by name is rejected",
			supportedPlatforms: `[{ kind = "meshPlatform", name = "my-azure.eu-de" }]`,
			expectError:        regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			name:               "a platform type identified by uuid is rejected",
			supportedPlatforms: fmt.Sprintf(`[{ kind = "meshPlatformType", uuid = %q }]`, platformUuid),
			expectError:        regexp.MustCompile(`Invalid Attribute Combination`),
		},
		{
			name:               "an entry with both identifiers is rejected",
			supportedPlatforms: fmt.Sprintf(`[{ kind = "meshPlatform", name = "my-azure.eu-de", uuid = %q }]`, platformUuid),
			expectError:        regexp.MustCompile(`Invalid Attribute Combination`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ApplyAndTest(t, resource.TestCase{
				Steps: []resource.TestStep{{
					Config:      supportedPlatformsConfig(test.supportedPlatforms),
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

	const terraformImplementation = `{ terraform = { terraform_version = "1.9.0", repository_url = "https://github.com/example/bb.git" } }`
	const manualImplementation = `{ manual = {} }`

	config := func(implementation, input string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft          = true
    inputs         = { candidate = %s }
    implementation = %s
  }
}`, input, implementation)
	}

	tests := []struct {
		name           string
		implementation string
		input          string
		expectError    *regexp.Regexp
	}{
		{
			// A manual building block is carried out by a person, so there is no code to fall back to.
			name:           "optional input rejected on manual implementation",
			implementation: manualImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", is_optional = true }`,
			expectError:    regexp.MustCompile(`cannot be optional on a manual building block`),
		},
		{
			// Optionality is a decision of whoever fills the input in, so only their assignment types qualify.
			name:           "optional input rejected for a non-user assignment type",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "STATIC", argument = jsonencode("c"), is_optional = true }`,
			expectError:    regexp.MustCompile(`cannot be optional with this assignment_type`),
		},
		{
			name:           "optional BOOLEAN input rejected",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "BOOLEAN", assignment_type = "USER_INPUT", is_optional = true }`,
			expectError:    regexp.MustCompile(`type BOOLEAN cannot be optional`),
		},
		{
			name:           "optional input with default_value rejected",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", is_optional = true, default_value = jsonencode("c") }`,
			expectError:    regexp.MustCompile(`must not have a default value`),
		},
		{
			name:           "optional input with sensitive default_value rejected",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", is_optional = true, sensitive = { default_value = { secret_value = "c" } } }`,
			expectError:    regexp.MustCompile(`must not have a default value`),
		},
		{
			name:           "optional USER_INPUT accepted",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", is_optional = true }`,
		},
		{
			name:           "optional PLATFORM_OPERATOR_MANUAL_INPUT accepted",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT", is_optional = true }`,
		},
		{
			// None of the rules applies to an input that is not optional.
			name:           "non-optional BOOLEAN with default_value accepted",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "BOOLEAN", assignment_type = "USER_INPUT", default_value = jsonencode(true) }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: config(tt.implementation, tt.input)}
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

	inputConfig := func(input string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft = true
    inputs = {
      settings = { display_name = "Settings", %s }
    }
    implementation = { manual = {} }
  }
}`, input)
	}

	schema := `json_schema = jsonencode({ type = "object", properties = { region = { type = "string" } } })`

	tests := []struct {
		name        string
		input       string
		expectError *regexp.Regexp
	}{
		{
			name:  "JSON input with a schema accepted",
			input: `type = "JSON", assignment_type = "USER_INPUT", ` + schema,
		},
		{
			name:        "JSON input without a schema rejected",
			input:       `type = "JSON", assignment_type = "USER_INPUT"`,
			expectError: regexp.MustCompile(`json_schema is required`),
		},
		{
			name:        "schema on an input of another type rejected",
			input:       `type = "STRING", assignment_type = "USER_INPUT", ` + schema,
			expectError: regexp.MustCompile(`json_schema must not be set`),
		},
		{
			name:  "input of another type without a schema accepted",
			input: `type = "STRING", assignment_type = "USER_INPUT"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: inputConfig(tt.input)}
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

	const terraformImplementation = `{ terraform = { terraform_version = "1.9.0", repository_url = "https://github.com/example/bb.git" } }`
	const manualImplementation = `{ manual = {} }`

	config := func(implementation, input string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft          = true
    inputs         = {
      cloud_provider = { display_name = "Cloud", type = "STRING", assignment_type = "USER_INPUT" }
      candidate      = %s
    }
    implementation = %s
  }
}`, input, implementation)
	}

	tests := []struct {
		name           string
		implementation string
		input          string
		expectError    *regexp.Regexp
	}{
		{
			name:           "condition rejected on manual implementation",
			implementation: manualImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", condition = "input.cloud_provider == \"aws\"" }`,
			expectError:    regexp.MustCompile(`cannot have a condition on a manual building block`),
		},
		{
			name:           "condition rejected for a non-user assignment type",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "STATIC", argument = jsonencode("c"), condition = "input.cloud_provider == \"aws\"" }`,
			expectError:    regexp.MustCompile(`cannot have a condition with this assignment_type`),
		},
		{
			name:           "blank condition rejected",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", condition = "  " }`,
			expectError:    regexp.MustCompile(`must not be blank`),
		},
		{
			name:           "overlong condition rejected",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", condition = "input.cloud_provider == \"` + strings.Repeat("x", 600) + `\"" }`,
			expectError:    regexp.MustCompile(`string length must be at\s+most 512`),
		},
		{
			name:           "conditional USER_INPUT accepted",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "USER_INPUT", condition = "input.cloud_provider == \"aws\"" }`,
		},
		{
			name:           "conditional PLATFORM_OPERATOR_MANUAL_INPUT accepted",
			implementation: terraformImplementation,
			input:          `{ display_name = "Candidate", type = "STRING", assignment_type = "PLATFORM_OPERATOR_MANUAL_INPUT", condition = "input.cloud_provider == \"aws\"" }`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := resource.TestStep{Config: config(tt.implementation, tt.input)}
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
	manualConfig := func(outputs string) string {
		return fmt.Sprintf(`
resource "meshstack_building_block_definition" "test" {
  metadata = { owned_by_workspace = "my-workspace" }
  spec     = { display_name = "Test", description = "Test" }
  version_spec = {
    draft = true
    inputs = {
      tenant = { display_name = "Tenant", type = "STRING", assignment_type = "STATIC", argument = jsonencode("t") }
      region = { display_name = "Region", type = "STRING", assignment_type = "STATIC", argument = jsonencode("r") }
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
			// Outputs are keyed by input; a key with no matching input is rejected (mirrors the backend 400).
			name:        "output without a matching input rejected",
			outputs:     `outputs = { tenant = { assignment_type = "SUMMARY" }, surplus = { assignment_type = "SUMMARY" } }`,
			expectError: regexp.MustCompile(`no matching input`),
		},
		{
			// type is always derived for manual outputs, so declaring it is rejected.
			name:        "declared type rejected on manual output",
			outputs:     `outputs = { tenant = { type = "STRING", assignment_type = "SUMMARY" } }`,
			expectError: regexp.MustCompile(`type must not be set`),
		},
		{
			// A no-op override (NONE assignment and no display_name) cannot be reconstructed by the diff rule.
			name:        "no-op override rejected (NONE assignment, no display_name)",
			outputs:     `outputs = { tenant = { assignment_type = "NONE" } }`,
			expectError: regexp.MustCompile(`no effect`),
		},
		{
			name:        "no-op override rejected (empty object)",
			outputs:     `outputs = { tenant = {} }`,
			expectError: regexp.MustCompile(`no effect`),
		},
		{
			// display_order alone is not a supported override (deliberately not a membership signal).
			name:        "display_order-only override rejected",
			outputs:     `outputs = { tenant = { display_order = 5 } }`,
			expectError: regexp.MustCompile(`no effect`),
		},
		{
			// A display_name equal to the input's is not a real override.
			name:        "display_name equal to input rejected as no-op",
			outputs:     `outputs = { tenant = { display_name = "Tenant" } }`,
			expectError: regexp.MustCompile(`no effect`),
		},
		// Accepted: a sparse subset, an empty map (no overrides), and a display_name override.
		{
			name:    "empty outputs map accepted (no overrides)",
			outputs: `outputs = {}`,
		},
		{
			name:    "subset override via display_name accepted",
			outputs: `outputs = { tenant = { display_name = "Custom Tenant" } }`,
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
			name:    fmt.Sprintf("subset override with %s accepted", assignmentType),
			outputs: fmt.Sprintf(`outputs = { tenant = { assignment_type = %q } }`, assignmentType),
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

// withPreRunScript returns v with the terraform implementation's pre-run script changed, which is a
// version_spec change and therefore cuts a new draft.
func (v bbdVars) withPreRunScript(script string) bbdVars {
	v.preRunScript = script
	return v
}
