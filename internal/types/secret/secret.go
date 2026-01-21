package secret

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
)

type Secret struct {
	basetypes.ObjectValue
	attributePath *path.Path
}

var (
	// Ensure the implementation satisfies the expected interfaces to the Terraform Framework.
	_ basetypes.ObjectValuable = Secret{}

	// Implementing xattr.ValidateableAttribute for Secret allows us to capture the Secret.attributePath of the secret value in Secret.ValidateAttribute,
	// which is used to retrieve the plaintext value from config in Secret.GetPlaintextIfFingerprintChanged.
	_ xattr.ValidateableAttribute = Secret{}
)

//goland:noinspection GoMixedReceiverTypes
func (s *Secret) SetFromClientDto(dto *clientTypes.Secret, diags *diag.Diagnostics) {
	if s.attributePath == nil {
		diags.AddError("Bug in Secret.SetFromClientDto",
			"Secret value attribute path is nil. "+
				"This provider implementation bug should never happen.")
		return
	}

	if dto == nil {
		// this happens for optional secrets (nullable), set null object then
		s.ObjectValue = types.ObjectNull(secretAttributeTypes)
		return
	} else if s.IsNull() {
		diags.AddAttributeError(*s.attributePath, "Converting Secret from client DTO failed",
			"The API told us that there is a secret present, but current plan/state does not say so. "+
				"Adapt configuration to make it consistent.")
		return
	} else if dto.Hash == nil {
		diags.AddAttributeError(*s.attributePath, "Converting Secret from client DTO failed",
			"Got no secret hash from API.")
		return
	}

	var fingerprintValue attr.Value
	if previousFingerprint := s.Attributes()["fingerprint"]; previousFingerprint.IsNull() || previousFingerprint.IsUnknown() {
		// when importing, take over Hash as the (computed) fingerprint field
		fingerprintValue = types.StringValue(*dto.Hash)
	} else {
		// otherwise, keep previous fingerprint as-is
		fingerprintValue = previousFingerprint
	}

	var diagsObjectValue diag.Diagnostics
	s.ObjectValue, diagsObjectValue = types.ObjectValue(secretAttributeTypes, map[string]attr.Value{
		"value":       types.StringNull(),
		"fingerprint": fingerprintValue,
		"hash":        types.StringValue(*dto.Hash),
	})
	diags.Append(diagsObjectValue...)
}

func (s Secret) Equal(o attr.Value) bool {
	other, ok := o.(Secret)
	if !ok {
		return false
	}
	return s.ObjectValue.Equal(other.ObjectValue)
}

func (s Secret) Type(context.Context) attr.Type {
	return typeImpl{basetypes.ObjectType{AttrTypes: secretAttributeTypes}}
}

func (s Secret) ValidateAttribute(_ context.Context, request xattr.ValidateAttributeRequest, _ *xattr.ValidateAttributeResponse) {
	if s.attributePath != nil {
		*s.attributePath = request.Path.Copy()
	}
}
