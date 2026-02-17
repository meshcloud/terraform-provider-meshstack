package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.Object = ExactlyOneAttributeValidator{}

type ExactlyOneAttributeValidator struct{}

func (v ExactlyOneAttributeValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	var attributeNames, nonNullAttributeNames []string
	for attrName, value := range req.ConfigValue.Attributes() {
		if _, ok := value.Type(ctx).(types.ObjectType); !ok {
			continue
		}
		attributeNames = append(attributeNames, attrName)
		if !value.IsNull() {
			nonNullAttributeNames = append(nonNullAttributeNames, attrName)
		}
	}

	switch len(nonNullAttributeNames) {
	case 0:
		resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(req.Path, fmt.Sprintf("All attributes %s are null, exactly one is required.", attributeNames)))
	case 1:
		return
	default:
		resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(req.Path, fmt.Sprintf("More than exactly one attribute is not null: %s", nonNullAttributeNames)))
	}
}

func (v ExactlyOneAttributeValidator) Description(_ context.Context) string {
	return "Exactly one attribute of this object must be non-null"
}

func (v ExactlyOneAttributeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
