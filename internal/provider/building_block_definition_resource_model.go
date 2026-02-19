package provider

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/ptr"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/hash"
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

type supportedPlatformRef struct {
	Kind string `tfsdk:"kind"`
	Name string `tfsdk:"name"` // for kind meshPlatformType
}

type buildingBlockDefinitionVersionRef struct {
	Uuid        generic.NullIsUnknown[string]                                         `tfsdk:"uuid"`
	Number      generic.NullIsUnknown[int64]                                          `tfsdk:"number"`
	State       generic.NullIsUnknown[client.MeshBuildingBlockDefinitionVersionState] `tfsdk:"state"`
	ContentHash generic.NullIsUnknown[string]                                         `tfsdk:"content_hash"`
}

type buildingBlockDefinitionRef struct {
	Kind string `tfsdk:"kind"`
	Uuid string `tfsdk:"uuid"`
}

func newBuildingBlockDefinitionRef(uuid string) buildingBlockDefinitionRef {
	return buildingBlockDefinitionRef{
		Kind: "meshBuildingBlockDefinition",
		Uuid: uuid,
	}
}

// withSetEmptyContainersToNull transform all empty slices/maps as returned from API into null value (optional value),
// if the plan/state also specifies it as null (optional/omitted attribute).
// This accounts for somewhat unprecise handling in the backend for optional inputs, but let's take that shortcut
// and keep the config in-sync ignoring the bogus backend change.
func withSetEmptyContainersToNull(ctx context.Context, plan, state generic.AttributeGetter) generic.ConverterOption {
	return generic.WithValueFromEmptyContainer(func(attributePath path.Path) (haveNil bool, err error) {
		var attributeValue attr.Value
		var diags diag.Diagnostics
		if plan != nil {
			diags.Append(plan.GetAttribute(ctx, attributePath, &attributeValue)...)
		} else if state != nil {
			diags.Append(state.GetAttribute(ctx, attributePath, &attributeValue)...)
		}
		if diags.HasError() || attributeValue == nil {
			return true, fmt.Errorf("cannot get attribute from plan/state at %s: %s", attributePath, diags)
		}
		return attributeValue.IsNull(), nil
	})
}

func buildingBlockDefinitionConverterOptions(ctx context.Context, plan, state generic.AttributeGetter) generic.ConverterOptions {
	return generic.ConverterOptions{
		// Transform ref input in schema to simple string (aka the platform type).
		generic.WithValueFromConverterFor[client.BuildingBlockDefinitionSupportedPlatform](generic.ValueFromConverterForTypedNilHandler[supportedPlatformRef](),
			func(_ path.Path, value client.BuildingBlockDefinitionSupportedPlatform) (tftypes.Value, error) {
				return generic.ValueFrom(supportedPlatformRef{Kind: "meshPlatformType", Name: string(value)})
			},
		),
		generic.WithValueToConverterFor[client.BuildingBlockDefinitionSupportedPlatform](func(_ path.Path, in tftypes.Value) (client.BuildingBlockDefinitionSupportedPlatform, error) {
			// Handling this Ref (struct) to simple String Value could be extracted into re-usable converter I suppose (similar to schema_utils.go functions)
			ref, err := generic.ValueTo[supportedPlatformRef](in)
			if err != nil {
				return "", err
			}
			if ref.Kind != "meshPlatformType" {
				return "", fmt.Errorf("expected meshPlatformType for kind in given supported platform ref, got %s", ref.Kind)
			}
			return client.BuildingBlockDefinitionSupportedPlatform(ref.Name), nil
		}),
		generic.WithUseSetForElementsOf[client.BuildingBlockDefinitionSupportedPlatform](),

		withSetEmptyContainersToNull(ctx, plan, state),

		generic.WithUseSetForElementsOf[clientTypes.StringSetElem](),
	}
}

type buildingBlockDefinitionVersionSpec struct {
	Draft bool `tfsdk:"draft"`
	// inline all other properties, but also adapt the input type for internal secret/sensitive handling
	client.MeshBuildingBlockDefinitionVersionSpec
}

