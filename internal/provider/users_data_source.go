package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

type usersDataSource struct {
	meshUserClient client.MeshUserClient
}

func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshUserClient = client.User
	})...)
}

func (d *usersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List meshUsers, optionally only those with access to a workspace or project. " +
			"Deleted users are not listed. Use it to check many usernames at once, for example the members of " +
			"`meshstack_workspace_user_group` resources.\n\n" +
			"~> Needs an API key with the admin permission `ADM_USER_LIST`. meshStack has no workspace-scoped " +
			"permission to list users: the `workspace_identifier` filter narrows the list for such a key and grants no access.",

		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				MarkdownDescription: "List only users with a role in this workspace, directly or through a group.",
				Optional:            true,
			},
			"workspace_role": schema.StringAttribute{
				MarkdownDescription: "List only users with this workspace role, such as `Workspace Manager`, directly or through a group.",
				Optional:            true,
			},
			"project_identifier": schema.StringAttribute{
				MarkdownDescription: "List only users with access to this project. Requires `workspace_identifier`, " +
					"because a project identifier is only unique within its workspace.",
				Optional:   true,
				Validators: []validator.String{stringvalidator.AlsoRequires(path.MatchRoot("workspace_identifier"))},
			},
			"users": schema.ListNestedAttribute{
				MarkdownDescription: "The users, in the order meshStack created them.",
				Computed:            true,
				NestedObject:        schema.NestedAttributeObject{Attributes: userAttributes()},
			},
		},
	}
}

func (d *usersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspaceIdentifier, workspaceRole, projectIdentifier *string
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("workspace_identifier"), &workspaceIdentifier)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("workspace_role"), &workspaceRole)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("project_identifier"), &projectIdentifier)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.meshUserClient.List(ctx, client.MeshUserFilter{
		WorkspaceIdentifier: valueOrEmpty(workspaceIdentifier),
		WorkspaceRole:       valueOrEmpty(workspaceRole),
		ProjectIdentifier:   valueOrEmpty(projectIdentifier),
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to list users", err.Error())
		return
	}
	models := make([]userModel, len(users))
	for i, user := range users {
		models[i] = newUserModel(user)
	}
	resp.Diagnostics.Append(generic.SetAttributeTo(ctx, &resp.State, path.Root("users"), models)...)
}
