package provider

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

type buildingBlockDefinition struct {
	Metadata    client.MeshBuildingBlockDefinitionMetadata `tfsdk:"metadata"`
	Spec        client.MeshBuildingBlockDefinitionSpec     `tfsdk:"spec"`
	VersionSpec buildingBlockDefinitionVersionSpec         `tfsdk:"version_spec"`

	Versions             []buildingBlockDefinitionVersionRef `tfsdk:"versions"`
	VersionLatest        buildingBlockDefinitionVersionRef   `tfsdk:"version_latest"`
	VersionLatestRelease *buildingBlockDefinitionVersionRef  `tfsdk:"version_latest_release"`

	Ref buildingBlockDefinitionRef `tfsdk:"ref"`
}

type buildingBlockDefinitionVersionRef struct {
	Uuid        generic.NullIsUnknown[string]                                         `tfsdk:"uuid"`
	Number      generic.NullIsUnknown[int64]                                          `tfsdk:"number"`
	State       generic.NullIsUnknown[client.MeshBuildingBlockDefinitionVersionState] `tfsdk:"state"`
	ContentHash generic.NullIsUnknown[string]                                         `tfsdk:"content_hash"`
	Kind        generic.NullIsUnknown[string]                                         `tfsdk:"kind"`
}

type buildingBlockDefinitionRef struct {
	Kind string `tfsdk:"kind"`
	Uuid string `tfsdk:"uuid"`
}

func newBuildingBlockDefinitionRef(uuid string) buildingBlockDefinitionRef {
	return buildingBlockDefinitionRef{
		Kind: client.MeshObjectKind.BuildingBlockDefinition,
		Uuid: uuid,
	}
}

func buildingBlockDefinitionConverterOptions() generic.ConverterOptions {
	return generic.ConverterOptions{
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
	}
}

type buildingBlockDefinitionVersionSpec struct {
	Draft bool `tfsdk:"draft"`
	// inline all other properties, but also adapt the input type for internal secret/sensitive handling
	client.MeshBuildingBlockDefinitionVersionSpec
}

func (model buildingBlockDefinitionVersionSpec) ToClientDto(buildingBlockDefinitionUuid string) (dto client.MeshBuildingBlockDefinitionVersionSpec) {
	dto = model.MeshBuildingBlockDefinitionVersionSpec
	dto.BuildingBlockDefinitionRef = &client.UuidRef{
		Kind: client.MeshObjectKind.BuildingBlockDefinition,
		Uuid: buildingBlockDefinitionUuid,
	}
	if dto.RunnerRef == nil {
		dto.RunnerRef = &client.UuidRef{
			Kind: client.MeshObjectKind.BuildingBlockRunner,
			Uuid: SharedBuildingBlockRunnerUuid,
		}
	}
	if model.Draft {
		dto.State = client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr()
	} else {
		dto.State = client.MeshBuildingBlockDefinitionVersionStateReleased.Ptr()
	}
	if dto.VersionNumber == nil {
		dto.VersionNumber = new(int64(1))
	}

	// Backend doesn't accept null inputs/outputs, so work around this
	if dto.Inputs == nil {
		dto.Inputs = make(map[string]*client.MeshBuildingBlockDefinitionInput)
	}
	if dto.Outputs == nil {
		dto.Outputs = make(map[string]client.MeshBuildingBlockDefinitionOutput)
	}

	return
}

