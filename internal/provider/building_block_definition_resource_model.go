package provider

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
	"github.com/meshcloud/terraform-provider-meshstack/internal/util/hash"
)

type buildingBlockDefinition struct {
	Metadata    buildingBlockDefinitionMetadata        `tfsdk:"metadata"`
	Spec        client.MeshBuildingBlockDefinitionSpec `tfsdk:"spec"`
	VersionSpec buildingBlockDefinitionVersionSpec     `tfsdk:"version_spec"`

	Versions             types.List   `tfsdk:"versions"`
	VersionLatest        types.Object `tfsdk:"version_latest"`
	VersionLatestRelease types.Object `tfsdk:"version_latest_release"`
}

type buildingBlockDefinitionMetadata struct {
	client.MeshBuildingBlockDefinitionMetadataAdapter[generic.Value[clientTypes.String]]
}

func (model buildingBlockDefinitionMetadata) ToClientDto(diags *diag.Diagnostics) client.MeshBuildingBlockDefinitionMetadata {
	return client.MeshBuildingBlockDefinitionMetadata{
		MeshBuildingBlockDefinitionMetadataBase: model.MeshBuildingBlockDefinitionMetadataBase,

		Uuid:                model.Uuid.GetPtr(diags),
		CreatedOn:           model.CreatedOn.GetPtr(diags),
		MarkedForDeletionOn: model.MarkedForDeletionOn.GetPtr(diags),
		MarkedForDeletionBy: model.MarkedForDeletionBy.GetPtr(diags),
	}
}

//goland:noinspection GoMixedReceiverTypes
func (model *buildingBlockDefinitionMetadata) SetFromClientDto(dto client.MeshBuildingBlockDefinitionMetadata, diags *diag.Diagnostics) {
	model.MeshBuildingBlockDefinitionMetadataBase = dto.MeshBuildingBlockDefinitionMetadataBase
	model.Uuid.SetRequired(dto.Uuid, diags)
	model.CreatedOn.SetRequired(dto.CreatedOn, diags)
	model.MarkedForDeletionOn.SetOptional(dto.MarkedForDeletionOn, diags)
	model.MarkedForDeletionBy.SetOptional(dto.MarkedForDeletionBy, diags)
}

type buildingBlockDefinitionVersionSpec struct {
	Draft bool `tfsdk:"draft"`
	// inline all other properties, but also adapt the input type for internal secret/sensitive handling
	client.MeshBuildingBlockDefinitionVersionSpecAdapter[
		*buildingBlockDefinitionVersionInput,
		secret.Secret,
		generic.Value[client.MeshBuildingBlockDefinitionVersionState],
		generic.Value[clientTypes.Number],
	]
}

//nolint:unused
type buildingBlockDefinitionVersionInput struct {
	client.MeshBuildingBlockDefinitionInputAdapter[generic.Value[any]]
	Sensitive *buildingBlockDefinitionVersionInputSensitive `tfsdk:"sensitive"`
}

type buildingBlockDefinitionVersionInputSensitive struct {
	Argument     secret.Secret `tfsdk:"argument"`
	DefaultValue secret.Secret `tfsdk:"default_value"`
}

type buildingBlockDefinitionVersionRef struct {
	Uuid        string                                         `tfsdk:"uuid"`
	Number      int64                                          `tfsdk:"number"`
	State       client.MeshBuildingBlockDefinitionVersionState `tfsdk:"state"`
	ContentHash string                                         `tfsdk:"content_hash"`
}

