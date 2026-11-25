package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

func NewProjectUserBindingResource() resource.Resource {
	binding := newProjectBindingResource("user", "Username.",
		func(c client.Client) bindingResourceClient[client.MeshProjectUserBinding] {
			return c.ProjectUserBinding
		})
	binding.attributes["status"] = userBindingStatusResourceAttribute()
	binding.approvalStatus = func(b *client.MeshProjectUserBinding) bindingApprovalStatus { return b.Status }
	return binding
}

func NewProjectGroupBindingResource() resource.Resource {
	binding := newProjectBindingResource("group", "Groupname.",
		func(c client.Client) bindingResourceClient[client.MeshProjectGroupBinding] {
			return c.ProjectGroupBinding
		})
	binding.attributes["status"] = groupBindingStatusResourceAttribute()
	binding.approvalStatus = func(b *client.MeshProjectGroupBinding) bindingApprovalStatus { return b.Status }
	return binding
}

func newProjectBindingResource[B any](
	subject, subjectNameDescription string,
	selectClient func(client.Client) bindingResourceClient[B],
) *bindingResource[B] {
	return &bindingResource[B]{
		target:       "project",
		subject:      subject,
		selectClient: selectClient,
		version:      1,
		upgraders: map[int64]resource.StateUpgrader{
			0: rawStateUpgrader(addKindToState([]string{"target_ref"}, client.MeshObjectKind.Project)),
		},
		converters: projectTargetRefConverters,
		attributes: bindingResourceAttributes(bindingResourceSchemaOptions{
			target:                 "project",
			subject:                subject,
			targetRef:              meshProjectRef(meshRefOptions{Description: "Selects the project to which this binding applies.", RequiresReplace: true}),
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

var projectTargetRefConverters = generic.ConverterOptions{
	generic.WithValueFromConverterFor[client.MeshProjectTargetRef](generic.ValueFromConverterForTypedNilHandler[projectRef](),
		func(_ path.Path, in client.MeshProjectTargetRef) (tftypes.Value, error) {
			return generic.ValueFrom(newProjectRef(in.OwnedByWorkspace, in.Name))
		}),
	generic.WithValueToConverterFor[client.MeshProjectTargetRef](func(_ path.Path, in tftypes.Value) (client.MeshProjectTargetRef, error) {
		ref, err := generic.ValueTo[projectRef](in, generic.WithSetUnknownValueToZero())
		return client.MeshProjectTargetRef{Name: ref.Name, OwnedByWorkspace: ref.OwnedByWorkspace}, err
	}),
}