func buildingBlockDefinitionVersionConverterOptions(ctx context.Context, config, plan, state generic.AttributeGetter) generic.ConverterOptions {
	type buildingBlockDefinitionVersionInputSensitive struct {
		Argument     *secret.Secret `tfsdk:"argument"`
		DefaultValue *secret.Secret `tfsdk:"default_value"`
	}

	type buildingBlockDefinitionInputWithSensitive struct {
		client.MeshBuildingBlockDefinitionInput
		Sensitive *buildingBlockDefinitionVersionInputSensitive `tfsdk:"sensitive"`
	}

	return append(
		// Support converting secrets in implementation config
		secret.WithConverterSupport(ctx, config, plan, state),

		generic.WithSliceTypeAsSet(clientTypes.IsSet),

		// Handle implementation
		generic.WithValueFromConverterFor[client.MeshBuildingBlockDefinitionImplementation](nil, func(attributePath path.Path, in client.MeshBuildingBlockDefinitionImplementation) (tftypes.Value, error) {
			if in.Type == client.MeshBuildingBlockImplementationTypeManual {
				// manual, as a currently empty struct, is returned as nil by the backend (probably some "omitempty" optimization on Jackson's deserialization)
				// However, Terraform schema wants an empty struct here, to match the input config 'manual = {}'
				in.Manual = &client.MeshBuildingBlockDefinitionManualImplementation{}
			}
			return generic.ValueFrom(in, secret.WithConverterSupport(ctx, config, plan, state).Append(generic.WithAttributePath(attributePath))...)
		}),

		// dependency_refs (types.Set[client.UuidRef]) needs no custom converter — mapped generically like runner_ref.

		// Handle version spec inputs: From Client DTO to model
		generic.WithValueFromConverterFor[client.MeshBuildingBlockDefinitionInput](
			func() (tftypes.Value, error) {
				return generic.ValueFrom[*buildingBlockDefinitionInputWithSensitive](nil,
					secretOrAnyValueFromConverter,
					// for selectable values
					generic.WithSliceTypeAsSet(clientTypes.IsSet),
				)
			},
			func(attributePath path.Path, in client.MeshBuildingBlockDefinitionInput) (tftypes.Value, error) {
				// Note that client.MeshBuildingBlockDefinitionInput.UnmarshalJSON ensures that the IsSensitive flag is consistent with the clientTypes.SecretOrAny aka Variant[X, Y] state
				out := buildingBlockDefinitionInputWithSensitive{MeshBuildingBlockDefinitionInput: in}
				if in.IsSensitive {
					var errs []error
					convertSecret := func(in clientTypes.Secret, attributeName string) (out secret.Secret) {
						secretValue, err := secret.ValueFromConverter(ctx, plan, state, attributePath.AtName("sensitive").AtName(attributeName), in)
						if err != nil {
							errs = append(errs, err)
							return
						}
						out, err = generic.ValueTo[secret.Secret](secretValue)
						if err != nil {
							errs = append(errs, err)
						}
						return
					}
					sensitive := buildingBlockDefinitionVersionInputSensitive{}
					if in.Argument.HasX() {
						sensitive.Argument = new(convertSecret(in.Argument.X, "argument"))
					}
					if in.DefaultValue.HasX() {
						sensitive.DefaultValue = new(convertSecret(in.DefaultValue.X, "default_value"))
					}
					if err := errors.Join(errs...); err != nil {
						return tftypes.Value{}, err
					}
					out.Sensitive = &sensitive
				}
				return generic.ValueFrom(out,
					generic.WithAttributePath(attributePath),
					secretOrAnyValueFromConverter,
					generic.WithSliceTypeAsSet(clientTypes.IsSet), // selectable values are sets
				)
			}),

		// Handle version spec inputs: From model to Client DTO
		generic.WithValueToConverterFor[client.MeshBuildingBlockDefinitionInput](func(attributePath path.Path, in tftypes.Value) (client.MeshBuildingBlockDefinitionInput, error) {
			model, err := generic.ValueTo[buildingBlockDefinitionInputWithSensitive](in, secretOrAnyValueToConverter, generic.WithSetUnknownValueToZero())
			if err != nil {
				return client.MeshBuildingBlockDefinitionInput{}, err
			}
			if model.Sensitive != nil {
				model.IsSensitive = true
				var errs []error
				convertSecret := func(_ secret.Secret, attributeName string) (out clientTypes.Secret) {
					out, err = secret.ValueToConverter(ctx, config, plan, state, attributePath.AtName("sensitive").AtName(attributeName))
					if err != nil {
						errs = append(errs, err)
					}
					return
				}
				if model.Sensitive.Argument != nil {
					model.Argument.X = convertSecret(*model.Sensitive.Argument, "argument")
				}
				if model.Sensitive.DefaultValue != nil {
					model.DefaultValue.X = convertSecret(*model.Sensitive.DefaultValue, "default_value")
				}
				if err := errors.Join(errs...); err != nil {
					return client.MeshBuildingBlockDefinitionInput{}, err
				}
			}
			return model.MeshBuildingBlockDefinitionInput, nil
		}),
	)
}

func (model *buildingBlockDefinition) SetFromClientDto(dto *client.MeshBuildingBlockDefinition, diags *diag.Diagnostics) {
	model.Metadata = dto.Metadata

	if len(model.Spec.NotificationSubscribers) > 0 {
		sortAndUnique := func(s []string) (result []string) {
			result = slices.Clone(s)
			slices.Sort(result)
			result = slices.Compact(result)
			return
		}
		if slices.Compare(sortAndUnique(model.Spec.NotificationSubscribers), sortAndUnique(dto.Spec.NotificationSubscribers)) != 0 {
			missingSubscribers := slices.DeleteFunc(slices.Clone(model.Spec.NotificationSubscribers), func(subscriber string) bool {
				return slices.Contains(dto.Spec.NotificationSubscribers, subscriber)
			})
			diags.AddWarning("Notification Subscribers modified", fmt.Sprintf(
				"The backend only accepted the following notification subscribers: %s. The following are missing: %s. Please adapt your configuration for Building Block Definition %s.",
				dto.Spec.NotificationSubscribers, missingSubscribers, *model.Metadata.Uuid,
			))
			// Ignore response from backend to avoid config vs. state drift and ugly Terraform error complaining about this
			// The next read will show a state drift again.
			dto.Spec.NotificationSubscribers = model.Spec.NotificationSubscribers
		}
	}
	model.Spec = dto.Spec
}

