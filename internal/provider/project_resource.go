package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
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
		MarkdownDescription: "Single project by name and workspace.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Project datatype version",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshProject`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

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
					// TODO: Blocks would be more terraform-y.
					"tags": schema.MapAttribute{
						ElementType: types.ListType{ElemType: types.StringType},
						Optional:    true,
						Computed:    true,
						Default:     mapdefault.StaticValue(types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})),
					},
					// These can have defaults set upon creation
					"payment_method_identifier": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
					"substitute_payment_method_identifier": schema.StringAttribute{
						Optional: true,
						Computed: true,
					},
				},
			},
		},
	}
}

// These structs use Terraform types so that we can read the plan and check for unknown/null values.
type projectCreate struct {
	ApiVersion types.String    `json:"apiVersion" tfsdk:"api_version"`
	Kind       types.String    `json:"kind" tfsdk:"kind"`
	Metadata   projectMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       projectSpec     `json:"spec" tfsdk:"spec"`
}

type projectMetadata struct {
	Name             types.String `json:"name" tfsdk:"name"`
	OwnedByWorkspace types.String `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        types.String `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        types.String `json:"deletedOn" tfsdk:"deleted_on"`
}

type projectSpec struct {
	DisplayName                       types.String `json:"displayName" tfsdk:"display_name"`
	Tags                              types.Map    `json:"tags" tfsdk:"tags"`
	PaymentMethodIdentifier           types.String `json:"paymentMethodIdentifier" tfsdk:"payment_method_identifier"`
	SubstitutePaymentMethodIdentifier types.String `json:"substitutePaymentMethodIdentifier" tfsdk:"substitute_payment_method_identifier"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		diags = plan.Spec.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Unknown values result in empty strings but we want nil instead
	var paymentMethodIdentifier *string
	if !plan.Spec.PaymentMethodIdentifier.IsUnknown() {
		paymentMethodIdentifier = plan.Spec.PaymentMethodIdentifier.ValueStringPointer()
	}

	var substitutePaymentMethodIdentifier *string
	if !plan.Spec.SubstitutePaymentMethodIdentifier.IsUnknown() {
		paymentMethodIdentifier = plan.Spec.SubstitutePaymentMethodIdentifier.ValueStringPointer()
	}

	create := client.MeshProjectCreate{
		Metadata: client.MeshProjectCreateMetadata{
			Name:             plan.Metadata.Name.ValueString(),
			OwnedByWorkspace: plan.Metadata.OwnedByWorkspace.ValueString(),
		},
		Spec: client.MeshProjectSpec{
			DisplayName:                       plan.Spec.DisplayName.ValueString(),
			Tags:                              tags,
			PaymentMethodIdentifier:           paymentMethodIdentifier,
			SubstitutePaymentMethodIdentifier: substitutePaymentMethodIdentifier,
		},
	}

	project, err := r.meshProjectClient.Create(ctx, &create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating project",
			"Could not create project, unexpected error: "+err.Error(),
		)
		return
	}

	project.Spec.Tags = tags

	diags = resp.State.Set(ctx, project)
	resp.Diagnostics.Append(diags...)
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

	project, err := r.meshProjectClient.Read(ctx, workspace, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read project", err.Error())
	}

	if project == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// client data maps directly to the schema so we just need to set the state
	resp.Diagnostics.Append(resp.State.Set(ctx, project)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		diags = plan.Spec.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Unknown values result in empty strings but we want nil instead
	var paymentMethodIdentifier *string
	if !plan.Spec.PaymentMethodIdentifier.IsUnknown() {
		paymentMethodIdentifier = plan.Spec.PaymentMethodIdentifier.ValueStringPointer()
	}

	var substitutePaymentMethodIdentifier *string
	if !plan.Spec.SubstitutePaymentMethodIdentifier.IsUnknown() {
		paymentMethodIdentifier = plan.Spec.SubstitutePaymentMethodIdentifier.ValueStringPointer()
	}

	create := client.MeshProjectCreate{
		Metadata: client.MeshProjectCreateMetadata{
			Name:             plan.Metadata.Name.ValueString(),
			OwnedByWorkspace: plan.Metadata.OwnedByWorkspace.ValueString(),
		},
		Spec: client.MeshProjectSpec{
			DisplayName:                       plan.Spec.DisplayName.ValueString(),
			Tags:                              tags,
			PaymentMethodIdentifier:           paymentMethodIdentifier,
			SubstitutePaymentMethodIdentifier: substitutePaymentMethodIdentifier,
		},
	}

	project, err := r.meshProjectClient.Update(ctx, &create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating project",
			"Could not update project, unexpected error: "+err.Error(),
		)
		return
	}

	project.Spec.Tags = tags

	diags = resp.State.Set(ctx, project)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state client.MeshProject

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.meshProjectClient.Delete(ctx, state.Metadata.OwnedByWorkspace, state.Metadata.Name)
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
