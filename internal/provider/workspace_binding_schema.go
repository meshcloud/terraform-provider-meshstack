package provider

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func workspaceBindingDataSourceSchema(subject, subjectNameDescription string) schema.Schema {
	return schema.Schema{
		MarkdownDescription: "Single workspace " + subject + " binding by name.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace " + subject + " binding metadata.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "The name identifies the binding and is unique across the meshStack.",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 45),
						},
					},
				},
			},

			"role_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace role assigned by this binding.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of the workspace role, for example `Workspace Manager` or `Workspace Member`.",
						Computed:            true,
					},
				},
			},

			"target_ref": schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace to which this binding applies.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Workspace identifier.",
						Computed:            true,
					},
				},
			},

			"subject": schema.SingleNestedAttribute{
				MarkdownDescription: "The " + subject + " assigned by this binding.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: subjectNameDescription,
						Computed:            true,
					},
				},
			},

			"expiry_date": schema.StringAttribute{
				MarkdownDescription: "Expiry date of this binding as an ISO 8601 date (`YYYY-MM-DD`), after which the binding is no longer effective. Null if the binding never expires.",
				Computed:            true,
			},
		},
	}
}
