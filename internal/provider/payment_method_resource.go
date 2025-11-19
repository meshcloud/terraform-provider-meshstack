package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &paymentMethodResource{}
	_ resource.ResourceWithConfigure   = &paymentMethodResource{}
	_ resource.ResourceWithImportState = &paymentMethodResource{}
)

func NewPaymentMethodResource() resource.Resource {
	return &paymentMethodResource{}
}

type paymentMethodResource struct {
	client *client.MeshStackProviderClient
}

func (r *paymentMethodResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_payment_method"
}

func (r *paymentMethodResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.MeshStackProviderClient)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *MeshStackProviderClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *paymentMethodResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Represents a meshStack payment method.\n\n~> **Note:** Managing payment methods requires an API key with sufficient admin permissions.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Payment method datatype version",
				Computed:            true,
				Default:             stringdefault.StaticString("v2"),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPaymentMethod`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPaymentMethod"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Payment method identifier.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`),
								"must be alphanumeric with dashes, must be lowercase, and have no leading, trailing or consecutive dashes",
							),
						},
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this payment method.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the payment method.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the payment method.",
						Computed:            true,
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the payment method.",
						Required:            true,
					},
					"expiration_date": schema.StringAttribute{
						MarkdownDescription: "Expiration date of the payment method (ISO 8601 format).",
						Optional:            true,
					},
					"amount": schema.Int64Attribute{
						MarkdownDescription: "Amount associated with the payment method.",
						Optional:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the payment method.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapValueMust(types.ListType{ElemType: types.StringType}, map[string]attr.Value{})),
					},
				},
			},
		},
	}
}

func (r *paymentMethodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	paymentMethod := client.MeshPaymentMethodCreate{
		Metadata: client.MeshPaymentMethodCreateMetadata{},
	}

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &paymentMethod.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &paymentMethod.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &paymentMethod.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &paymentMethod.Metadata.OwnedByWorkspace)...)

	if resp.Diagnostics.HasError() {
		return
	}

	createdPaymentMethod, err := r.client.CreatePaymentMethod(&paymentMethod)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Payment Method",
			"Could not create payment method, unexpected error: "+err.Error(),
		)
		return
	}

	if createdPaymentMethod.Spec.Tags == nil {
		createdPaymentMethod.Spec.Tags = make(map[string][]string)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, createdPaymentMethod)...)
}

func (r *paymentMethodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var workspace, name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	paymentMethod, err := r.client.ReadPaymentMethod(workspace, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read payment method '%s' in workspace '%s'", name, workspace),
			err.Error(),
		)
		return
	}

	if paymentMethod == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	if paymentMethod.Spec.Tags == nil {
		paymentMethod.Spec.Tags = make(map[string][]string)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, paymentMethod)...)
}

func (r *paymentMethodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	paymentMethod := client.MeshPaymentMethodCreate{
		Metadata: client.MeshPaymentMethodCreateMetadata{},
	}

	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("api_version"), &paymentMethod.ApiVersion)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec"), &paymentMethod.Spec)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &paymentMethod.Metadata.Name)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &paymentMethod.Metadata.OwnedByWorkspace)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("spec").AtName("tags"), &paymentMethod.Spec.Tags)...)

	if resp.Diagnostics.HasError() {
		return
	}

	updatedPaymentMethod, err := r.client.UpdatePaymentMethod(paymentMethod.Metadata.OwnedByWorkspace, paymentMethod.Metadata.Name, &paymentMethod)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Payment Method",
			"Could not update payment method, unexpected error: "+err.Error(),
		)
		return
	}

	if updatedPaymentMethod.Spec.Tags == nil {
		updatedPaymentMethod.Spec.Tags = make(map[string][]string)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, updatedPaymentMethod)...)
}

func (r *paymentMethodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var workspace, name string

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePaymentMethod(workspace, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not delete payment method '%s' in workspace '%s'", name, workspace),
			err.Error(),
		)
		return
	}
}

func (r *paymentMethodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	identifier := strings.Split(req.ID, ".")

	if len(identifier) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: workspace.payment-method-identifier Got: %q", req.ID),
		)
		return
	}

	workspace := identifier[0]
	paymentMethodName := identifier[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), workspace)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), paymentMethodName)...)
}
