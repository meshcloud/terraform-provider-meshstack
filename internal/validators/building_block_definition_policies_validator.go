package validators

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

// meshStack validates a building block definition's approvals and drift schedule against the
// implementation type of its latest version, and rejects a combination that the implementation type does not support.
// The two validators below repeat those rules at plan time, so a configuration that can never
// apply fails before anything is written: the resource writes the definition, then the version, then
// the policies, so a backend rejection would otherwise arrive after two requests already landed.
//
// Every check is skipped while a value it needs is still unknown, which makes the validators
// best-effort - the rules hold for fully known configuration and meshStack enforces the rest.
var (
	_ validator.Object = BuildingBlockDefinitionApprovals{}
	_ validator.Object = BuildingBlockDefinitionSchedule{}
)

// implementationTypeOf reads which implementation the configured version_spec selects. It returns an
// empty string while the implementation block, or the choice within it, is not known yet.
func implementationTypeOf(ctx context.Context, config tfsdk.Config, diagnostics *diag.Diagnostics) client.MeshBuildingBlockImplementationType {
	var implementation types.Object
	diagnostics.Append(config.GetAttribute(ctx, path.Root("version_spec").AtName("implementation"), &implementation)...)
	if diagnostics.HasError() || implementation.IsNull() || implementation.IsUnknown() {
		return ""
	}
	// The attribute names mirror client.MeshBuildingBlockDefinitionImplementation's tfsdk tags; exactly
	// one is set, which ExactlyOneAttributeValidator on the implementation block already enforces.
	byAttributeName := map[string]client.MeshBuildingBlockImplementationType{
		"manual":                client.MeshBuildingBlockImplementationTypeManual.Unwrap(),
		"terraform":             client.MeshBuildingBlockImplementationTypeTerraform.Unwrap(),
		"github_workflows":      client.MeshBuildingBlockImplementationTypeGithubWorkflows.Unwrap(),
		"gitlab_pipeline":       client.MeshBuildingBlockImplementationTypeGitlabPipeline.Unwrap(),
		"azure_devops_pipeline": client.MeshBuildingBlockImplementationTypeAzureDevOpsPipeline.Unwrap(),
	}
	for attributeName, value := range implementation.Attributes() {
		if !value.IsNull() && !value.IsUnknown() {
			return byAttributeName[attributeName]
		}
	}
	return ""
}

// BuildingBlockDefinitionApprovals rejects an enabled approval gate on an implementation without dry
// runs. Approving a run means reviewing what it would change, and only a dry run can show that.
type BuildingBlockDefinitionApprovals struct{}

func (v BuildingBlockDefinitionApprovals) Description(_ context.Context) string {
	return fmt.Sprintf("Ensures no approval gate is enabled unless the version's implementation is %s", client.MeshBuildingBlockImplementationTypeTerraform)
}

func (v BuildingBlockDefinitionApprovals) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BuildingBlockDefinitionApprovals) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	var enabledApprovalGates []string
	for name, value := range req.ConfigValue.Attributes() {
		if gate, ok := value.(types.Bool); ok && !gate.IsUnknown() && gate.ValueBool() {
			enabledApprovalGates = append(enabledApprovalGates, name)
		}
	}
	if len(enabledApprovalGates) == 0 {
		return
	}
	slices.Sort(enabledApprovalGates)

	terraform := client.MeshBuildingBlockImplementationTypeTerraform.Unwrap()
	implementationType := implementationTypeOf(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || implementationType == "" || implementationType == terraform {
		return
	}

	resp.Diagnostics.AddAttributeError(req.Path,
		"Approvals require a dry run",
		fmt.Sprintf("The %s approval gate is enabled, but the version's implementation is %s, which has no dry run for an approver to review. "+
			"Only the %s implementation supports approvals.",
			strings.Join(enabledApprovalGates, " and "), implementationType, terraform),
	)
}

// BuildingBlockDefinitionSchedule rejects an internally inconsistent drift schedule, and one
// the version's implementation cannot run.
type BuildingBlockDefinitionSchedule struct{}

func (v BuildingBlockDefinitionSchedule) Description(_ context.Context) string {
	return "Ensures the drift schedule is self-consistent and supported by the version's implementation"
}

