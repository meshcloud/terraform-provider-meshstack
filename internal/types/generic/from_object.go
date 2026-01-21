package generic

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func FromObject[T any](ctx context.Context, o types.Object, diags *diag.Diagnostics) (result T) {
	diags.Append(o.As(ctx, &result, basetypes.ObjectAsOptions{})...)
	return
}
