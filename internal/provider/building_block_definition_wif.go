package provider

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

const (
	versionWifIssuerDescription  = "OIDC issuer URL of the identity provider that issues the tokens of a run."
	versionWifSubjectDescription = "The subject claim of those tokens: the runner's subject template with every placeholder filled in for this definition. " +
		"Grant cloud access to this identity."
	versionWifAudienceDescription  = "Audience the federated identity token is issued for."
	versionWifTokenPathDescription = "Path the runner writes that token to."
)

func versionWifCloudDescription(cloud string) string {
	return "Workload identity federation values for " + cloud + ". Null when the runner does not federate with " + cloud + "."
}

// versionWifAttribute builds the computed block of the identity a run of a version presents to a cloud.
func versionWifAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"issuer":  schema.StringAttribute{MarkdownDescription: versionWifIssuerDescription, Computed: true},
			"subject": schema.StringAttribute{MarkdownDescription: versionWifSubjectDescription, Computed: true},
			"gcp":     versionWifCloudAttribute("GCP"),
			"aws":     versionWifCloudAttribute("AWS"),
			"azure":   versionWifCloudAttribute("Azure"),
		},
	}
}

// versionWifCloudAttribute builds the computed per-cloud block of that identity.
func versionWifCloudAttribute(cloud string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: versionWifCloudDescription(cloud),
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"audience":   schema.StringAttribute{MarkdownDescription: versionWifAudienceDescription, Computed: true},
			"token_path": schema.StringAttribute{MarkdownDescription: versionWifTokenPathDescription, Computed: true},
		},
	}
}

// dataSourceVersionWifAttribute is versionWifAttribute for a data source schema.
func dataSourceVersionWifAttribute(description string) datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		Attributes: map[string]datasourceschema.Attribute{
			"issuer":  datasourceschema.StringAttribute{MarkdownDescription: versionWifIssuerDescription, Computed: true},
			"subject": datasourceschema.StringAttribute{MarkdownDescription: versionWifSubjectDescription, Computed: true},
			"gcp":     dataSourceVersionWifCloudAttribute("GCP"),
			"aws":     dataSourceVersionWifCloudAttribute("AWS"),
			"azure":   dataSourceVersionWifCloudAttribute("Azure"),
		},
	}
}

func dataSourceVersionWifCloudAttribute(cloud string) datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		MarkdownDescription: versionWifCloudDescription(cloud),
		Computed:            true,
		Attributes: map[string]datasourceschema.Attribute{
			"audience":   datasourceschema.StringAttribute{MarkdownDescription: versionWifAudienceDescription, Computed: true},
			"token_path": datasourceschema.StringAttribute{MarkdownDescription: versionWifTokenPathDescription, Computed: true},
		},
	}
}
