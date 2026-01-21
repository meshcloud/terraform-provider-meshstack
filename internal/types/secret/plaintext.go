package secret

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

func NewPlaintextSupplierForCreate(ctx context.Context, request resource.CreateRequest) PlaintextSupplier {
	return PlaintextSupplier{
		getConfigValue: getAttributeFunc(request.Config.GetAttribute).Adapt(ctx),
		getPlanValue:   getAttributeFunc(request.Plan.GetAttribute).Adapt(ctx),
	}
}

func NewPlaintextSupplierForUpdate(ctx context.Context, request resource.UpdateRequest) PlaintextSupplier {
	return PlaintextSupplier{
		getConfigValue: getAttributeFunc(request.Config.GetAttribute).Adapt(ctx),
		getPlanValue:   getAttributeFunc(request.Plan.GetAttribute).Adapt(ctx),
		getStateValue:  getAttributeFunc(request.State.GetAttribute).Adapt(ctx),
	}
}

type getAttributeFunc func(ctx context.Context, path path.Path, target any) diag.Diagnostics

type getStringValueFunc func(attributePath path.Path, diags *diag.Diagnostics) *string

func (f getAttributeFunc) Adapt(ctx context.Context) getStringValueFunc {
	return func(attributePath path.Path, diags *diag.Diagnostics) (result *string) {
		diags.Append(f(ctx, attributePath, &result)...)
		return
	}
}

type PlaintextSupplier struct {
	getConfigValue getStringValueFunc
	getPlanValue   getStringValueFunc
	getStateValue  getStringValueFunc
}

func (s Secret) GetRequiredPlaintextIfFingerprintChanged(supplier PlaintextSupplier, diags *diag.Diagnostics) *clientTypes.Secret {
	plaintext := s.GetPlaintextIfFingerprintChanged(supplier, diags)
	if plaintext == nil {
		diags.AddAttributeError(*s.attributePath, "Required plaintext not present",
			"Check other diagnostic output for details.")
	}
	return plaintext
}

func (s Secret) GetPlaintextIfFingerprintChanged(supplier PlaintextSupplier, diags *diag.Diagnostics) *clientTypes.Secret {
	if s.attributePath == nil {
		diags.AddError("Bug in GetPlaintextIfFingerprintChanged",
			"Secret value attribute path is nil. "+
				"This provider implementation bug should never happen.")
		return nil
	}

	if s.IsNull() {
		// happens for optional secrets
		return nil
	}

	getPlaintextSecret := func() *clientTypes.Secret {
		plaintextValue := supplier.getConfigValue(s.attributePath.AtName("value"), diags)
		if diags.HasError() {
			return nil
		}
		return &clientTypes.Secret{Plaintext: plaintextValue}
	}

	if supplier.getStateValue == nil {
		// there's no state available, so assume changed fingerprint (resource is newly created) and get the value!
		return getPlaintextSecret()
	}

	fingerprintPath := s.attributePath.AtName("fingerprint")
	fingerprintState := supplier.getStateValue(fingerprintPath, diags)
	fingerprintPlan := supplier.getPlanValue(fingerprintPath, diags)
	if diags.HasError() {
		return nil
	}

	switch {
	case fingerprintState == nil && fingerprintPlan == nil:
		// both fingerprints null -> no change
		fallthrough
	case fingerprintState != nil && fingerprintPlan != nil && *fingerprintPlan == *fingerprintState:
		// both non-null and values match -> no change
		// but send hash to backend again for validation that it hasn't changed from our end
		hashState := supplier.getStateValue(s.attributePath.AtName("hash"), diags)
		if diags.HasError() {
			return nil
		}
		return &clientTypes.Secret{Hash: hashState}
	default:
		// anything else is a change in fingerprint, which triggers getting the plaintext value again,
		// aka the secret is rotated!
		return getPlaintextSecret()
	}
}
