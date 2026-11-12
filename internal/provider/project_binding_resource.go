package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/meshcloud/meshstack-cli/client"
)

func NewProjectUserBindingResource() resource.Resource {
	binding := newProjectBindingResource("user", "Username.",
		func(c client.Client) bindingResourceClient[client.MeshProjectUserBinding] {
			return c.ProjectUserBinding
		})
	binding.attributes["status"] = userBindingStatusResourceAttribute()
	return binding
}

func NewProjectGroupBindingResource() resource.Resource {
	return newProjectBindingResource("group", "Groupname.",
		func(c client.Client) bindingResourceClient[client.MeshProjectGroupBinding] {
			return c.ProjectGroupBinding
		})
}

func newProjectBindingResource[B any](
	subject, subjectNameDescription string,
	selectClient func(client.Client) bindingResourceClient[B],
) *bindingResource[B] {
	return &bindingResource[B]{
		target:       "project",
		subject:      subject,
		selectClient: selectClient,
		attributes: bindingResourceAttributes(bindingResourceSchemaOptions{
			target:  "project",
			subject: subject,
			targetRef: map[string]schema.Attribute{
				"name":               requiredReplaceOnChangeString("Project identifier."),
				"owned_by_workspace": requiredReplaceOnChangeString("Identifier of workspace containing the target project."),
			},
			subjectNameDescription: subjectNameDescription,
			expiryDate: schema.StringAttribute{
				MarkdownDescription: "Last day the binding is effective (`YYYY-MM-DD`); null if it never expires. Changing it replaces the binding.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		}),
	}
}