func (model *buildingBlockDefinition) SetFromVersionClientDtos(diags *diag.Diagnostics, isDraft generic.NullIsUnknown[bool], bbdUuid string, versionDtos ...client.MeshBuildingBlockDefinitionVersion) {
	if len(versionDtos) == 0 {
		diags.AddError("Building Block Definition without versions found",
			"This should never happen for a properly created building block definition. "+
				"The API shows unexpected behavior.")
		return
	}

	// sort by ascending version number (newest version is last, oldest version first)
	slices.SortFunc(versionDtos, func(a, b client.MeshBuildingBlockDefinitionVersion) int {
		return cmp.Compare(*a.Spec.VersionNumber, *b.Spec.VersionNumber)
	})

	model.Versions = make([]buildingBlockDefinitionVersionRef, len(versionDtos))
	for i, versionDto := range versionDtos {
		model.Versions[i] = buildingBlockDefinitionVersionRef{
			Uuid:        generic.KnownValue(versionDto.Metadata.Uuid),
			Number:      generic.KnownValue(*versionDto.Spec.VersionNumber),
			State:       generic.KnownValue(*versionDto.Spec.State),
			ContentHash: generic.KnownValue(calculateBuildingBlockDefinitionVersionContentHash(versionDto.Spec, diags).toBase64()),
			Kind:        generic.KnownValue(client.MeshObjectKind.BuildingBlockDefinitionVersion),
		}
	}
	if diags.HasError() {
		return
	}
	latestIndex := len(versionDtos) - 1
	inputsNil := model.VersionSpec.Inputs == nil
	model.VersionSpec = buildingBlockDefinitionVersionSpec{
		MeshBuildingBlockDefinitionVersionSpec: versionDtos[latestIndex].Spec,
	}
	if inputsNil && len(model.VersionSpec.Inputs) == 0 {
		model.VersionSpec.Inputs = nil
	}

	if !isDraft.IsUnknown() && !isDraft.Get() {
		// draft=false from config/plan, let's always keep this desired value!
		model.VersionSpec.Draft = false
		if *model.VersionSpec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap() {
			// show a warning though if that doesn't fit the state (aka state is still DRAFT), as the version might be "in review"
			// the publication status IN_REVIEW is currently not available in the meshObject API.
			// It would only make sense to have some auto_approve flag next to draft,
			// provided the API token has sufficient permissions to approve it.
			diags.AddWarning("Building Block Definition release needs admin approval", fmt.Sprintf(
				"The latest version of the Building Block Definition %s needs admin approval before its state can change to %s as desired by your current Terraform configuration draft=false. "+
					"Approve the release in the Admin panel and re-run Terraform, which makes this warning disappear.", bbdUuid, client.MeshBuildingBlockDefinitionVersionStateReleased))
		}
	} else {
		model.VersionSpec.Draft = *model.VersionSpec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()
	}
	model.VersionLatest = model.Versions[latestIndex]

	if model.VersionLatest.State.Get() == client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap() {
		model.VersionLatestRelease = new(model.VersionLatest)
	} else if len(model.Versions) > 1 {
		if latestButOneVersion := model.Versions[latestIndex-1]; latestButOneVersion.State.Get() == client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap() {
			model.VersionLatestRelease = new(latestButOneVersion)
		} else {
			diags.AddError("version_latest_release points to non-release version", fmt.Sprintf(
				"The attribute version_latest_release has state %s and not expected %s. The API shows unexpected behavior.",
				latestButOneVersion.State.Get(), client.MeshBuildingBlockDefinitionVersionStateReleased,
			))
		}
	} else {
		model.VersionLatestRelease = nil
	}

	model.Ref = newBuildingBlockDefinitionRef(bbdUuid)
}

// versionSpecContainsPlaintextSecret reports whether the typed version_spec DTO carries a secret whose
// plaintext value is set, i.e. a secret is being created or rotated. It walks the typed DTO and only
// inspects clientTypes.Secret values, so arbitrary user JSON (which lands in the variant's "any" branch,
// never in a Secret) is never mistaken for a secret. The backend only ever returns hashes, so a true
// result always stems from a locally-planned value.
func versionSpecContainsPlaintextSecret(versionSpecDto client.MeshBuildingBlockDefinitionVersionSpec) bool {
	found := false
	_ = reflectwalk.Walk(reflect.ValueOf(versionSpecDto), func(_ reflectwalk.WalkPath, v reflect.Value) error {
		if s, ok := v.Interface().(clientTypes.Secret); ok && s.Plaintext != nil {
			found = true
		}
		return nil
	})
	return found
}
