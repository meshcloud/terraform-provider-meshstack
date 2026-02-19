package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

type Secret struct {
	Value   *string `tfsdk:"secret_value"`
	Version *string `tfsdk:"secret_version"`
	Hash    *string `tfsdk:"secret_hash"`
}

const (
	valueAttributeKey   = "secret_value"
	versionAttributeKey = "secret_version"
	hashAttributeKey    = "secret_hash"
)

// WithConverterSupport enables resources to use Secret representations in their ResourceSchema, while the client uses [clientTypes.Secret].
// See ValueFromConverter and ValueToConverter for details of the bidirectional conversion.
func WithConverterSupport(ctx context.Context, config, plan, state generic.AttributeGetter) generic.ConverterOptions {
	return generic.ConverterOptions{
		generic.WithValueFromConverterFor[clientTypes.Secret](generic.ValueFromConverterForTypedNilHandler[Secret](),
			func(attributePath path.Path, in clientTypes.Secret) (tftypes.Value, error) {
				return ValueFromConverter(ctx, plan, state, attributePath, in)
			}),
		generic.WithValueToConverterFor[clientTypes.Secret](func(attributePath path.Path, in tftypes.Value) (clientTypes.Secret, error) {
			return ValueToConverter(ctx, config, plan, state, attributePath)
		}),
	}
}

// WithDatasourceConverter converts read in hashes from the backend to the Terraform DatasourceSchema representation.
// As data sources are read-only, only generic.ValueFrom conversion is supported.
func WithDatasourceConverter() generic.ConverterOption {
	type datasourceSecret struct {
		Hash string `tfsdk:"secret_hash"`
	}
	return generic.WithValueFromConverterFor[clientTypes.Secret](generic.ValueFromConverterForTypedNilHandler[datasourceSecret](),
		func(attributePath path.Path, in clientTypes.Secret) (tftypes.Value, error) {
			return generic.ValueFrom(datasourceSecret{Hash: *in.Hash})
		})
}

// ValueFromConverter is called during [generic.ValueFrom] when converting Terraform value from a client DTO representation.
// According to the given plan and state (during create and update resource phase), this converter copies over a given hash value as the initial secret_version.
// This way resources with secrets can be imported without explicitly specifying the correct version initially.
// Typically used with WithConverterSupport in conjunction with ValueToConverter,
// but the building_block_definition resource has some special needs as it combines secret values with arbitrary json-encoded strings.
func ValueFromConverter(ctx context.Context, plan, state generic.AttributeGetter, attributePath path.Path, in clientTypes.Secret) (out tftypes.Value, err error) {
	var diags diag.Diagnostics
	defer func() {
		if diags.HasError() {
			err = errors.Join(err, fmt.Errorf("error while converting value from secret: %s", diags))
		}
	}()

	// check some values returned from the API
	if in.Hash == nil {
		return out, fmt.Errorf("API returned no hash for secret")
	} else if in.Plaintext != nil {
		return out, fmt.Errorf("API returned unexpected plaintext for secret")
	}

	getVersion := func(getter generic.AttributeGetter) *string {
		if getter == nil {
			return nil
		}
		var versionValue types.String
		diags.Append(getter.GetAttribute(ctx, attributePath.AtName(versionAttributeKey), &versionValue)...)
		if versionValue.IsUnknown() {
			return nil
		}
		return versionValue.ValueStringPointer()
	}

	var version *string
	if versionFromPlan := getVersion(plan); versionFromPlan != nil {
		// when importing or initially creating without version provided in plan,
		// take over Hash as the (computed) version field
		version = versionFromPlan
	} else if versionFromState := getVersion(state); versionFromState != nil {
		// finally, resort to version from state
		version = versionFromState
	} else {
		version = in.Hash
	}

	if diags.HasError() {
		return
	}

	return generic.ValueFrom(Secret{
		Value:   nil,
		Version: version,
		Hash:    in.Hash,
	})
}

