package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

type projectBindingModel struct {
	client.MeshProjectBinding
	Expired bool `tfsdk:"expired"`
}

func newProjectBindingModel(binding client.MeshProjectBinding) projectBindingModel {
	return projectBindingModel{MeshProjectBinding: binding, Expired: bindingExpired(binding.ExpiryDate)}
}

// projectBindingView lets the lookups read a model that embeds projectBindingModel.
type projectBindingView interface {
	projectBinding() client.MeshProjectBinding
}

func (m projectBindingModel) projectBinding() client.MeshProjectBinding {
	return m.MeshProjectBinding
}

type projectBindingClient[B any] interface {
	Read(ctx context.Context, name string) (*B, error)
	List(ctx context.Context, filter client.MeshProjectBindingListFilter) ([]B, error)
}

func newProjectBindingDataSource[B any, M projectBindingView](
	subject, subjectNameDescription string,
	selectClient func(client.Client) projectBindingClient[B],
	model func(B) M,
) *bindingDataSource[M] {
	return &bindingDataSource[M]{
		typeName: "project_" + subject + "_binding",
		kind:     "project " + subject + " binding",
		attributes: bindingDataSourceAttributes("project", "`Project Reader`", subject, subjectNameDescription,
			schema.SingleNestedAttribute{
				MarkdownDescription: "Project to which this binding applies.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name":               schema.StringAttribute{MarkdownDescription: "Project identifier.", Computed: true},
					"owned_by_workspace": schema.StringAttribute{MarkdownDescription: "Identifier of the workspace that owns the project.", Computed: true},
				},
			}, "name", "owned_by_workspace"),
		lookups: []bindingLookup[M]{
			{path.Root("metadata").AtName("name"), func(m M) string { return m.projectBinding().Metadata.Name }},
			{path.Root("target_ref").AtName("owned_by_workspace"), func(m M) string { return m.projectBinding().TargetRef.OwnedByWorkspace }},
			{path.Root("target_ref").AtName("name"), func(m M) string { return m.projectBinding().TargetRef.Name }},
			{path.Root("subject").AtName("name"), func(m M) string { return m.projectBinding().Subject.Name }},
			{path.Root("role_ref").AtName("name"), func(m M) string { return m.projectBinding().RoleRef.Name }},
		},
		selectClient: func(c client.Client) bindingClient[M] {
			bindings := selectClient(c)
			// The API filters by project only, so subject and role are matched on the listed bindings.
			list := func(ctx context.Context, filter bindingFilter) ([]B, error) {
				return bindings.List(ctx, client.MeshProjectBindingListFilter{
					WorkspaceIdentifier: filter["target_ref.owned_by_workspace"],
					ProjectIdentifier:   filter["target_ref.name"],
				})
			}
			return newBindingClient(bindings.Read, list, model)
		},
	}
}
