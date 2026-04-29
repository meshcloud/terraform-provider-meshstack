package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	clientTypes "github.com/meshcloud/terraform-provider-meshstack/client/types"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ resource.Resource               = &apiKeyResource{}
	_ resource.ResourceWithConfigure  = &apiKeyResource{}
	_ resource.ResourceWithModifyPlan = &apiKeyResource{}
)

func NewApiKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	meshApiKeyClient client.MeshApiKeyClient
}

func (r *apiKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		r.meshApiKeyClient = providerClient.ApiKey
	})...)
}

func (r *apiKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a meshStack API key." + previewDisclaimer(),

		Attributes: map[string]schema.Attribute{
			"metadata": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "API key metadata.",
				Attributes: map[string]schema.Attribute{
					"owned_by_workspace": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Identifier of the workspace that owns the API key.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"uuid": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "UUID of the API key (server-generated).",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "API key specification.",
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Display name of the API key.",
					},
					"permissions": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						MarkdownDescription: "Permissions assigned to the API key. " +
							"See [API Permissions](https://docs.meshcloud.io/api/authentication/api-permissions/) for detailed documentation. " +
							"Each permission exists as a workspace-scoped variant, an admin-scoped (`ADM_`-prefixed) variant, or both. " +
							"`ADM_`-prefixed permissions grant access across all workspaces. " +
							"`MANAGED_`-prefixed permissions grant cross-workspace access scoped to resources managed by the API key's workspace (e.g. building block definitions, landing zones, or platforms it owns)." +
							permissionsMarkdown(),
						Validators: []validator.Set{
							setvalidator.ValueStringsAre(
								stringvalidator.OneOf(client.Permissions.AllCodes()...),
							),
						},
					},
					"expires_at": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Expiry date of the API key (ISO date, e.g. `2025-12-31`). If omitted, the key never expires. Setting an expiry is recommended for security best practices. Changing this rotates the secret and a new client_secret is returned.",
					},
				},
			},

			"status": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "API key status.",
				PlanModifiers:       []planmodifier.Object{objectplanmodifier.UseStateForUnknown()},
				Attributes: map[string]schema.Attribute{
					"client_id": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "The client ID used for authentication.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
					"client_secret": schema.StringAttribute{
						Computed:            true,
						Sensitive:           true,
						MarkdownDescription: "The client secret for authentication. Stored in state after creation and rotated when `expires_at` changes. The API only returns this value on create and on secret rotation.",
						PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
					},
				},
			},
		},
	}
}

func (r *apiKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to do on create or delete.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	expiresAtPath := path.Root("spec").AtName("expires_at")
	var planExpiresAt, stateExpiresAt types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, expiresAtPath, &planExpiresAt)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, expiresAtPath, &stateExpiresAt)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// If either value is unknown, we can't compare — leave client_secret untouched.
	if planExpiresAt.IsUnknown() || stateExpiresAt.IsUnknown() {
		return
	}

	if planExpiresAt.ValueString() != stateExpiresAt.ValueString() {
		// Rotating the secret: mark client_secret as unknown so TF expects a new value.
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status").AtName("client_secret"), types.StringUnknown())...)
	}
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[*client.MeshApiKey](ctx, req.Plan, &resp.Diagnostics,
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
		generic.WithSetUnknownValueToZero(),
	)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.meshApiKeyClient.Create(ctx, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create API key", err.Error())
		return
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, created, generic.WithSliceTypeAsSet(clientTypes.IsSet))...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey, err := r.meshApiKeyClient.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read API key", err.Error())
		return
	}

	if apiKey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// The API never returns client_secret on Read; preserve it from state.
	if apiKey.Status != nil && apiKey.Status.ClientSecret == nil {
		stateSecret := generic.GetAttribute[*string](ctx, req.State, path.Root("status").AtName("client_secret"), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		apiKey.Status.ClientSecret = stateSecret
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, apiKey, generic.WithSliceTypeAsSet(clientTypes.IsSet))...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := generic.Get[*client.MeshApiKey](ctx, req.Plan, &resp.Diagnostics,
		generic.WithSliceTypeAsSet(clientTypes.IsSet),
		generic.WithSetUnknownValueToZero(),
	)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := r.meshApiKeyClient.Update(ctx, *plan.Metadata.Uuid, plan)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update API key", err.Error())
		return
	}

	if updated.Status.ClientSecret == nil {
		// Secret was not rotated; preserve value from state.
		stateSecret := generic.GetAttribute[*string](ctx, req.State, path.Root("status").AtName("client_secret"), &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		updated.Status.ClientSecret = stateSecret
	}

	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, updated, generic.WithSliceTypeAsSet(clientTypes.IsSet))...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.meshApiKeyClient.Delete(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Unable to delete API key", err.Error())
	}
}

func permissionsMarkdown() string {
	return client.Permissions.MarkdownString()
}
