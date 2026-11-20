package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ resource.Resource                = &projectTagValueResource{}
	_ resource.ResourceWithConfigure   = &projectTagValueResource{}
	_ resource.ResourceWithImportState = &projectTagValueResource{}
)

func NewProjectTagValueResource() resource.Resource {
	return &projectTagValueResource{}
}

type projectTagValueResource struct {
	meshProjectClient client.MeshProjectClient
}

type projectTagValueMetadata struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	ProjectIdentifier   types.String `tfsdk:"project_identifier"`
	Key                 types.String `tfsdk:"key"`
}

type projectTagValueSpec struct {
	Value types.String `tfsdk:"value"`
}

type projectTagValueModel struct {
	Metadata projectTagValueMetadata `tfsdk:"metadata"`
	Spec     projectTagValueSpec     `tfsdk:"spec"`
}

func (r *projectTagValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_tag_value"
}

func (r *projectTagValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(c client.Client) {
		r.meshProjectClient = c.Project
	})...)
}

func (r *projectTagValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = tagResourceSchema(
		"Manages a single value of a multi-select tag on a meshStack project.\n\n"+
			projectTagCaveatsFor("meshstack_project_tag_value")+"\n\n"+
			tagValueNote("project"),
		"Project tag value",
		map[string]schema.Attribute{
			"workspace_identifier": requiredReplaceOnChangeString("Identifier of the workspace that owns the target project."),
			"project_identifier":   requiredReplaceOnChangeString("Identifier of the target project."),
			"key":                  requiredReplaceOnChangeString("Key of the tag."),
		},
		map[string]schema.Attribute{"value": requiredReplaceOnChangeString("Value to add to the tag.")},
	)
}

func (r *projectTagValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectTagValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace := plan.Metadata.WorkspaceIdentifier.ValueString()
	name := plan.Metadata.ProjectIdentifier.ValueString()
	key := plan.Metadata.Key.ValueString()
	value := plan.Spec.Value.ValueString()

	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if project == nil {
		resp.Diagnostics.AddError("Error Reading Project", fmt.Sprintf("Project '%s' not found in workspace '%s'", name, workspace))
		return
	}

	if tags, changed := withTagValue(project.Spec.Tags, key, value); changed {
		if err := writeProjectTags(ctx, r.meshProjectClient, project, tags); err != nil {
			resp.Diagnostics.AddError("Error Adding Project Tag Value", fmt.Sprintf("Could not add value '%s' to tag '%s' on project '%s' in workspace '%s': %v", value, key, name, workspace, err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectTagValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectTagValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	project := readTaggedProject(ctx, r.meshProjectClient, state.Metadata.WorkspaceIdentifier.ValueString(), state.Metadata.ProjectIdentifier.ValueString(), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if project == nil || !slices.Contains(project.Spec.Tags[state.Metadata.Key.ValueString()], state.Spec.Value.ValueString()) {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectTagValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectTagValueModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectTagValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectTagValueModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workspace := state.Metadata.WorkspaceIdentifier.ValueString()
	name := state.Metadata.ProjectIdentifier.ValueString()
	key := state.Metadata.Key.ValueString()
	value := state.Spec.Value.ValueString()

	defer lockProject(workspace, name)()
	project := readTaggedProject(ctx, r.meshProjectClient, workspace, name, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || project == nil {
		return
	}

	if tags, changed := withoutTagValue(project.Spec.Tags, key, value); changed {
		if err := writeProjectTags(ctx, r.meshProjectClient, project, tags); err != nil {
			resp.Diagnostics.AddError("Error Removing Project Tag Value", fmt.Sprintf("Could not remove value '%s' from tag '%s' on project '%s' in workspace '%s': %v", value, key, name, workspace, err))
		}
	}
}

func (r *projectTagValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ids, key, value, ok := parseTagValueImportID(req.ID, 2)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace_identifier.project_identifier.key=value. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("workspace_identifier"), ids[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("project_identifier"), ids[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("spec").AtName("value"), value)...)
}
