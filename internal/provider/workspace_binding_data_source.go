package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

type workspaceBindingModel struct {
	client.MeshWorkspaceBinding
	Expired bool `tfsdk:"expired"`
}

func newWorkspaceBindingModel(binding client.MeshWorkspaceBinding) workspaceBindingModel {
	return workspaceBindingModel{MeshWorkspaceBinding: binding, Expired: bindingExpired(binding.ExpiryDate)}
}

// workspaceBindingView lets the lookups read a model that embeds workspaceBindingModel.
type workspaceBindingView interface {
	workspaceBinding() client.MeshWorkspaceBinding
}

func (m workspaceBindingModel) workspaceBinding() client.MeshWorkspaceBinding {
	return m.MeshWorkspaceBinding
}

type workspaceBindingClient[B any] interface {
	Read(ctx context.Context, name string) (*B, error)
	List(ctx context.Context, filter client.MeshWorkspaceBindingListFilter) ([]B, error)
}

func newWorkspaceBindingDataSource[B any, M workspaceBindingView](
	subject, subjectNameDescription string,
	listBySubject bool,
	selectClient func(client.Client) workspaceBindingClient[B],
	model func(B) M,
) *bindingDataSource[M] {
	return &bindingDataSource[M]{
		typeName: "workspace_" + subject + "_binding",
		kind:     "workspace " + subject + " binding",
		attributes: bindingDataSourceAttributes("workspace", "`Workspace Manager` or `Workspace Member`", subject, subjectNameDescription,
			schema.SingleNestedAttribute{
				MarkdownDescription: "Workspace to which this binding applies.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{MarkdownDescription: "Workspace identifier.", Computed: true},
				},
			}, "name"),
		lookups: []bindingLookup[M]{
			{path.Root("metadata").AtName("name"), func(m M) string { return m.workspaceBinding().Metadata.Name }},
			{path.Root("target_ref").AtName("name"), func(m M) string { return m.workspaceBinding().TargetRef.Name }},
			{path.Root("subject").AtName("name"), func(m M) string { return m.workspaceBinding().Subject.Name }},
			{path.Root("role_ref").AtName("name"), func(m M) string { return m.workspaceBinding().RoleRef.Name }},
		},
		selectClient: func(c client.Client) bindingClient[M] {
			bindings := selectClient(c)
			list := func(ctx context.Context, filter bindingFilter) ([]B, error) {
				listFilter := client.MeshWorkspaceBindingListFilter{
					WorkspaceIdentifier: filter["target_ref.name"],
					WorkspaceRole:       filter["role_ref.name"],
				}
				if listBySubject {
					listFilter.Username = filter["subject.name"]
				}
				return bindings.List(ctx, listFilter)
			}
			return newBindingClient(bindings.Read, list, model)
		},
	}
}
