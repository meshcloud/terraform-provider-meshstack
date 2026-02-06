package generic

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/attr/xattr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	reflectwalk "github.com/meshcloud/terraform-provider-meshstack/internal/util/reflect"
)

var (
	// Ensure the implementation satisfies the expected interfaces to the Terraform Framework.
	// Must match the supported types in TypeFor.
	// Note that T=any is represented as jsontypes.NormalizedType, which is a basetypes.StringTypable.
	_ basetypes.StringTypable     = Type[any]{}
	_ basetypes.Int64Typable      = Type[any]{}
	_ basetypes.BoolTypable       = Type[any]{}
	_ basetypes.ObjectTypable     = Type[any]{}
	_ attr.TypeWithAttributeTypes = Type[any]{}
)

type Type[T any] struct {
	underlyingType attr.Type
	underlyingNull attr.Value
	valueFactory   func(any, *diag.Diagnostics) Value[T]
	// only non-nil for tftypes.Number types, such as int64
	bigFloatMapper func(*big.Float, *diag.Diagnostics) T
}

func TypeFor[T any]() Type[T] {
	var zero T
	zeroValue := reflect.ValueOf(zero)
	switch zeroValue.Kind() {
	case reflect.Int64:
		return Type[T]{
			basetypes.Int64Type{},
			basetypes.NewInt64Null(),
			func(v any, _ *diag.Diagnostics) Value[T] {
				return newValue[T](basetypes.NewInt64Value(reflect.ValueOf(v).Int()))
			},
			bigFloatMapper[T, int64]((*big.Float).Int64),
		}
	case reflect.String:
		return Type[T]{
			basetypes.StringType{},
			basetypes.NewStringNull(),
			func(v any, _ *diag.Diagnostics) Value[T] {
				strValue := reflect.ValueOf(v)
				if kind := strValue.Kind(); kind != reflect.String {
					// safeguard against unexpected usage, as reflect.Value.String() stringifies for many different kinds
					// but only reflect.String kind returns the actual string value!
					panic(fmt.Sprintf("expected kind %s, got %s", reflect.String, kind))
				}
				return newValue[T](basetypes.NewStringValue(strValue.String()))
			},
			nil,
		}
	case reflect.Bool:
		return Type[T]{
			basetypes.BoolType{},
			basetypes.NewBoolNull(),
			func(v any, _ *diag.Diagnostics) Value[T] {
				return newValue[T](basetypes.NewBoolValue(reflect.ValueOf(v).Bool()))
			},
			nil,
		}
	case reflect.Struct:
		attrType := buildObjectValueRecursively(nil, zeroValue).
			Type(context.Background())
		objectAttrType, ok := attrType.(basetypes.ObjectType)
		if !ok {
			panic(fmt.Sprintf("expected basetypes.ObjectType, got %T", attrType))
		}
		return Type[T]{
			objectAttrType,
			basetypes.NewObjectNull(objectAttrType.AttrTypes),
			func(v any, diags *diag.Diagnostics) Value[T] {
				return newValue[T](buildObjectValueRecursively(nil, reflect.ValueOf(v)))
			},
			nil,
		}
	case reflect.Invalid:
		// This happens for T=any (where reflection basically does not work on zero aka nil value)
		return Type[T]{
			jsontypes.NormalizedType{},
			jsontypes.NewNormalizedNull(),
			anyValueFactory[T],
			nil,
		}
	default:
		panic(fmt.Sprintf("type of kind %s is currently not supported", zeroValue.Kind()))
	}
}

