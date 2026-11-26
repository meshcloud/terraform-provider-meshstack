package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSource              = &workspaceUserGroupDataSource{}
	_ datasource.DataSourceWithConfigure = &workspaceUserGroupDataSource{}
)

func NewWorkspaceUserGroupDataSource() datasource.DataSource {
	return &workspaceUserGroupDataSource{}
}

type workspaceUserGroupDataSource struct {
	meshWorkspaceUserGroupClient client.MeshWorkspaceUserGroupClient
}

func (d *workspaceUserGroupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_group"
}

func (d *workspaceUserGroupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshWorkspaceUserGroupClient = client.WorkspaceUserGroup
	})...)
}

func (d *workspaceUserGroupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := workspaceUserGroupDataSourceAttributes()
	attributes["metadata"] = schema.SingleNestedAttribute{
		MarkdownDescription: "Identifiers of the workspace user group.",
		Required:            true,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace user group.",
				Required:            true,
			},
			"owned_by_workspace": schema.StringAttribute{
				MarkdownDescription: "Identifier of the workspace that owns the group.",
				Required:            true,
			},
		},
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a workspace user group by its identifier and the workspace that owns it. " +
			"For a global group, set `metadata.owned_by_workspace` to `meshcloud-global-partner`.",

		Attributes: attributes,
	}
}

func workspaceUserGroupDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ref": workspaceUserGroupRefOutput(),
		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Identifiers of the workspace user group.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"name":               computedString("Identifier of the workspace user group."),
				"owned_by_workspace": computedString("Identifier of the workspace that owns the group."),
			},
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Configuration of the workspace user group.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"display_name": computedString("Name of the group as meshPanel shows it."),
				"egid":         computedString("Identifier of the group in the system it comes from, such as your identity provider."),
				"members": schema.SetAttribute{
					MarkdownDescription: "Usernames of the group's members.",
					Computed:            true,
					ElementType:         types.StringType,
				},
				"tags": tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.WorkspaceUserGroup, Output: true}),
			},
		},
		"status": schema.SingleNestedAttribute{
			MarkdownDescription: "Information meshStack keeps about the group.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"source_system": computedString("System that manages the group, such as `SCIM` for a group your identity provider synchronizes."),
				"disabled_usernames": schema.SetAttribute{
					MarkdownDescription: "Members whose user is disabled, for example by SCIM.",
					Computed:            true,
					ElementType:         types.StringType,
				},
			},
		},
	}
}

func (d *workspaceUserGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var metadata workspaceUserGroupMetadataModel
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := d.meshWorkspaceUserGroupClient.Read(ctx, client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)),
			err.Error(),
		)
		return
	}
	if group == nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Workspace user group '%s' not found", client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)),
			fmt.Sprintf("No workspace user group '%s' exists in workspace '%s', or your API key cannot see it.", metadata.Name, metadata.OwnedByWorkspace),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newWorkspaceUserGroupModel(ctx, *group, &resp.Diagnostics))...)
}

func newWorkspaceUserGroupModel(ctx context.Context, group client.MeshWorkspaceUserGroup, diags *diag.Diagnostics) workspaceUserGroupModel {
	members, membersDiags := types.SetValueFrom(ctx, types.StringType, group.Spec.Members)
	diags.Append(membersDiags...)
	return workspaceUserGroupModel{
		Ref: newWorkspaceUserGroupRef(group.Metadata.OwnedByWorkspace, group.Metadata.Name),
		Metadata: workspaceUserGroupMetadataModel{
			Name:             group.Metadata.Name,
			OwnedByWorkspace: group.Metadata.OwnedByWorkspace,
		},
		Spec: workspaceUserGroupSpecModel{
			DisplayName: group.Spec.DisplayName,
			Egid:        group.Spec.Egid,
			Members:     members,
			Tags:        group.Spec.Tags,
		},
		Status: newWorkspaceUserGroupStatusModel(&group),
	}
}
