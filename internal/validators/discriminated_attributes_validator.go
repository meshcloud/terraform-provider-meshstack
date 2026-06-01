package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.Object = DiscriminatedAttributesValidator{}

// DiscriminatedAttributesValidator validates a discriminated-union object: the string value of
// the Discriminator attribute selects which one of the controlled attributes must be set, and
// requires every other controlled attribute to be null.
//
// The controlled set is the set of attribute names in RequiredFor's values. Discriminator values
// not present in RequiredFor are ignored (validate the discriminator itself with
// stringvalidator.OneOf). Null/unknown objects and unknown attribute values are skipped.
type DiscriminatedAttributesValidator struct {
	// Discriminator is the name of the string attribute whose value selects the required attribute.
	Discriminator string
	// RequiredFor maps a discriminator value to the attribute that must be set for that value.
	RequiredFor map[string]string
}

func (v DiscriminatedAttributesValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	attrs := req.ConfigValue.Attributes()

	discValue, ok := attrs[v.Discriminator].(types.String)
	if !ok || discValue.IsNull() || discValue.IsUnknown() {
		return
	}
	discriminator := discValue.ValueString()

	wanted, known := v.RequiredFor[discriminator]
	if !known {
		return
	}

	// Build the controlled set (deduplicated) so each attribute is checked once.
	controlled := make(map[string]struct{}, len(v.RequiredFor))
	for _, attrName := range v.RequiredFor {
		controlled[attrName] = struct{}{}
	}

	for attrName := range controlled {
		value, present := attrs[attrName]
		if !present || value.IsUnknown() {
			continue
		}
		switch {
		case attrName == wanted && value.IsNull():
			resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
				req.Path,
				fmt.Sprintf("Attribute %q is required when %s = %q.", wanted, v.Discriminator, discriminator),
			))
		case attrName != wanted && !value.IsNull():
			resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
				req.Path,
				fmt.Sprintf("Attribute %q must not be set when %s = %q (set %q instead).", attrName, v.Discriminator, discriminator, wanted),
			))
		}
	}
}

func (v DiscriminatedAttributesValidator) Description(_ context.Context) string {
	return fmt.Sprintf("The value of %q selects which attribute is required; the others must be null", v.Discriminator)
}

func (v DiscriminatedAttributesValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