// ValueToConverter is called during [generic.ValueTo] when converting Terraform value to a client DTO representation.
// According to the given plan and state (during create and update resource phase), this converter pulls the write-only attribute secret_value
// if the secret_version changes and provides this as a one-off value to the backend. Thus, secret rotation can be controlled with secret_version.
// Typically used with WithConverterSupport in conjunction with ValueFromConverter,
// but the building_block_definition resource has some special needs as it combines secret values with arbitrary json-encoded strings.
func ValueToConverter(ctx context.Context, config, plan, state generic.AttributeGetter, attributePath path.Path) (out clientTypes.Secret, err error) {
	var diags diag.Diagnostics
	defer func() {
		// check if getAttributeFrom failed and appended something to diags
		if diags.HasError() {
			err = errors.Join(err, fmt.Errorf("error while converting value to secret: %s", diags))
		}
	}()

	getAttributeFrom := func(getter generic.AttributeGetter, name string) (out *string) {
		diags.Append(getter.GetAttribute(ctx, attributePath.AtName(name), &out)...)
		if diags.HasError() {
			return
		}
		if out == nil {
			diags.AddError("Invalid secret state", fmt.Sprintf("Got nil/null value for attribute %s", name))
		} else if strings.TrimSpace(*out) == "" {
			diags.AddError("Invalid secret state", fmt.Sprintf("Got empty/whitespace only value '%s' for attribute %s", *out, name))
		}
		return
	}

	if state == nil {
		// there's no state available, so assume changed version (resource is newly created) and get the ephemeral value as plaintext!
		return clientTypes.Secret{Plaintext: getAttributeFrom(config, valueAttributeKey)}, nil
	} else if plan == nil {
		// there's no plan, so we're simply reading the hash from the state
		return clientTypes.Secret{Hash: getAttributeFrom(state, hashAttributeKey)}, nil
	}

	versionFromState := getAttributeFrom(state, versionAttributeKey)
	versionFromPlan := getAttributeFrom(plan, versionAttributeKey)
	if diags.HasError() {
		return
	}

	switch {
	case versionFromState == nil && versionFromPlan == nil:
		// both versions null -> no change
		fallthrough
	case versionFromState != nil && versionFromPlan != nil && *versionFromPlan == *versionFromState:
		// both non-null and values match -> no change
		// but send hash to backend again for validation that it hasn't changed from our end
		return clientTypes.Secret{Hash: getAttributeFrom(state, hashAttributeKey)}, nil
	default:
		// anything else is a change in version, which triggers getting the plaintext value again,
		// aka the secret value is rotated!
		return clientTypes.Secret{Plaintext: getAttributeFrom(config, valueAttributeKey)}, nil
	}

}

// SetHashToUnknownIfVersionChanged constructs a visitor which sets the secret_hash of the secret at the given attribute to unknown
// if the secret_version changes according to the given plan and state. Used together with WalkSecretPathsIn.
func SetHashToUnknownIfVersionChanged(ctx context.Context, plan, state generic.AttributeGetter, responsePlan generic.AttributeSetter) func(attributePath path.Path, diags *diag.Diagnostics) (versionChanged bool) {
	return func(attributePath path.Path, diags *diag.Diagnostics) (versionChanged bool) {
		var versionFromPlan types.String
		diags.Append(plan.GetAttribute(ctx, attributePath.AtName(versionAttributeKey), &versionFromPlan)...)
		if diags.HasError() {
			return
		}

		if versionFromPlan.IsUnknown() {
			versionChanged = true
			responsePlan.SetAttribute(ctx, attributePath.AtName(hashAttributeKey), types.StringUnknown())
			return
		}

		var versionFromState types.String
		diags.Append(state.GetAttribute(ctx, attributePath.AtName(versionAttributeKey), &versionFromState)...)
		if diags.HasError() {
			return
		}

		if !versionFromPlan.Equal(versionFromState) {
			versionChanged = true
			responsePlan.SetAttribute(ctx, attributePath.AtName(hashAttributeKey), types.StringUnknown())
		}
		return
	}
}

// WalkSecretPathsIn finds all secrets matching the Secret object representation in the given raw Terraform value (usually a req.Plan.Raw).
// It calls the given visitor with the attributePath where the secret is located.
// See SetHashToUnknownIfVersionChanged for an example visitor.
func WalkSecretPathsIn(raw tftypes.Value, diags *diag.Diagnostics, visitor func(attributePath path.Path, diags *diag.Diagnostics)) {
	convertTfPath := func(tfType *tftypes.AttributePath) (result path.Path) {
		result = path.Empty()
		for _, step := range tfType.Steps() {
			switch step := step.(type) {
			case tftypes.AttributeName:
				result = result.AtName(string(step))
			case tftypes.ElementKeyInt:
				result = result.AtListIndex(int(step))
			case tftypes.ElementKeyString:
				result = result.AtMapKey(string(step))
			default:
				panic(fmt.Sprintf("cannot handle: %#v", step))
			}
		}
		return result
	}

	secretType := func() tftypes.Type {
		emptySecret, err := generic.ValueFrom[*Secret](nil)
		if err != nil {
			panic(err)
		}
		return emptySecret.Type()
	}()

	if err := tftypes.Walk(raw, func(tfAttributePath *tftypes.AttributePath, value tftypes.Value) (continueWalk bool, err error) {
		defer func() {
			if diags.HasError() {
				// abort transform walk quickly when error encountered
				err = errors.Join(err, fmt.Errorf("diagnostics has errors at %s", tfAttributePath))
			}
		}()
		// check if the current TfValue can be converted to Secret
		if secretType.Equal(value.Type()) {
			visitor(convertTfPath(tfAttributePath), diags)
			return false, err
		}
		return true, err
	}); err != nil {
		diags.AddError("Walking plan value in search for secrets failed", err.Error())
	}
}
