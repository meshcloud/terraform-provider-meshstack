package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.List = AlphabeticallySorted{}

type AlphabeticallySorted struct{}

func (v AlphabeticallySorted) Description(ctx context.Context) string {
	return "Ensures list items are sorted alphabetically"
}

func (v AlphabeticallySorted) MarkdownDescription(ctx context.Context) string {
	return "Ensures list items are sorted alphabetically"
}

func (v AlphabeticallySorted) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	if len(elements) <= 1 {
		return
	}

	var stringValues []string
	for _, elem := range elements {
		strVal, ok := elem.(types.String)
		if !ok {
			resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
				req.Path,
				"Invalid Attribute Type",
				"Expected string elements in list",
			))
			return
		}

		if strVal.IsNull() || strVal.IsUnknown() {
			continue
		}

		stringValues = append(stringValues, strVal.ValueString())
	}

	for i := 0; i < len(stringValues)-1; i++ {
		if strings.Compare(stringValues[i], stringValues[i+1]) > 0 {
			resp.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
				req.Path,
				"Invalid Attribute Configuration",
				fmt.Sprintf("List items must be sorted alphabetically. Item at index %d (%q) comes after item at index %d (%q) alphabetically",
					i, stringValues[i], i+1, stringValues[i+1]),
			))
			return
		}
	}
}
