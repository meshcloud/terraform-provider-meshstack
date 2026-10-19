package provider

import (
	"fmt"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	resourceschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// versionWifAttribute builds the computed block of the identity a run of a version presents to a cloud.
func versionWifAttribute(description string) resourceschema.SingleNestedAttribute {
	return resourceschema.SingleNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		Attributes: map[string]resourceschema.Attribute{
			"issuer": resourceschema.StringAttribute{
				MarkdownDescription: "OIDC issuer URL of the identity provider that issues the tokens of a run.",
				Computed:            true,
			},
			"subject": resourceschema.StringAttribute{
				MarkdownDescription: "The subject claim of those tokens: the runner's subject template with every placeholder filled in for this definition. " +
					"Grant cloud access to this identity.",
				Computed: true,
			},
			"gcp":   versionWifCloudAttribute("GCP"),
			"aws":   versionWifCloudAttribute("AWS"),
			"azure": versionWifCloudAttribute("Azure"),
		},
	}
}

// versionWifCloudAttribute builds the computed per-cloud block of that identity.
func versionWifCloudAttribute(cloud string) resourceschema.SingleNestedAttribute {
	return resourceschema.SingleNestedAttribute{
		MarkdownDescription: "Workload identity federation values for " + cloud + ". Null when the runner does not federate with " + cloud + ".",
		Computed:            true,
		Attributes: map[string]resourceschema.Attribute{
			"audience": resourceschema.StringAttribute{
				MarkdownDescription: "Audience the federated identity token is issued for.",
				Computed:            true,
			},
			"token_path": resourceschema.StringAttribute{
				MarkdownDescription: "Path the runner writes that token to.",
				Computed:            true,
			},
		},
	}
}

// dataSourceVersionWifAttribute is versionWifAttribute for a data source schema, converted so both
// schemas keep the same shape and descriptions.
func dataSourceVersionWifAttribute(description string) datasourceschema.Attribute {
	return computedDataSourceAttribute(versionWifAttribute(description))
}

// computedDataSourceAttribute converts a computed-only resource attribute built from strings and nested
// objects into its data source counterpart.
func computedDataSourceAttribute(attribute resourceschema.Attribute) datasourceschema.Attribute {
	switch a := attribute.(type) {
	case resourceschema.StringAttribute:
		return datasourceschema.StringAttribute{MarkdownDescription: a.MarkdownDescription, Computed: true}
	case resourceschema.SingleNestedAttribute:
		attributes := make(map[string]datasourceschema.Attribute, len(a.Attributes))
		for name, nested := range a.Attributes {
			attributes[name] = computedDataSourceAttribute(nested)
		}
		return datasourceschema.SingleNestedAttribute{MarkdownDescription: a.MarkdownDescription, Computed: true, Attributes: attributes}
	default:
		panic(fmt.Sprintf("computedDataSourceAttribute: unsupported attribute type %T", attribute))
	}
}
