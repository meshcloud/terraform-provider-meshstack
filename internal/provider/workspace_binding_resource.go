package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewWorkspaceUserBindingResource() resource.Resource {
	binding := newWorkspaceBindingResource("user",
		"Name of the workspace role to assign, for example `Workspace Owner`, `Workspace Manager` or `Workspace Member`. meshStack looks the role up by this name and rejects one it does not know.",
		"Username.",
		func(c client.Client) bindingResourceClient[client.MeshWorkspaceUserBinding] {
			return c.WorkspaceUserBinding
		})
	binding.attributes["status"] = userBindingStatusResourceAttribute()
	binding.approvalStatus = func(b *client.MeshWorkspaceUserBinding) bindingApprovalStatus { return b.Status }
	return binding
}

func NewWorkspaceGroupBindingResource() resource.Resource {
	binding := newWorkspaceBindingResource("group",
		"Name of the workspace role to assign, for example `Workspace Manager` or `Workspace Member`. meshStack looks the role up by this name and rejects one it does not know. `Workspace Owner` can only be assigned to a user, not to a group.",
		"Identifier of the workspace user group.",
		func(c client.Client) bindingResourceClient[client.MeshWorkspaceGroupBinding] {
			return c.WorkspaceGroupBinding
		})
	binding.attributes["status"] = groupBindingStatusResourceAttribute()
	binding.approvalStatus = func(b *client.MeshWorkspaceGroupBinding) bindingApprovalStatus { return b.Status }
	return binding
}

func newWorkspaceBindingResource[B any](
	subject, roleNameDescription, subjectNameDescription string,
	selectClient func(client.Client) bindingResourceClient[B],
) *bindingResource[B] {
	return &bindingResource[B]{
		target:       "workspace",
		subject:      subject,
		selectClient: selectClient,
		attributes: bindingResourceAttributes(bindingResourceSchemaOptions{
			target:              "workspace",
			subject:             subject,
			roleNameDescription: roleNameDescription,
			targetRef: map[string]schema.Attribute{
				"name": requiredReplaceOnChangeString("Workspace identifier."),
			},
			subjectNameDescription: subjectNameDescription,
			expiryDate: schema.StringAttribute{
				MarkdownDescription: "Expiry date for this binding as an ISO 8601 date (`YYYY-MM-DD`). After this date the binding is no longer effective. " +
					"If omitted, the binding never expires — unless recertification is enabled for the bound role, in which case meshStack assigns the maximum allowed expiry date (today plus the configured recertification period) and returns it here. " +
					"That is why this attribute is also computed: its value reflects what meshStack stored, which may differ from an unset configuration. " +
					"An explicitly configured date beyond the recertification maximum (or in the past) is rejected. Changing this value replaces the binding.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		}),
	}
}
