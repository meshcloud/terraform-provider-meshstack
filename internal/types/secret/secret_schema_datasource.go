package secret

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

type DatasourceSchemaOptions struct {
	MarkdownDescription string
}

// DatasourceSchema represents a secret read out from the backend (hash-only, sorry).
// Still the hash is useful to detect if secrets have changed externally.
// Use together with generic.ValueFrom and WithDatasourceConverter.
func DatasourceSchema(opts DatasourceSchemaOptions) (result schema.SingleNestedAttribute) {
	return schema.SingleNestedAttribute{
		MarkdownDescription: opts.MarkdownDescription,
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			hashAttributeKey: schema.StringAttribute{
				MarkdownDescription: "Hash value of the secret stored in the backend. " +
					"If this hash has changed without changes in the version attribute, the secret was changed externally.",
				Computed: true,
			},
		},
	}
}
