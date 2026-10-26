package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

type workspaceBindingModel struct {
	client.MeshWorkspaceBinding
	Expired bool `tfsdk:"expired"`
}

type workspaceBindingClient[B any] interface {
	Read(ctx context.Context, name string) (*B, error)
	List(ctx context.Context, filter client.MeshWorkspaceBindingListFilter) ([]B, error)
}

func newWorkspaceBindingDataSource[B ~struct{ client.MeshWorkspaceBinding }](
	subject, subjectNameDescription string,
	listBySubject bool,
	selectClient func(client.Client) workspaceBindingClient[B],
) datasource.DataSource {
	return &bindingDataSource[workspaceBindingModel]{
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
		lookups: []bindingLookup[workspaceBindingModel]{
			{path.Root("metadata").AtName("name"), func(m workspaceBindingModel) string { return m.Metadata.Name }},
			{path.Root("target_ref").AtName("name"), func(m workspaceBindingModel) string { return m.TargetRef.Name }},
			{path.Root("subject").AtName("name"), func(m workspaceBindingModel) string { return m.Subject.Name }},
			{path.Root("role_ref").AtName("name"), func(m workspaceBindingModel) string { return m.RoleRef.Name }},
		},
		selectClient: func(c client.Client) bindingClient[workspaceBindingModel] {
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
			return newBindingClient(bindings.Read, list, func(binding B) workspaceBindingModel {
				b := struct{ client.MeshWorkspaceBinding }(binding).MeshWorkspaceBinding
				return workspaceBindingModel{MeshWorkspaceBinding: b, Expired: bindingExpired(b.ExpiryDate)}
			})
		},
	}
}