func (model *buildingBlockDefinition) SetVersionRefsFromClientDto(
	ctx context.Context, diags *diag.Diagnostics,
	versionDtos ...client.MeshBuildingBlockDefinitionVersion,
) (latestVersionDto client.MeshBuildingBlockDefinitionVersion) {
	versionLatestType := model.VersionLatest.Type(ctx)
	if !model.VersionLatestRelease.Type(ctx).Equal(versionLatestType) {
		diags.AddError("Inconsistent Version Attribute Types",
			"Output attribute version_latest has different schema than version_latest_release. "+
				"This is an implementation bug.")
	} else if !model.Versions.ElementType(ctx).Equal(versionLatestType) {
		diags.AddError("Inconsistent Version Attribute Types",
			"Output attribute versions has different element schema than version_latest_release. "+
				"This is an implementation bug.")
	}

	if len(versionDtos) == 0 {
		diags.AddError("Building Block Definition without versions found",
			"This should never happen for a properly created building block definition. "+
				"The API shows unexpected behavior.")
	}

	if diags.HasError() {
		return
	}

	// sort by ascending version number (newest version is last, oldest version first)
	slices.SortFunc(versionDtos, func(a, b client.MeshBuildingBlockDefinitionVersion) int {
		return cmp.Compare(*a.Spec.VersionNumber, *b.Spec.VersionNumber)
	})
	latestIndex := len(versionDtos) - 1
	latestVersionDto = versionDtos[latestIndex]

	var versionRefs []buildingBlockDefinitionVersionRef
	for _, versionDto := range versionDtos {
		if contentHash, err := versionContentHash(versionDto.Spec); err != nil {
			diags.AddError("Failed to determine content hash", fmt.Sprintf(
				"Content hashing of version DTO %s failed: %s", versionDto.Metadata.Uuid, err.Error(),
			))
		} else {
			versionRefs = append(versionRefs, buildingBlockDefinitionVersionRef{
				Uuid:        versionDto.Metadata.Uuid,
				Number:      *versionDto.Spec.VersionNumber,
				State:       *versionDto.Spec.State,
				ContentHash: contentHash,
			})
		}
	}
	if diags.HasError() {
		return
	}
	// assume versionRefs and versionDtos have the same length (which is always the case from the conversion above)
	versionAttributeTypes := model.VersionLatest.AttributeTypes(ctx)

	var convertDiags diag.Diagnostics
	model.Versions, convertDiags = types.ListValueFrom(ctx, versionLatestType, versionRefs)
	diags.Append(convertDiags...)

	model.VersionLatest, convertDiags = types.ObjectValueFrom(ctx, versionAttributeTypes, versionRefs[latestIndex])
	diags.Append(convertDiags...)

	if *versionDtos[latestIndex].Spec.State == client.MeshBuildingBlockDefinitionVersionState(client.MeshBuildingBlockDefinitionVersionStateReleased) {
		model.VersionLatestRelease = model.VersionLatest
	} else if len(versionDtos) > 1 {
		if *versionDtos[latestIndex-1].Spec.State == client.MeshBuildingBlockDefinitionVersionState(client.MeshBuildingBlockDefinitionVersionStateReleased) {
			model.VersionLatestRelease, convertDiags = types.ObjectValueFrom(ctx, versionAttributeTypes, versionRefs[latestIndex-1])
			diags.Append(convertDiags...)
		} else {
			diags.AddError("version_latest_release points to non-release version", fmt.Sprintf(
				"The attribute version_latest_release has state %s and not expected %s. The API shows unexpected behavior.",
				*versionDtos[latestIndex-1].Spec.State, client.MeshBuildingBlockDefinitionVersionStateReleased,
			))
		}
	} else {
		model.VersionLatestRelease = types.ObjectNull(versionAttributeTypes)
	}
	return
}

func versionContentHash(versionSpecJson client.MeshBuildingBlockDefinitionVersionSpec) (string, error) {
	// Ignore version and state fields by setting them to constant values, always!
	versionSpecJson.VersionNumber = nil
	versionSpecJson.State = nil

	// Converting it first from/to JSON makes hashing more stable, as fields with 'omitempty' are ignored.
	// Additionally, all numbers are converted to float64, even integers (which also allows changing DTO model types later on).
	// Also, the current Hasher implementation does not support structs for now, but map[string]any works!
	var buffer bytes.Buffer
	if err := json.NewEncoder(&buffer).Encode(versionSpecJson); err != nil {
		return "", err
	}
	var converted any
	if err := json.NewDecoder(&buffer).Decode(&converted); err != nil {
		return "", err
	}

	versionSpecHash, err := hash.Hasher{}.Hash(converted,
		// Safeguard against accidentally hashing plaintext values (should never happen as backend never returns plaintext values)
		hash.DisallowMapKeys("plaintext"),
	)
	if err != nil {
		return "", err
	}
	// add some versioning prefix to migrate possible changes in the hashes later on,
	// but let's hope migrating/fixing the hashes is never required
	return "v1:" + versionSpecHash.Hex(), nil
}

