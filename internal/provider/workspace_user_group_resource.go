package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ resource.Resource                = &workspaceUserGroupResource{}
	_ resource.ResourceWithConfigure   = &workspaceUserGroupResource{}
	_ resource.ResourceWithImportState = &workspaceUserGroupResource{}
)

func NewWorkspaceUserGroupResource() resource.Resource {
	return &workspaceUserGroupResource{}
}

type workspaceUserGroupResource struct {
	meshWorkspaceUserGroupClient client.MeshWorkspaceUserGroupClient
}

type workspaceUserGroupMetadataModel struct {
	Name             string `tfsdk:"name"`
	OwnedByWorkspace string `tfsdk:"owned_by_workspace"`
}

type workspaceUserGroupSpecModel struct {
	DisplayName string              `tfsdk:"display_name"`
	Egid        *string             `tfsdk:"egid"`
	Members     types.Set           `tfsdk:"members"`
	Tags        map[string][]string `tfsdk:"tags"`
}

type workspaceUserGroupStatusModel struct {
	SourceSystem      *string  `tfsdk:"source_system"`
	DisabledUsernames []string `tfsdk:"disabled_usernames"`
}

type workspaceUserGroupModel struct {
	Metadata workspaceUserGroupMetadataModel `tfsdk:"metadata"`
	Spec     workspaceUserGroupSpecModel     `tfsdk:"spec"`
	Status   workspaceUserGroupStatusModel   `tfsdk:"status"`
}

func newWorkspaceUserGroupStatusModel(group *client.MeshWorkspaceUserGroup) workspaceUserGroupStatusModel {
	return workspaceUserGroupStatusModel{
		SourceSystem:      group.Metadata.SourceSystem,
		DisabledUsernames: group.Status.DisabledUsernames,
	}
}

func (spec workspaceUserGroupSpecModel) membersOr(ctx context.Context, unmanaged []string, diags *diag.Diagnostics) []string {
	if spec.Members.IsNull() {
		if unmanaged == nil {
			return []string{}
		}
		return unmanaged
	}
	members := []string{}
	diags.Append(spec.Members.ElementsAs(ctx, &members, false)...)
	return members
}

func (r *workspaceUserGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace_user_group"
}

func (r *workspaceUserGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshWorkspaceUserGroupClient = client.WorkspaceUserGroup
	})...)
}

func (r *workspaceUserGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A group of users in a meshStack workspace. Bind the group to a workspace or project role with " +
			"`meshstack_workspace_group_binding` or `meshstack_project_group_binding`.\n\n" +
			"~> **Members:** Set `spec.members` to manage the group's complete member list from this resource. " +
			"Leave it unset to manage the members elsewhere, for example with `meshstack_workspace_user_group_member` " +
			"or in meshPanel. Do not combine `spec.members` with `meshstack_workspace_user_group_member` resources " +
			"for the same group: each would remove the members the other adds.\n\n" +
			"~> **Deletion:** Deleting the group also deletes all of its role bindings.",

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Identifiers of the workspace user group.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name":               requiredReplaceOnChangeString("Identifier of the workspace user group. A group binding names the group by it in `subject.name`."),
					"owned_by_workspace": requiredReplaceOnChangeString("Identifier of the workspace that owns the group."),
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Configuration of the workspace user group.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Name of the group as meshPanel shows it.",
						Required:            true,
					},
					"egid": schema.StringAttribute{
						MarkdownDescription: "Identifier of the group in the system it comes from, such as the group's ID in your " +
							"identity provider. meshStack keeps it unique across all workspace user groups.",
						Optional: true,
					},
					"members": schema.SetAttribute{
						MarkdownDescription: "Usernames of the group's members. Each one must belong to an existing meshUser; " +
							"look it up with the `meshstack_user` or `meshstack_users` data source. " +
							"When you set this attribute, Terraform removes every member it does not list. " +
							"When you leave it unset, Terraform leaves the members as they are.",
						Optional:    true,
						ElementType: types.StringType,
					},
					"tags": tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.WorkspaceUserGroup}),
				},
			},

			"status": schema.SingleNestedAttribute{
				MarkdownDescription: "Information meshStack keeps about the group.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"source_system": schema.StringAttribute{
						MarkdownDescription: "System that manages the group, such as `SCIM` for a group your identity provider " +
							"synchronizes. Null for a group created with Terraform.",
						Computed: true,
					},
					"disabled_usernames": schema.SetAttribute{
						MarkdownDescription: "Members whose user is disabled, for example by SCIM. A disabled user stays a member " +
							"and regains access when the user is enabled again.",
						Computed:    true,
						ElementType: types.StringType,
					},
				},
			},
		},
	}
}

