package secret

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	// Ensure the implementation satisfies the expected interfaces to the Terraform Framework.
	_ basetypes.ObjectTypable = typeImpl{}
)

type typeImpl struct {
	basetypes.ObjectType
}

func (t typeImpl) Equal(o attr.Type) bool {
	if other, ok := o.(typeImpl); ok {
		return t.ObjectType.Equal(other.ObjectType)
	} else {
		return false
	}
}

func (t typeImpl) String() string {
	return "secret.typeImpl"
}

func (t typeImpl) ValueFromObject(_ context.Context, in basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	var emptyPath path.Path
	return Secret{
		ObjectValue:   in,
		attributePath: &emptyPath,
	}, nil
}

func (t typeImpl) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.ObjectType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	objectValue, ok := attrValue.(basetypes.ObjectValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type of %T (expected ObjectValue)", attrValue)
	}
	objectValuable, diags := t.ValueFromObject(ctx, objectValue)
	if diags.HasError() {
		return nil, fmt.Errorf("unexpected error converting ObjectValue to ObjectValuable: %v", diags)
	}
	return objectValuable, nil
}

func (t typeImpl) ValueType(_ context.Context) attr.Value {
	return Secret{ObjectValue: types.ObjectUnknown(secretAttributeTypes)}
}
