package generic

import (
	"context"
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

var (
	// Ensure the implementation satisfies the expected interfaces to the Terraform Framework.
	// Must match the Supported types, but note that T=any is represented as jsontypes.NormalizedType, which is a basetypes.StringTypable.
	_ basetypes.StringTypable = typeImpl[any]{}
	_ basetypes.Int64Typable  = typeImpl[any]{}
)

type typeImpl[T Supported] struct {
	underlyingType attr.Type
	underlyingNull attr.Value

	valueFactory           func(any, *diag.Diagnostics) Value[T]
	attributeSchemaFactory func(self typeImpl[T], opts attributeSchemaOptions) schema.Attribute

	// only non-nil for tftypes.Number types, such as int64
	bigFloatMapper func(*big.Float, *diag.Diagnostics) T
}

func typeFor[T Supported]() typeImpl[T] {
	var zero T
	switch reflect.ValueOf(zero).Kind() {
	case reflect.Int64:
		return typeImpl[T]{
			basetypes.Int64Type{},
			basetypes.NewInt64Null(),
			func(v any, _ *diag.Diagnostics) Value[T] {
				return newValue[T](basetypes.NewInt64Value(reflect.ValueOf(v).Int()))
			},
			func(self typeImpl[T], opts attributeSchemaOptions) schema.Attribute {
				return schema.Int64Attribute{
					CustomType:          self,
					MarkdownDescription: opts.MarkdownDescription,
					Optional:            opts.Flags.Has(AttributeOptional),
					Computed:            opts.Flags.Has(AttributeComputed),
					Required:            opts.Flags.Has(AttributeRequired),
				}
			},
			bigFloatMapper[T, int64]((*big.Float).Int64),
		}
	case reflect.String:
		return typeImpl[T]{
			basetypes.StringType{},
			basetypes.NewStringNull(),
			func(v any, _ *diag.Diagnostics) Value[T] {
				strValue := reflect.ValueOf(v)
				if kind := strValue.Kind(); kind != reflect.String {
					panic(fmt.Sprintf("expected kind %s, got %s", reflect.String, kind))
				}
				return newValue[T](basetypes.NewStringValue(strValue.String()))
			},
			func(self typeImpl[T], opts attributeSchemaOptions) schema.Attribute {
				return schema.StringAttribute{
					CustomType:          self,
					MarkdownDescription: opts.MarkdownDescription,
					Optional:            opts.Flags.Has(AttributeOptional),
					Computed:            opts.Flags.Has(AttributeComputed),
					Required:            opts.Flags.Has(AttributeRequired),
					Validators:          opts.StringValidators,
				}
			},
			nil,
		}
	case reflect.Invalid:
		// This happens for T=any (where reflection basically does not work on zero aka nil value)
		return typeImpl[T]{
			jsontypes.NormalizedType{},
			jsontypes.NewNormalizedNull(),
			anyValueFactory[T],
			func(self typeImpl[T], opts attributeSchemaOptions) schema.Attribute {
				return schema.StringAttribute{
					CustomType:          self,
					MarkdownDescription: opts.MarkdownDescription,
					Optional:            opts.Flags.Has(AttributeOptional),
					Computed:            opts.Flags.Has(AttributeComputed),
					Required:            opts.Flags.Has(AttributeRequired),
					Validators:          opts.StringValidators,
				}
			},
			nil,
		}
	default:
		panic(fmt.Sprintf("type of kind %s is currently not supported", reflect.ValueOf(zero).Kind()))
	}
}

func (t typeImpl[T]) Validate(context.Context, tftypes.Value, path.Path) diag.Diagnostics {
	// Still, basetypes.Int64Typable requires use to implement this deprecated interface
	panic("the deprecated interface xattr.TypeWithValidate is not supported")
}

func (t typeImpl[T]) TerraformType(ctx context.Context) tftypes.Type {
	return t.underlyingType.TerraformType(ctx)
}

func (t typeImpl[T]) ValueFromTerraform(ctx context.Context, value tftypes.Value) (attr.Value, error) {
	v, err := t.underlyingType.ValueFromTerraform(ctx, value)
	if err != nil {
		return nil, err
	}
	return newValue[T](v), nil
}

func (t typeImpl[T]) ValueType(ctx context.Context) attr.Value {
	return newValue[T](t.underlyingType.ValueType(ctx))
}

func (t typeImpl[T]) Equal(other attr.Type) bool {
	if otherType, ok := other.(typeImpl[T]); ok {
		return t.underlyingType.Equal(otherType.underlyingType)
	} else {
		return false
	}
}

func (t typeImpl[T]) String() string {
	return fmt.Sprintf("%T", t)
}

func (t typeImpl[T]) ApplyTerraform5AttributePathStep(step tftypes.AttributePathStep) (any, error) {
	return t.underlyingType.ApplyTerraform5AttributePathStep(step)
}

func fromValue[T Supported, Typable attr.Type, Value, Valuable attr.Value](
	t typeImpl[T],
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

func (t typeImpl[T]) ValueFromString(ctx context.Context, value basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.StringTypable.ValueFromString)
}

func (t typeImpl[T]) ValueFromInt64(ctx context.Context, value basetypes.Int64Value) (basetypes.Int64Valuable, diag.Diagnostics) {
	return fromValue(t, ctx, value, basetypes.Int64Typable.ValueFromInt64)
}
