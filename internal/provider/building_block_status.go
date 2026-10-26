package provider

import (
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

const buildingBlockStatusDeprecation = "Use `succeeded`, or `health_status` and `action` for finer checks."

var (
	buildingBlockSucceededDescription = "True when the latest run succeeded and the building block needs no attention: " +
		"`health_status` is `SUCCEEDED` and `action` is one of `NO_ACTION`, `UPGRADE_AVAILABLE`, `REVIEW_DRIFT`. " +
		"On meshStack versions before the `REVIEW_ABORTED_RUN` action an aborted run reads as succeeded."
	buildingBlockHealthStatusDescription = "Outcome of the latest run that speaks for the building block. One of " +
		client.BuildingBlockHealthStatuses.Markdown() + ". Only a run that finished as succeeded or failed moves it: it keeps " +
		"the previous outcome while a run is pending, in progress, waiting for input or approval, or after a run was aborted, " +
		"and it stays `NONE` until the first run finished. Read `action` to tell these apart."
	buildingBlockActionDescription = "What the building block needs next, as in the Action column of meshPanel. One of " +
		client.BuildingBlockActions.Markdown() + ". `PROVIDE_INPUT` covers user and operator input, `REVIEW_DRIFT` reports a " +
		"dry run that detected changes, `REVIEW_ABORTED_RUN` an aborted or rejected latest run, and " +
		"`UPGRADE_AVAILABLE` a building block that needs nothing else while a newer definition version exists. " +
		"Later meshStack releases may add values, which `succeeded` reads as not succeeded."
)

// buildingBlockStatusModel adds the provider-only succeeded to the status meshStack reports.
type buildingBlockStatusModel struct {
	client.MeshBuildingBlockV2Status
	Succeeded bool `tfsdk:"succeeded"`
}

func newBuildingBlockStatusModel(status client.MeshBuildingBlockV2Status) buildingBlockStatusModel {
	return buildingBlockStatusModel{
		MeshBuildingBlockV2Status: status,
		Succeeded: status.HealthStatus == client.BuildingBlockHealthStatusSucceeded && slices.Contains(
			[]client.BuildingBlockAction{client.BuildingBlockActionNoAction.Unwrap(), client.BuildingBlockActionUpgradeAvailable.Unwrap(), client.BuildingBlockActionReviewDrift.Unwrap()},
			status.Action.Unwrap(),
		),
	}
}

var buildingBlockStatusAnyConverters = []generic.ConverterOption{withValueFromConverterForClientTypeAny(), withValueToConverterForClientTypeAny()}

func withBuildingBlockStatusValueFromConverter() generic.ConverterOption {
	return generic.WithValueFromConverterFor[client.MeshBuildingBlockV2Status](
		func() (tftypes.Value, error) {
			return generic.ValueFrom[*buildingBlockStatusModel](nil, buildingBlockStatusAnyConverters...)
		},
		func(attributePath path.Path, in client.MeshBuildingBlockV2Status) (tftypes.Value, error) {
			return generic.ValueFrom(newBuildingBlockStatusModel(in), append(buildingBlockStatusAnyConverters, generic.WithAttributePath(attributePath))...)
		})
}

func withBuildingBlockStatusValueToConverter() generic.ConverterOption {
	return generic.WithValueToConverterFor[client.MeshBuildingBlockV2Status](func(_ path.Path, in tftypes.Value) (client.MeshBuildingBlockV2Status, error) {
		model, err := generic.ValueTo[buildingBlockStatusModel](in, append(buildingBlockStatusAnyConverters, generic.WithSetUnknownValueToZero())...)
		return model.MeshBuildingBlockV2Status, err
	})
}