func (model buildingBlockDefinitionVersionSpec) ToClientDto(buildingBlockDefinitionUuid string, secretSupplier secret.PlaintextSupplier, diags *diag.Diagnostics) (dto client.MeshBuildingBlockDefinitionVersionSpec) {
	dto = client.MeshBuildingBlockDefinitionVersionSpec{
		MeshBuildingBlockDefinitionVersionSpecBase: model.MeshBuildingBlockDefinitionVersionSpecBase,
		Implementation: model.ImplementationToClientDto(secretSupplier, diags),
		Inputs:         model.InputsToClientDto(secretSupplier, diags),
	}
	dto.BuildingBlockDefinitionRef = client.BuildingBlockDefinitionRef{
		Kind: "meshBuildingBlockDefinition",
		Uuid: buildingBlockDefinitionUuid,
	}
	if model.Draft {
		dto.State = clientTypes.PtrTo(client.MeshBuildingBlockDefinitionVersionState(client.MeshBuildingBlockDefinitionVersionStateDraft))
	} else {
		dto.State = clientTypes.PtrTo(client.MeshBuildingBlockDefinitionVersionState(client.MeshBuildingBlockDefinitionVersionStateReleased))
	}
	if model.VersionNumber.IsNull() || model.VersionNumber.IsUnknown() {
		// start counting at version number 1 (maybe the backend could also define this?)
		dto.VersionNumber = clientTypes.PtrTo[int64](1)
	} else {
		dto.VersionNumber = model.VersionNumber.GetPtr(diags)
	}
	return
}

func (model buildingBlockDefinitionVersionSpec) ImplementationToClientDto(secretSupplier secret.PlaintextSupplier, diags *diag.Diagnostics) (dto client.MeshBuildingBlockDefinitionImplementation[*clientTypes.Secret]) {
	dto.MeshBuildingBlockDefinitionImplementationBase = model.Implementation.MeshBuildingBlockDefinitionImplementationBase
	if terraformImpl := model.Implementation.Terraform; terraformImpl != nil {
		dto.Terraform = &client.MeshBuildingBlockDefinitionTerraformImplementation[*clientTypes.Secret]{
			MeshBuildingBlockDefinitionTerraformImplementationBase: terraformImpl.MeshBuildingBlockDefinitionTerraformImplementationBase,
			SSHPrivateKey: terraformImpl.SSHPrivateKey.GetPlaintextIfFingerprintChanged(secretSupplier, diags),
		}
	} else if gitlabPipelineImpl := model.Implementation.GitlabPipeline; gitlabPipelineImpl != nil {
		dto.GitlabPipeline = &client.MeshBuildingBlockDefinitionGitLabPipelineImplementation[*clientTypes.Secret]{
			MeshBuildingBlockDefinitionGitLabPipelineImplementationBase: gitlabPipelineImpl.MeshBuildingBlockDefinitionGitLabPipelineImplementationBase,
			PipelineTriggerToken: gitlabPipelineImpl.PipelineTriggerToken.GetRequiredPlaintextIfFingerprintChanged(secretSupplier, diags),
		}
	}
	return
}

func (model buildingBlockDefinitionVersionSpec) InputsToClientDto(secretSupplier secret.PlaintextSupplier, diags *diag.Diagnostics) (dto map[string]*client.MeshBuildingBlockDefinitionInput) {
	dto = make(map[string]*client.MeshBuildingBlockDefinitionInput, len(model.Inputs))
	for k, input := range model.Inputs {
		dtoInput := &client.MeshBuildingBlockDefinitionInput{
			MeshBuildingBlockDefinitionInputAdapter: client.MeshBuildingBlockDefinitionInputAdapter[clientTypes.SecretOrAny]{
				MeshBuildingBlockDefinitionInputBase: input.MeshBuildingBlockDefinitionInputBase,
			},
		}
		// Secrets as sensitive inputs are transferred as [clientTypes.Variant.X] field,
		// non-sensitive as [clientTypes.Variant.Y] field.
		// Optionality is a bit cumbersome, as clientTypes.Variant is non-empty as it represent empty if neither X nor Y are non-zero.
		if input.Sensitive != nil {
			dtoInput.IsSensitive = true
			if argumentSecret := input.Sensitive.Argument.GetPlaintextIfFingerprintChanged(secretSupplier, diags); argumentSecret != nil {
				dtoInput.Argument = clientTypes.SecretOrAny{X: *argumentSecret}
			}
			if defaultValueSecret := input.Sensitive.DefaultValue.GetPlaintextIfFingerprintChanged(secretSupplier, diags); defaultValueSecret != nil {
				dtoInput.DefaultValue = clientTypes.SecretOrAny{X: *defaultValueSecret}
			}
		} else {
			dtoInput.IsSensitive = false
			if argument := input.Argument.Get(diags); argument != nil {
				dtoInput.Argument = clientTypes.SecretOrAny{Y: argument}
			}
			if defaultValue := input.DefaultValue.Get(diags); defaultValue != nil {
				dtoInput.DefaultValue = clientTypes.SecretOrAny{Y: defaultValue}
			}
		}
		dto[k] = dtoInput
	}
	return
}

