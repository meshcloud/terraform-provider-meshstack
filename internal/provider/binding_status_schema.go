package provider

import (
	"fmt"

	datasourceschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/meshcloud/meshstack-cli/client"
)

const (
	bindingStatusDescription = "Read-only status of the binding."
	userDisabledDescription  = "Whether the bound user is disabled, for example through SCIM. A disabled user keeps the binding but has no access until the user is enabled again."
	approvalDescription      = "`ASSIGNED` when the role is in effect. `PENDING` while the binding waits for approval in meshPanel, because your meshStack requires four-eyes approval for role assignments. A pending binding gives no access yet."
)

func userBindingStatusResourceAttribute() schema.SingleNestedAttribute {
	return bindingStatusResourceAttribute(map[string]schema.Attribute{
		"user_disabled": schema.BoolAttribute{
			MarkdownDescription: userDisabledDescription,
			Computed:            true,
		},
	})
}

func groupBindingStatusResourceAttribute() schema.SingleNestedAttribute {
	return bindingStatusResourceAttribute(map[string]schema.Attribute{})
}

func bindingStatusResourceAttribute(attributes map[string]schema.Attribute) schema.SingleNestedAttribute {
	attributes["approval"] = schema.StringAttribute{
		MarkdownDescription: approvalDescription,
		Computed:            true,
	}
	return schema.SingleNestedAttribute{
		MarkdownDescription: bindingStatusDescription,
		Computed:            true,
		PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
		Attributes:          attributes,
	}
}

func userBindingStatusDataSourceAttribute() datasourceschema.SingleNestedAttribute {
	return bindingStatusDataSourceAttribute(map[string]datasourceschema.Attribute{
		"user_disabled": datasourceschema.BoolAttribute{
			MarkdownDescription: userDisabledDescription,
			Computed:            true,
		},
	})
}

func groupBindingStatusDataSourceAttribute() datasourceschema.SingleNestedAttribute {
	return bindingStatusDataSourceAttribute(map[string]datasourceschema.Attribute{})
}

func bindingStatusDataSourceAttribute(attributes map[string]datasourceschema.Attribute) datasourceschema.SingleNestedAttribute {
	attributes["approval"] = datasourceschema.StringAttribute{
		MarkdownDescription: approvalDescription,
		Computed:            true,
	}
	return datasourceschema.SingleNestedAttribute{
		MarkdownDescription: bindingStatusDescription,
		Computed:            true,
		Attributes:          attributes,
	}
}

type bindingApprovalStatus interface {
	WaitsForApproval() bool
}

var (
	_ bindingApprovalStatus = (*client.MeshUserBindingStatus)(nil)
	_ bindingApprovalStatus = (*client.MeshGroupBindingStatus)(nil)
)

func warnIfWaitingForApproval(diags *diag.Diagnostics, bindingName string, status bindingApprovalStatus) {
	if !status.WaitsForApproval() {
		return
	}
	diags.AddWarning(
		"Binding waits for approval",
		fmt.Sprintf("Your meshStack requires four-eyes approval for role assignments, so the binding %q gives no access yet. "+
			"Workspace managers approve it in meshPanel. Then status.approval becomes ASSIGNED on the next refresh.", bindingName),
	)
}