func (v BuildingBlockDefinitionSchedule) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v BuildingBlockDefinitionSchedule) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	// An omitted attribute is null here, and the schema default is applied after validation, so each
	// value below falls back to that default rather than skipping the check.
	attributes := req.ConfigValue.Attributes()
	mode, modeKnown := configuredString(attributes["mode"], client.MeshBuildingBlockScheduleModeDisabled.String())
	frequency, frequencyKnown := configuredString(attributes["frequency"], client.MeshBuildingBlockScheduleFrequencyNone.String())
	automaticApproval, automaticApprovalKnown := configuredBool(attributes["automatic_approval"], false)
	if !modeKnown {
		return
	}

	var (
		disabled            = client.MeshBuildingBlockScheduleModeDisabled.String()
		driftDetection      = client.MeshBuildingBlockScheduleModeDriftDetection.String()
		driftReconciliation = client.MeshBuildingBlockScheduleModeDriftReconciliation.String()
		noFrequency         = client.MeshBuildingBlockScheduleFrequencyNone.String()
	)

	if frequencyKnown {
		switch {
		case mode == disabled && frequency != noFrequency:
			resp.Diagnostics.AddAttributeError(req.Path.AtName("frequency"),
				"Schedule frequency without a schedule",
				fmt.Sprintf("A frequency only applies when mode is %s or %s, but mode is %s. Set frequency to %s.",
					driftDetection, driftReconciliation, disabled, noFrequency))
		case mode != disabled && frequency == noFrequency:
			resp.Diagnostics.AddAttributeError(req.Path.AtName("frequency"),
				"Schedule frequency required",
				fmt.Sprintf("Mode %s runs on a schedule, so frequency must not be %s.", mode, noFrequency))
		}
	}

	if automaticApprovalKnown && automaticApproval && mode != driftReconciliation {
		resp.Diagnostics.AddAttributeError(req.Path.AtName("automatic_approval"),
			"Automatic approval without drift reconciliation",
			fmt.Sprintf("Automatic approval applies to a reconciliation run, so it can only be true when mode is %s, but mode is %s.",
				driftReconciliation, mode))
	}

	if mode == disabled {
		return
	}

	terraform := client.MeshBuildingBlockImplementationTypeTerraform.Unwrap()
	implementationType := implementationTypeOf(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || implementationType == "" {
		return
	}

	switch {
	case implementationType == client.MeshBuildingBlockImplementationTypeManual.Unwrap():
		resp.Diagnostics.AddAttributeError(req.Path.AtName("mode"),
			"Scheduling is not supported for manual building blocks",
			fmt.Sprintf("An operator applies a manual building block by hand, so meshStack cannot detect or fix drift on a schedule. Set mode to %s.",
				disabled))
	case mode == driftDetection && implementationType != terraform:
		resp.Diagnostics.AddAttributeError(req.Path.AtName("mode"),
			"Drift detection requires a dry run",
			fmt.Sprintf("Mode %s reports drift without fixing it. That needs a dry run, which only the %s implementation has, "+
				"but the version's implementation is %s. Use mode %s with automatic_approval set to true instead.",
				driftDetection, terraform, implementationType, driftReconciliation))
	case mode == driftReconciliation && implementationType != terraform && automaticApprovalKnown && !automaticApproval:
		resp.Diagnostics.AddAttributeError(req.Path.AtName("automatic_approval"),
			"Drift reconciliation without automatic approval requires a dry run",
			fmt.Sprintf("An approver reviews the dry run of a reconciliation run, which only the %s implementation can produce, but the version's implementation is %s. "+
				"Set automatic_approval to true.", terraform, implementationType))
	}
}

// configuredString reports the attribute's configured value, substituting whenNull for a null one.
// The second result is false only for an unknown value, which no check can be made against.
func configuredString(value attr.Value, whenNull string) (string, bool) {
	typed, ok := value.(types.String)
	if !ok || typed.IsUnknown() {
		return "", false
	}
	if typed.IsNull() {
		return whenNull, true
	}
	return typed.ValueString(), true
}

// configuredBool is configuredString for a bool attribute.
func configuredBool(value attr.Value, whenNull bool) (bool, bool) {
	typed, ok := value.(types.Bool)
	if !ok || typed.IsUnknown() {
		return false, false
	}
	if typed.IsNull() {
		return whenNull, true
	}
	return typed.ValueBool(), true
}
