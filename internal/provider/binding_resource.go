package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.ResourceWithConfigure   = &bindingResource[client.MeshWorkspaceUserBinding]{}
	_ resource.ResourceWithImportState = &bindingResource[client.MeshWorkspaceUserBinding]{}
)

type bindingResourceClient[B any] interface {
	Create(ctx context.Context, binding *B) (*B, error)
	Read(ctx context.Context, name string) (*B, error)
	Delete(ctx context.Context, name string) error
}

type bindingResource[B any] struct {
	// target is "workspace" or "project", subject is "user" or "group".
	target, subject string
	attributes      map[string]schema.Attribute
	selectClient    func(client.Client) bindingResourceClient[B]
	// approvalStatus lets Create and Read warn about a binding that waits for four-eyes approval.
	approvalStatus func(*B) bindingApprovalStatus
	client         bindingResourceClient[B]
}

func (r *bindingResource[B]) kind() string {
	return r.target + " " + r.subject + " binding"
}

func (r *bindingResource[B]) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.target + "_" + r.subject + "_binding"
}

func (r *bindingResource[B]) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.client = r.selectClient(client)
	})...)
}

func (r *bindingResource[B]) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: capitalize(r.kind()) + " assigns a " + r.subject + " with a specific role to a " + r.target + ".",
		Attributes:          r.attributes,
	}
}

func (r *bindingResource[B]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// An omitted expiry_date and the computed status plan as unknown, which the DTO's pointers cannot hold.
	plan := generic.Get[*B](ctx, req.Plan, &resp.Diagnostics, generic.WithSetUnknownValueToZero())
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating "+r.kind(),
			"Could not create "+r.kind()+", unexpected error: "+err.Error(),
		)
		return
	}

	var name string
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	warnIfWaitingForApproval(&resp.Diagnostics, name, r.approvalStatus(binding))
	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

func (r *bindingResource[B]) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	binding, err := r.client.Read(ctx, name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read "+r.kind(), err.Error())
	}

	if binding == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	warnIfWaitingForApproval(&resp.Diagnostics, name, r.approvalStatus(binding))
	resp.Diagnostics.Append(resp.State.Set(ctx, binding)...)
}

func (r *bindingResource[B]) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(capitalize(r.kind())+"s can't be updated", "Unsupported operation: "+r.kind()+"s can't be updated.")
}

func (r *bindingResource[B]) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var name string
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.Delete(ctx, name); err != nil {
		resp.Diagnostics.AddError(
			"Error deleting "+r.kind(),
			"Could not delete "+r.kind()+", unexpected error: "+err.Error(),
		)
	}
}

func (r *bindingResource[B]) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}

type bindingResourceSchemaOptions struct {
	target, subject        string
	roleNameDescription    string
	targetRef              map[string]schema.Attribute
	subjectNameDescription string
	expiryDate             schema.StringAttribute
}

func bindingResourceAttributes(opts bindingResourceSchemaOptions) map[string]schema.Attribute {
	name := requiredReplaceOnChangeString("The name identifies the binding and must be unique across the meshStack.")
	name.Validators = []validator.String{stringvalidator.LengthBetween(1, 45)}

	return map[string]schema.Attribute{
		"metadata": schema.SingleNestedAttribute{
			Required:            true,
			MarkdownDescription: capitalize(opts.target) + " " + opts.subject + " binding metadata.",
			Attributes:          map[string]schema.Attribute{"name": name},
		},
		"role_ref": schema.SingleNestedAttribute{
			MarkdownDescription: "Selects the role to use for this " + opts.target + " binding.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"name": requiredReplaceOnChangeString(opts.roleNameDescription),
			},
		},
		"target_ref": schema.SingleNestedAttribute{
			MarkdownDescription: "Selects the " + opts.target + " to which this binding applies.",
			Required:            true,
			Attributes:          opts.targetRef,
		},
		"subject": schema.SingleNestedAttribute{
			MarkdownDescription: "Selects the " + opts.subject + " for this binding.",
			Required:            true,
			Attributes: map[string]schema.Attribute{
				"name": requiredReplaceOnChangeString(opts.subjectNameDescription),
			},
		},
		"expiry_date": opts.expiryDate,
	}
}
