package provider

import (
	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

const (
	userBindingStatusDescription = "Read-only status of the binding."
	userDisabledDescription      = "Whether the bound user is disabled, for example through SCIM. A disabled user keeps the binding but has no access until the user is enabled again."
)

func userBindingStatusResourceAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		MarkdownDescription: userBindingStatusDescription,
		Computed:            true,
		PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes: map[string]schema.Attribute{
			"user_disabled": schema.BoolAttribute{
				MarkdownDescription: userDisabledDescription,
				Computed:            true,
			},
		},
	}
}

func userBindingStatusDataSourceAttribute() datasourceschema.SingleNestedAttribute {
	return datasourceschema.SingleNestedAttribute{
		MarkdownDescription: userBindingStatusDescription,
		Computed:            true,
		Attributes: map[string]datasourceschema.Attribute{
			"user_disabled": datasourceschema.BoolAttribute{
				MarkdownDescription: userDisabledDescription,
				Computed:            true,
			},
		},
	}
}