func (model buildingBlockDefinitionVersionSpec) ToClientDto(buildingBlockDefinitionUuid string) (dto client.MeshBuildingBlockDefinitionVersionSpec) {
	dto = model.MeshBuildingBlockDefinitionVersionSpec
	dto.BuildingBlockDefinitionRef = &client.BuildingBlockDefinitionRef{
		Kind: "meshBuildingBlockDefinition",
		Uuid: buildingBlockDefinitionUuid,
	}
	if dto.RunnerRef == nil {
		implementationType := dto.Implementation.InferTypeFromNonNilField()
		dto.RunnerRef = getSharedBuildingBlockRunnerRef(implementationType)
	}
	if model.Draft {
		dto.State = client.MeshBuildingBlockDefinitionVersionStateDraft.Ptr()
	} else {
		dto.State = client.MeshBuildingBlockDefinitionVersionStateReleased.Ptr()
	}
	if dto.VersionNumber == nil {
		dto.VersionNumber = ptr.To(int64(1))
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

	secretOrAnyValueFromConverter := generic.WithValueFromConverterFor[clientTypes.SecretOrAny](generic.ValueFromConverterForTypedNilHandler[string](),
		func(attributePath path.Path, in clientTypes.SecretOrAny) (tftypes.Value, error) {
			// Marshal any value to JSON to eventually match jsontypes.Normalized (if value is present)
			// Note that the case HasX is explicitly handled below!
			if in.HasY() {
				marshalled, err := json.Marshal(in.Y)
				if err != nil {
					return tftypes.Value{}, err
				}
				return generic.ValueFrom(string(marshalled))
			}
			return generic.ValueFrom[*string](nil)
		})

	return append(
		// Support converting secrets in implementation config
		secret.WithConverterSupport(ctx, config, plan, state),

		// Transform all empty slices as returned from API into null value (optional).
		// This is somewhat unprecise handling in the backend for optional lists, but let's take that shortcut.
		withSetEmptyContainersToNull(ctx, plan, state),

		generic.WithUseSetForElementsOf[client.ApiPermission](),

		// Handle implementation
		generic.WithValueFromConverterFor[client.MeshBuildingBlockDefinitionImplementation](nil, func(attributePath path.Path, in client.MeshBuildingBlockDefinitionImplementation) (tftypes.Value, error) {
			if in.Type == client.MeshBuildingBlockImplementationTypeManual {
				// manual, as a currently empty struct, is returned as nil by the backend (probably some "omitempty" optimization on Jackson's deserialization)
				// However, Terraform schema wants an empty struct here, to match the input config 'manual = {}'
				in.Manual = &client.MeshBuildingBlockDefinitionManualImplementation{}
			}
			return generic.ValueFrom(in, secret.WithConverterSupport(ctx, config, plan, state).Append(generic.WithAttributePath(attributePath))...)
		}),

		// Handle DependencyDefinitionUUIDs
		generic.WithValueToConverterFor[client.BuildingBlockDependencyRef](func(attributePath path.Path, in tftypes.Value) (client.BuildingBlockDependencyRef, error) {
			ref, err := generic.ValueTo[buildingBlockDefinitionRef](in)
			if err != nil {
				return "", err
			}
			return client.BuildingBlockDependencyRef(ref.Uuid), nil
		}),

		generic.WithValueFromConverterFor[client.BuildingBlockDependencyRef](generic.ValueFromConverterForTypedNilHandler[buildingBlockDefinitionRef](),
			func(attributePath path.Path, in client.BuildingBlockDependencyRef) (tftypes.Value, error) {
				return generic.ValueFrom(newBuildingBlockDefinitionRef(string(in)))
			}),

		// Handle version spec inputs: From Client DTO to model
		generic.WithValueFromConverterFor[client.MeshBuildingBlockDefinitionInput](
			func() (tftypes.Value, error) {
				return generic.ValueFrom[*buildingBlockDefinitionInputWithSensitive](nil,
					secretOrAnyValueFromConverter,
					// for selectable values
					generic.WithUseSetForElementsOf[clientTypes.StringSetElem](),
				)
			},
			func(attributePath path.Path, in client.MeshBuildingBlockDefinitionInput) (tftypes.Value, error) {
				// Note that client.MeshBuildingBlockDefinitionInput.UnmarshalJSON ensures that the IsSensitive flag is consistent with the clientTypes.SecretOrAny aka Variant[X, Y] state
				out := buildingBlockDefinitionInputWithSensitive{MeshBuildingBlockDefinitionInput: in}
				converterOptions := generic.ConverterOptions{
					generic.WithAttributePath(attributePath), // pass down attribute path into walk
					secretOrAnyValueFromConverter,
					// for selectable values
					generic.WithUseSetForElementsOf[clientTypes.StringSetElem](),
					withSetEmptyContainersToNull(ctx, plan, state),
				}
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
						sensitive.Argument = ptr.To(convertSecret(in.Argument.X, "argument"))
					}
					if in.DefaultValue.HasX() {
						sensitive.DefaultValue = ptr.To(convertSecret(in.DefaultValue.X, "default_value"))
					}
					if err := errors.Join(errs...); err != nil {
						return tftypes.Value{}, err
					}
					out.Sensitive = &sensitive
				}
				return generic.ValueFrom(out, converterOptions...)
			}),

		// Handle version spec inputs: From model to Client DTO
		generic.WithValueToConverterFor[client.MeshBuildingBlockDefinitionInput](func(attributePath path.Path, in tftypes.Value) (client.MeshBuildingBlockDefinitionInput, error) {
			model, err := generic.ValueTo[buildingBlockDefinitionInputWithSensitive](in,
				generic.WithValueToConverterFor[clientTypes.SecretOrAny](func(attributePath path.Path, in tftypes.Value) (out clientTypes.SecretOrAny, err error) {
					if in.IsKnown() && !in.IsNull() {
						var jsonValue string
						err = in.As(&jsonValue)
						if err != nil {
							return
						}
						err = json.Unmarshal([]byte(jsonValue), &out.Y)
					}
					return
				}),
				generic.WithSetUnknownValueToZero(),
			)
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
		sortAndUnique := func(s []clientTypes.StringSetElem) (result []clientTypes.StringSetElem) {
			result = slices.Clone(s)
			slices.Sort(result)
			result = slices.Compact(result)
			return
		}
		if slices.Compare(sortAndUnique(model.Spec.NotificationSubscribers), sortAndUnique(dto.Spec.NotificationSubscribers)) != 0 {
			missingSubscribers := slices.DeleteFunc(slices.Clone(model.Spec.NotificationSubscribers), func(subscriber clientTypes.StringSetElem) bool {
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
			ContentHash: generic.KnownValue(versionContentHash(versionDto.Spec, diags)),
		}
	}
	if diags.HasError() {
		return
	}
	latestIndex := len(versionDtos) - 1
	model.VersionSpec = buildingBlockDefinitionVersionSpec{
		MeshBuildingBlockDefinitionVersionSpec: versionDtos[latestIndex].Spec,
	}

	if !isDraft.IsUnknown() && !isDraft.Get() {
		// draft=false from config/plan, let's always keep this desired value!
		model.VersionSpec.Draft = false
		if *model.VersionSpec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap() {
			// show a warning though if that doesn't fit the state (aka state is still DRAFT), as the version might be "in review"
			// the publication status IN_REVIEW is currently not available in the meshObject API, and it would only make sense to have some auto_approve flag next to draft
			// (provided the API token has sufficient permissions to approve this)
			diags.AddWarning("Building Block Definition release needs admin approval", fmt.Sprintf(
				"The latest version of the Building Block Definition %s needs admin approval before its state can change to %s as desired by your current Terraform configuration draft=false. "+
					"Approve the release in the Admin panel and re-run Terraform, which makes this warning disappear.", bbdUuid, client.MeshBuildingBlockDefinitionVersionStateReleased))
		}
	} else {
		model.VersionSpec.Draft = *model.VersionSpec.State == client.MeshBuildingBlockDefinitionVersionStateDraft.Unwrap()
	}
	model.VersionLatest = model.Versions[latestIndex]

	if model.VersionLatest.State.Get() == client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap() {
		model.VersionLatestRelease = ptr.To(model.VersionLatest)
	} else if len(model.Versions) > 1 {
		if latestButOneVersion := model.Versions[latestIndex-1]; latestButOneVersion.State.Get() == client.MeshBuildingBlockDefinitionVersionStateReleased.Unwrap() {
			model.VersionLatestRelease = ptr.To(latestButOneVersion)
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

func versionContentHash(versionSpecDto client.MeshBuildingBlockDefinitionVersionSpec, diags *diag.Diagnostics) string {
	if result, err := func() (string, error) {
		// Ignore version, state, and buildingBlockDefinitionRef fields by setting them to constant values, always!
		versionSpecDto.VersionNumber = nil
		versionSpecDto.State = nil
		versionSpecDto.BuildingBlockDefinitionRef = nil

		// Converting it first from/to JSON makes hashing more stable, as fields with 'omitempty' are ignored.
		// Additionally, all numbers are converted to float64, even integers (which also allows changing DTO model types later on).
		// Also, the current Hasher implementation does not support structs for now, but map[string]any works!
		var buffer bytes.Buffer
		if err := json.NewEncoder(&buffer).Encode(versionSpecDto); err != nil {
			return "", err
		}
		var converted any
		if err := json.NewDecoder(&buffer).Decode(&converted); err != nil {
			return "", err
		}

		versionSpecHash, err := hash.Hasher{}.Hash(converted,
			// Safeguard against accidentally hashing plaintext values
			// (should never happen as backend never returns plaintext values)
			hash.DisallowMapKeys("plaintext", "buildingBlockDefinitionRef"),
		)
		if err != nil {
			return "", err
		}
		// add some versioning prefix to migrate possible changes in the hashes later on,
		// but let's hope migrating/fixing the hashes is never required
		return "v1:" + versionSpecHash.Hex(), nil
	}(); err != nil {
		diags.AddError("Failed to determine content hash", fmt.Sprintf(
			"Content hashing of version_spec as client DTO failed: %s", err.Error(),
		))
		return ""
	} else {
		return result
	}
}