func buildObjectValueRecursively(root reflectwalk.WalkPath, v reflect.Value) (attrValue attr.Value) {
	if err := reflectwalk.Walk(v, func(path reflectwalk.WalkPath, v reflect.Value) (err error) {
		err = path.Stop() // always stop as we recursively descend by calling buildObjectValueRecursively below
		kind := v.Kind()
		switch kind {
		// The cases here should follow along the supported cases in [TypeFor] above.
		case reflect.Int64:
			attrValue = newValue[int64](basetypes.NewInt64Value(v.Int()))
			return
		case reflect.String:
			attrValue = newValue[string](basetypes.NewStringValue(v.String()))
			return
		case reflect.Bool:
			attrValue = newValue[bool](basetypes.NewBoolValue(v.Bool()))
			return
		case reflect.Struct:
			// Now the recursive case for the 'struct' (corresponding to a Terraform ObjectValue)
			attrTypes := map[string]attr.Type{}
			attrValues := map[string]attr.Value{}
			if err := path.WalkStruct(v, func(path reflectwalk.WalkPath, structField *reflectwalk.StructField, value reflect.Value) (err error) {
				err = path.Stop() // always stop as we recursively descend by calling buildObjectValueRecursively below
				tfsdkTagValue := structField.Tag.Get("tfsdk")
				if tfsdkTagValue == "-" {
					return
				}
				tfsdkTagValue = strings.TrimSpace(tfsdkTagValue)
				if tfsdkTagValue == "" {
					return fmt.Errorf("path %s: tfsdk tag is empty", path)
				} else if _, exists := attrTypes[tfsdkTagValue]; exists {
					return fmt.Errorf("path %s: tfsdk tag is already set", path)
				}
				attrValues[tfsdkTagValue] = buildObjectValueRecursively(path, value)
				attrTypes[tfsdkTagValue] = attrValues[tfsdkTagValue].Type(context.Background())
				return
			}); err != nil {
				return err
			}
			// This is where we're lost now I suppose, so nested structs can be constructed with correct generic T
			objectValue, diags := basetypes.NewObjectValue(attrTypes, attrValues)
			if diags.HasError() {
				return fmt.Errorf("diags: %v", diags)
			}
			attrValue = objectValue
			return
		case reflect.Ptr:
			// Simply dereference pointers, as their nullability (or nil-ability) is covered by Terraform's type system
			return path.WalkPointer(v, func(path reflectwalk.WalkPath, v reflect.Value) (err error) {
				attrValue = buildObjectValueRecursively(path, v)
				return path.Stop()
			})
		default:
			// Bail out if unsupported
			return fmt.Errorf("value kind %s not supported", kind)
		}
	}, reflectwalk.VisitEmbeddedNilStructs(), reflectwalk.WithRoot(root)); err != nil {
		panic(fmt.Errorf("cannot build object value for struct %T: %w", v.Interface(), err))
	}
	return
}

func (t Type[T]) Validate(ctx context.Context, value tftypes.Value, path path.Path) diag.Diagnostics {
	//nolint:staticcheck // SA1019: We still need to support the deprecated TypeWithValidate interface
	if typeWithValidate, ok := t.underlyingType.(xattr.TypeWithValidate); ok {
		return typeWithValidate.Validate(ctx, value, path)
	}
	return nil
}

func (t Type[T]) TerraformType(ctx context.Context) tftypes.Type {
	return t.underlyingType.TerraformType(ctx)
}

func (t Type[T]) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	v, err := t.underlyingType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	return newValue[T](v), nil
}

func (t Type[T]) ValueType(ctx context.Context) attr.Value {
	return newValue[T](t.underlyingType.ValueType(ctx))
}

func (t Type[T]) Equal(other attr.Type) bool {
	if otherType, ok := other.(Type[T]); ok {
		return t.underlyingType.Equal(otherType.underlyingType)
	} else {
		return false
	}
}

func (t Type[T]) String() string {
	return fmt.Sprintf("%T", t)
}

func (t Type[T]) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (any, error) {
	return t.underlyingType.ApplyTerraform5AttributePathStep(step)
}

func fromValue[T any, Typable attr.Type, Value, Valuable attr.Value](
	t Type[T],
	ctx context.Context,
	value Value,
	mapper func(Typable, context.Context, Value) (Valuable, diag.Diagnostics),
) (result Valuable, diags diag.Diagnostics) {
	if typable, ok := t.underlyingType.(Typable); ok {
		v, mapperDiags := mapper(typable, ctx, value)
		diags.Append(mapperDiags...)
		if diags.HasError() {
			return result, diags
		}
		if valuable, ok := attr.Value(newValue[T](v)).(Valuable); ok {
			return valuable, diags
		} else {
			diags.AddError("Generic Type Conversion Error", fmt.Sprintf("Result of newValue of %s is not compatible with valuable %T", t, valuable))
			return
		}
	} else {
		diags.AddError("Generic Type Conversion Error", fmt.Sprintf("Underlying type %s of generic type %s not a %T", t.underlyingType, t, typable))
		return
	}
}

func (t Type[T]) ValueFromString(ctx context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.StringTypable.ValueFromString)
}

func (t Type[T]) ValueFromBool(ctx context.Context, value basetypes.BoolValue) (basetypes.BoolValuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.BoolTypable.ValueFromBool)
}

func (t Type[T]) ValueFromInt64(ctx context.Context, value basetypes.Int64Value) (basetypes.Int64Valuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.Int64Typable.ValueFromInt64)
}

func (t Type[T]) ValueFromObject(ctx context.Context, value basetypes.ObjectValue) (basetypes.ObjectValuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.ObjectTypable.ValueFromObject)
}

func (t Type[T]) WithAttributeTypes(attrTypes map[string]attr.Type) attr.TypeWithAttributeTypes {
	if objectType, ok := t.underlyingType.(basetypes.ObjectType); ok {
		t.underlyingType = objectType.WithAttributeTypes(attrTypes)
		t.underlyingNull = basetypes.NewObjectNull(attrTypes)
		return t
	}
	panic(fmt.Sprintf("underlying type %T does not support attribute types", t.underlyingType))
}

func (t Type[T]) AttributeTypes() map[string]attr.Type {
	if objectType, ok := t.underlyingType.(basetypes.ObjectType); ok {
		return objectType.AttributeTypes()
	}
	return nil
}
