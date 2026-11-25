package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

// NewProjectResource is a helper function to simplify the provider implementation.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	meshProjectClient client.MeshProjectClient
}

// Metadata returns the resource type name.
func (r *projectResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

// Configure adds the provider configured client to the resource.
func (r *projectResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.meshProjectClient = client.Project
	})...)
}

// Schema defines the schema for the resource.
func (r *projectResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single project by name and workspace.\n\n" +
			"~> **Tag Management:** Manage a project's tags inline via `spec.tags` here. This is the recommended " +
			"approach and the only one that can set tags that are mandatory at project creation. The dedicated " +
			"`meshstack_project_tags` / `meshstack_project_tag` resources exist for projects this configuration " +
			"does not create, and carry caveats (full-project rewrites, races, provisional schema) documented on those " +
			"resources — reach for them only when necessary. Whichever approach you choose, use exactly one per " +
			"project; **do not mix inline `tags` with the dedicated tag resources**, as doing so causes state drift and " +
			"plan conflicts.",

		Attributes: map[string]schema.Attribute{
			"ref": meshProjectRef(meshRefOptions{Description: "Reference to this project, can be used as `target_ref` in project binding resources.", Output: true}),

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Project metadata. Name and workspace of the target Project must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"owned_by_workspace": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"created_on": schema.StringAttribute{
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{Computed: true},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Project specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{Required: true},
					"tags":         tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.Project, Restricted: true}),
					"payment_method_identifier": projectPaymentMethodIdentifier("payment_method_ref",
						"Identifier of the payment method of the project."),
					"payment_method_ref": projectPaymentMethodRef("payment_method_identifier",
						"Reference to the payment method of the project. meshStack may set one when it creates the project."),
					"substitute_payment_method_identifier": projectPaymentMethodIdentifier("substitute_payment_method_ref",
						"Payment method that meshStack makes the payment method of the project when `payment_method_identifier` expires. "+
							"meshStack then clears this attribute, so update both attributes after the expiry to end the drift."),
					"substitute_payment_method_ref": projectPaymentMethodRef("substitute_payment_method_identifier",
						"Reference to the payment method that meshStack makes the payment method of the project when `payment_method_ref` expires. "+
							"meshStack then clears this attribute, so update it and `payment_method_ref` after the expiry to end the drift."),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[projectModel](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	create := plan.meshProject()

	defer lockProject(create.Metadata.OwnedByWorkspace, create.Metadata.Name)()
	project, err := r.meshProjectClient.Create(ctx, create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	project.Spec.Tags = create.Spec.Tags
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, newProjectModel(project))...)
}

// Read refreshes the Terraform state with the latest data.
func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// read needed attributes individually instead of the full data structure which can break because of missing elements
	var workspace, name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	project, err := r.meshProjectClient.Read(ctx, client.NewMeshProjectId(workspace, name))
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", err.Error())
	}

	if project == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Keep only the tags we already track. The API returns a superset (every schema property plus
	// injected restricted-tag defaults) that the caller may be unable to manage, so mirroring it
	// verbatim would surface as drift. On import there is no prior state (tags is null); we keep the
	// full set so a normal import round-trips, accepting that a restricted default would then show up.
	project.Spec.Tags = reconcileTrackedTags(ctx, req.State, path.Root("spec").AtName("tags"), project.Spec.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, newProjectModel(*project))...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := generic.Get[projectModel](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}
	update := plan.meshProject()

	defer lockProject(update.Metadata.OwnedByWorkspace, update.Metadata.Name)()
	project, err := r.meshProjectClient.Update(ctx, update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	project.Spec.Tags = update.Spec.Tags
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, newProjectModel(project))...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var workspace, name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	defer lockProject(workspace, name)()
	err := r.meshProjectClient.Delete(ctx, client.NewMeshProjectId(workspace, name))
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting project",
			"Could not delete project, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	identifier := strings.Split(req.ID, ".")

	if len(identifier) != 2 || identifier[0] == "" || identifier[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace.project. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), identifier[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), identifier[1])...)
}

func projectPaymentMethodIdentifier(conflictingRef, description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description,
		DeprecationMessage:  "Use `spec." + conflictingRef + "` instead. This attribute will be removed in a future release.",
		Optional:            true,
		Computed:            true,
		Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName(conflictingRef))},
	}
}

func projectPaymentMethodRef(conflictingIdentifier, description string) schema.SingleNestedAttribute {
	ref := meshRefByName(meshRefOptions{Kind: client.MeshObjectKind.PaymentMethod, Description: description})
	ref.Required = false
	ref.Optional = true
	ref.Computed = true
	ref.Validators = []validator.Object{
		objectvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName(conflictingIdentifier)),
		objectvalidator.AlsoRequires(path.MatchRelative().AtName("name")),
	}
	ref.Attributes["name"] = schema.StringAttribute{
		MarkdownDescription: "Named identifier (`metadata.name`) of `" + client.MeshObjectKind.PaymentMethod + "`. Required when the reference is set.",
		Optional:            true,
		Computed:            true,
	}
	return ref
}