func (r *workspaceUserGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var metadata workspaceUserGroupMetadataModel
	var spec workspaceUserGroupSpecModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group := client.MeshWorkspaceUserGroup{
		Metadata: client.MeshWorkspaceUserGroupMetadata{Name: metadata.Name, OwnedByWorkspace: metadata.OwnedByWorkspace},
		Spec: client.MeshWorkspaceUserGroupSpec{
			DisplayName: spec.DisplayName,
			Egid:        spec.Egid,
			Members:     spec.membersOr(ctx, nil, &resp.Diagnostics),
			Tags:        spec.Tags,
		},
	}
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.meshWorkspaceUserGroupClient.Create(ctx, group)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not create workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(group.Metadata.OwnedByWorkspace, group.Metadata.Name)),
			err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceUserGroupModel{
		Metadata: metadata,
		Spec:     spec,
		Status:   newWorkspaceUserGroupStatusModel(&created),
	})...)
}

func (r *workspaceUserGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var metadata workspaceUserGroupMetadataModel
	var priorSpec types.Object
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec"), &priorSpec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, err := r.meshWorkspaceUserGroupClient.Read(ctx, client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)),
			err.Error(),
		)
		return
	}
	if group == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	membersManaged := priorSpec.IsNull()
	if !membersManaged {
		var priorMembers types.Set
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("spec").AtName("members"), &priorMembers)...)
		membersManaged = !priorMembers.IsNull()
	}
	members := types.SetNull(types.StringType)
	if membersManaged {
		var diags diag.Diagnostics
		members, diags = types.SetValueFrom(ctx, types.StringType, group.Spec.Members)
		resp.Diagnostics.Append(diags...)
	}
	tags := reconcileTrackedTags(ctx, req.State, path.Root("spec").AtName("tags"), group.Spec.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceUserGroupModel{
		Metadata: workspaceUserGroupMetadataModel{Name: group.Metadata.Name, OwnedByWorkspace: group.Metadata.OwnedByWorkspace},
		Spec: workspaceUserGroupSpecModel{
			DisplayName: group.Spec.DisplayName,
			Egid:        group.Spec.Egid,
			Members:     members,
			Tags:        tags,
		},
		Status: newWorkspaceUserGroupStatusModel(group),
	})...)
}

func (r *workspaceUserGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var metadata workspaceUserGroupMetadataModel
	var spec workspaceUserGroupSpecModel
	var sourceSystem *string
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("status").AtName("source_system"), &sourceSystem)...)
	if resp.Diagnostics.HasError() {
		return
	}

	identifier := client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)
	defer workspaceUserGroupWrites.lock(identifier)()

	var unmanagedMembers []string
	if spec.Members.IsNull() {
		current := readWorkspaceUserGroup(ctx, r.meshWorkspaceUserGroupClient, metadata.OwnedByWorkspace, metadata.Name, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		unmanagedMembers = current.Spec.Members
	}

	group := client.MeshWorkspaceUserGroup{
		Metadata: client.MeshWorkspaceUserGroupMetadata{Name: metadata.Name, OwnedByWorkspace: metadata.OwnedByWorkspace, SourceSystem: sourceSystem},
		Spec: client.MeshWorkspaceUserGroupSpec{
			DisplayName: spec.DisplayName,
			Egid:        spec.Egid,
			Members:     spec.membersOr(ctx, unmanagedMembers, &resp.Diagnostics),
			Tags:        spec.Tags,
		},
	}
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.meshWorkspaceUserGroupClient.Update(ctx, group)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Could not update workspace user group '%s'", identifier), err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, workspaceUserGroupModel{
		Metadata: metadata,
		Spec:     spec,
		Status:   newWorkspaceUserGroupStatusModel(&updated),
	})...)
}

func (r *workspaceUserGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var metadata workspaceUserGroupMetadataModel
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata"), &metadata)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.meshWorkspaceUserGroupClient.Delete(ctx, client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)); err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete workspace user group '%s'", client.NewMeshWorkspaceUserGroupId(metadata.OwnedByWorkspace, metadata.Name)),
			err.Error(),
		)
	}
}

func (r *workspaceUserGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ownedByWorkspace, name, ok := strings.Cut(req.ID, ".")
	if !ok || ownedByWorkspace == "" || name == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace.group. Got: %q", req.ID),
		)
		return
	}
	setWorkspaceUserGroupMetadata(ctx, &resp.State, ownedByWorkspace, name, &resp.Diagnostics)
}

func setWorkspaceUserGroupMetadata(ctx context.Context, state *tfsdk.State, ownedByWorkspace, name string, diags *diag.Diagnostics) {
	diags.Append(state.SetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), ownedByWorkspace)...)
	diags.Append(state.SetAttribute(ctx, path.Root("metadata").AtName("name"), name)...)
}

func readWorkspaceUserGroup(ctx context.Context, groups client.MeshWorkspaceUserGroupClient, ownedByWorkspace, name string, diags *diag.Diagnostics) *client.MeshWorkspaceUserGroup {
	identifier := client.NewMeshWorkspaceUserGroupId(ownedByWorkspace, name)
	group, err := groups.Read(ctx, identifier)
	if err != nil {
		diags.AddError(fmt.Sprintf("Could not read workspace user group '%s'", identifier), err.Error())
		return nil
	}
	if group == nil {
		diags.AddError(fmt.Sprintf("Workspace user group '%s' not found", identifier),
			fmt.Sprintf("No workspace user group '%s' exists in workspace '%s', or your API key cannot see it.", name, ownedByWorkspace))
	}
	return group
}