//goland:noinspection GoMixedReceiverTypes
func (model *buildingBlockDefinitionVersionSpec) SetFromClientDto(dto client.MeshBuildingBlockDefinitionVersionSpec, diags *diag.Diagnostics) {
	model.MeshBuildingBlockDefinitionVersionSpecBase = dto.MeshBuildingBlockDefinitionVersionSpecBase

	model.State.SetRequired(dto.State, diags)
	model.VersionNumber.SetRequired(dto.VersionNumber, diags)

	// This flag may change if the BBD version was "externally" changed
	model.Draft = *dto.State == client.MeshBuildingBlockDefinitionVersionState(client.MeshBuildingBlockDefinitionVersionStateDraft)

	model.Implementation.MeshBuildingBlockDefinitionImplementationBase =
		dto.Implementation.MeshBuildingBlockDefinitionImplementationBase
	if terraformImplDto := dto.Implementation.Terraform; terraformImplDto != nil {
		if model.Implementation.Terraform == nil {
			diags.AddError("Failed to set version_spec from client DTO",
				"Got non-nil Terraform implementation from API, but current model does not have a Terraform implementation")
			return
		}
		model.Implementation.Terraform.MeshBuildingBlockDefinitionTerraformImplementationBase =
			terraformImplDto.MeshBuildingBlockDefinitionTerraformImplementationBase
		model.Implementation.Terraform.SSHPrivateKey.SetFromClientDto(terraformImplDto.SSHPrivateKey, diags)

	} else if gitlabPipelineImplDto := dto.Implementation.GitlabPipeline; gitlabPipelineImplDto != nil {
		if model.Implementation.GitlabPipeline == nil {
			diags.AddError("Failed to set version_spec from client DTO",
				"Got non-nil Terraform implementation from API, but current model does not have a Terraform implementation")
			return
		}
		model.Implementation.GitlabPipeline.MeshBuildingBlockDefinitionGitLabPipelineImplementationBase =
			gitlabPipelineImplDto.MeshBuildingBlockDefinitionGitLabPipelineImplementationBase
		model.Implementation.GitlabPipeline.PipelineTriggerToken.SetFromClientDto(gitlabPipelineImplDto.PipelineTriggerToken, diags)
	}

	inputKeys := slices.Sorted(maps.Keys(model.Inputs))
	inputKeysDto := slices.Sorted(maps.Keys(dto.Inputs))
	if !slices.Equal(inputKeys, inputKeysDto) {
		diags.AddError("Failed to set version_spec from client DTO",
			fmt.Sprintf("Got mismatching input names: Model=%s, DTO=%s", inputKeys, inputKeysDto))
		return
	}
	for inputKey, inputDto := range dto.Inputs {
		input := model.Inputs[inputKey]
		if input == nil {
			diags.AddError("Failed to set version_spec from client DTO",
				fmt.Sprintf("Cannot find non-nil model input for %s", inputKey))
			continue
		}
		input.MeshBuildingBlockDefinitionInputBase = inputDto.MeshBuildingBlockDefinitionInputBase

		if inputDto.IsSensitive {
			if inputDto.Argument.HasY() || inputDto.DefaultValue.HasY() {
				diags.AddError("Invalid client DTO received",
					fmt.Sprintf("Input %s has sensitive=true but the backend returned unencrypted values for either 'argument' or 'default_value'. "+
						"The backend is misbehaving and we can't handle this properly.", inputKey))
				continue
			} else if input.Sensitive == nil {
				diags.AddError("Failed to set version_spec from client DTO",
					fmt.Sprintf("Sensitive is nil for input '%s' although DTO says it is sensitive, this is inconsistent", inputKey))
				continue
			}
			inputDto.Argument.WithX(func(value *clientTypes.Secret) {
				input.Sensitive.Argument.SetFromClientDto(value, diags)
			})
			inputDto.DefaultValue.WithX(func(value *clientTypes.Secret) {
				input.Sensitive.DefaultValue.SetFromClientDto(value, diags)
			})
		} else {
			if inputDto.Argument.HasX() || inputDto.DefaultValue.HasX() {
				diags.AddError("Invalid client DTO received",
					fmt.Sprintf("Input %s has sensitive=false but the backend returned encrypted values for either 'argument' or 'default_value'. "+
						"The backend is misbehaving and we can't handle this properly.", inputKey))
				continue
			}
			inputDto.Argument.WithY(func(value *clientTypes.Any) {
				input.Argument.SetOptional(value, diags)
			})
			inputDto.DefaultValue.WithY(func(value *clientTypes.Any) {
				input.DefaultValue.SetOptional(value, diags)
			})
		}
	}
}
