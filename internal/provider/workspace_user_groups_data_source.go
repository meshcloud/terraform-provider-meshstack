package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &workspaceUserGroupsDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceUserGroupsDataSource{}
)

func NewWorkspaceUserGroupsDataSource() datasource.DataSource {
	return &workspaceUserGroupsDataSource{}
}

type workspaceUserGroupsDataSource struct {
	meshWorkspaceUserGroupClient client.MeshWorkspaceUserGroupClient
}

func (d *workspaceUserGroupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_groups"
}

func (d *workspaceUserGroupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshWorkspaceUserGroupClient = client.WorkspaceUserGroup
	})...)
}

func (d *workspaceUserGroupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List workspace user groups, optionally only those that match the filters. " +
			"Use it to find a group that your identity provider created through SCIM, whose identifier meshStack generates: " +
			"filter by `egid` and pass `one(data.meshstack_workspace_user_groups.<name>.workspace_user_groups).metadata.name` " +
			"as `subject.name` of a `meshstack_workspace_group_binding` or `meshstack_project_group_binding`.\n\n" +
			"meshStack filters by `egid` and `display_name` since 2026.50.0. An older meshStack lists all groups, " +
			"and the provider applies the filters itself.",

		Attributes: map[string]schema.Attribute{
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "List only groups that this workspace owns. For global groups, set it to `meshcloud-global-partner`.",
				Optional:            true,
			},
			"egid": schema.StringAttribute{
				MarkdownDescription: "List only the group with this identifier in the system it comes from, such as the object ID " +
					"of an Entra ID group. meshStack keeps it unique, so the list has at most one group.",
				Optional: true,
			},
			"display_name": schema.StringAttribute{
				MarkdownDescription: "List only groups with this display name. The match ignores case. " +
					"Display names are not unique, so the list can have several groups.",
				Optional: true,
			},
			"workspace_user_groups": schema.ListNestedAttribute{
				MarkdownDescription: "The workspace user groups.",
				Computed:            true,
				NestedObject:        schema.NestedAttributeObject{Attributes: workspaceUserGroupDataSourceAttributes()},
			},
		},
	}
}

func (d *workspaceUserGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filter workspaceUserGroupsFilter
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("owned_by_workspace"), &filter.OwnedByWorkspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("egid"), &filter.Egid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("display_name"), &filter.DisplayName)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := d.meshWorkspaceUserGroupClient.List(ctx, client.MeshWorkspaceUserGroupFilter{
		WorkspaceIdentifier: valueOrEmpty(filter.OwnedByWorkspace),
		Egid:                valueOrEmpty(filter.Egid),
		DisplayName:         valueOrEmpty(filter.DisplayName),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list workspace user groups", err.Error())
		return
	}
	models := []workspaceUserGroupModel{}
	for _, group := range groups {
		if filter.matches(group) {
			models = append(models, newWorkspaceUserGroupModel(ctx, group, &resp.Diagnostics))
		}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace_user_groups"), models)...)
}

type workspaceUserGroupsFilter struct {
	OwnedByWorkspace *string
	Egid             *string
	DisplayName      *string
}

func (filter workspaceUserGroupsFilter) matches(group client.MeshWorkspaceUserGroup) bool {
	return (filter.Egid == nil || group.Spec.Egid != nil && *group.Spec.Egid == *filter.Egid) &&
		(filter.DisplayName == nil || strings.EqualFold(group.Spec.DisplayName, *filter.DisplayName))
}
