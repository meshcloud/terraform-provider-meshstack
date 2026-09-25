package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// versionWifAttribute builds the computed block of the identity a run of a version presents to a cloud.
func versionWifAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"issuer": schema.StringAttribute{
				MarkdownDescription: "OIDC issuer URL of the identity provider that issues the tokens of a run.",
				Computed:            true,
			},
			"subject": schema.StringAttribute{
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
func versionWifCloudAttribute(cloud string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: "Workload identity federation values for " + cloud + ". Null when the runner does not federate with " + cloud + ".",
		Computed:            true,
		Attributes: map[string]schema.Attribute{
			"audience": schema.StringAttribute{
				MarkdownDescription: "Audience the federated identity token is issued for.",
				Computed:            true,
			},
			"token_path": schema.StringAttribute{
				MarkdownDescription: "Path the runner writes that token to.",
				Computed:            true,
			},
		},
	}
}
