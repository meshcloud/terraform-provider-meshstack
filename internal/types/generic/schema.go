package generic

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type AttributeSchemaFlag int

const (
	AttributeOptional AttributeSchemaFlag = 1 << iota
	AttributeComputed
	AttributeRequired
)

func (f AttributeSchemaFlag) Has(flag AttributeSchemaFlag) bool {
	return f&flag != 0
}

type attributeSchemaOptions struct {
	MarkdownDescription string
	Flags               AttributeSchemaFlag
	StringValidators    []validator.String
}

type AttributeSchemaOption func(*attributeSchemaOptions)

func WithFlags(flags AttributeSchemaFlag) AttributeSchemaOption {
	return func(opts *attributeSchemaOptions) {
		opts.Flags = flags
	}
}

func WithMarkdownDescription(description string) AttributeSchemaOption {
	return func(opts *attributeSchemaOptions) {
		opts.MarkdownDescription = description
	}
}

func WithStringValidators(validators ...validator.String) AttributeSchemaOption {
	return func(opts *attributeSchemaOptions) {
		opts.StringValidators = validators
	}
}

func AttributeSchema[T Supported](options ...AttributeSchemaOption) schema.Attribute {
	opts := attributeSchemaOptions{}
	for _, option := range options {
		option(&opts)
	}
	t := typeFor[T]()
	return t.attributeSchemaFactory(t, opts)
}
