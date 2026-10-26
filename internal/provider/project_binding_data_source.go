package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

type projectBindingModel struct {
	client.MeshProjectBinding
	Expired bool `tfsdk:"expired"`
}

type projectBindingClient[B any] interface {
	Read(ctx context.Context, name string) (*B, error)
	List(ctx context.Context, filter client.MeshProjectBindingListFilter) ([]B, error)
}

func newProjectBindingDataSource[B ~struct{ client.MeshProjectBinding }](
	subject, subjectNameDescription string,
	selectClient func(client.Client) projectBindingClient[B],
) datasource.DataSource {
	return &bindingDataSource[projectBindingModel]{
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
		lookups: []bindingLookup[projectBindingModel]{
			{path.Root("metadata").AtName("name"), func(m projectBindingModel) string { return m.Metadata.Name }},
			{path.Root("target_ref").AtName("owned_by_workspace"), func(m projectBindingModel) string { return m.TargetRef.OwnedByWorkspace }},
			{path.Root("target_ref").AtName("name"), func(m projectBindingModel) string { return m.TargetRef.Name }},
			{path.Root("subject").AtName("name"), func(m projectBindingModel) string { return m.Subject.Name }},
			{path.Root("role_ref").AtName("name"), func(m projectBindingModel) string { return m.RoleRef.Name }},
		},
		selectClient: func(c client.Client) bindingClient[projectBindingModel] {
			bindings := selectClient(c)
			// The API filters by project only, so subject and role are matched on the listed bindings.
			list := func(ctx context.Context, filter bindingFilter) ([]B, error) {
				return bindings.List(ctx, client.MeshProjectBindingListFilter{
					WorkspaceIdentifier: filter["target_ref.owned_by_workspace"],
					ProjectIdentifier:   filter["target_ref.name"],
				})
			}
			return newBindingClient(bindings.Read, list, func(binding B) projectBindingModel {
				b := struct{ client.MeshProjectBinding }(binding).MeshProjectBinding
				return projectBindingModel{MeshProjectBinding: b, Expired: bindingExpired(b.ExpiryDate)}
			})
		},
	}
}
