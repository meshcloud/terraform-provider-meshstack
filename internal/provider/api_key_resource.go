package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &apiKeyResource{}
	_ resource.ResourceWithConfigure   = &apiKeyResource{}
	_ resource.ResourceWithImportState = &apiKeyResource{}
)

func NewApiKeyResource() resource.Resource {
	return &apiKeyResource{}
}

type apiKeyResource struct {
	meshApiKeyClient client.MeshApiKeyClient
}

type apiKeyResourceModel struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	Name                types.String `tfsdk:"name"`
	Authorities         types.List   `tfsdk:"authorities"`
	ExpiryDate          types.String `tfsdk:"expiry_date"`

	Uuid      types.String `tfsdk:"uuid"`
	Token     types.String `tfsdk:"token"`
	CreatedOn types.String `tfsdk:"created_on"`
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
		MarkdownDescription: "Manages a meshStack API key.\n\n" +
			"The API key token is only available after initial creation and is stored " +
			"(sensitive) in the Terraform state. It cannot be retrieved again from the API.",

		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Identifier of the workspace that owns the API key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the API key.",
			},
			"authorities": schema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Authorities (permission shortcodes) assigned to the API key.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"expiry_date": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Expiry date of the API key (ISO date, e.g. `2025-12-31`).",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the API key (Keycloak clientId).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Secret token of the API key. Only available after creation; cannot be retrieved again.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_on": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp of the API key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *apiKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authorities := extractAuthorities(ctx, data.Authorities, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.meshApiKeyClient.Create(ctx, &client.MeshApiKeyCreate{
		Metadata: client.MeshApiKeyCreateMetadata{
			Name:             data.Name.ValueString(),
			OwnedByWorkspace: data.WorkspaceIdentifier.ValueString(),
		},
		Spec: client.MeshApiKeySpec{
			Authorities: authorities,
			ExpiryDate:  data.ExpiryDate.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to create API key", err.Error())
		return
	}

	if created == nil || created.Metadata.Uuid == nil || *created.Metadata.Uuid == "" {
		resp.Diagnostics.AddError("Unable to create API key", "The API did not return a UUID.")
		return
	}

	if created.Token == nil || *created.Token == "" {
		resp.Diagnostics.AddError("Unable to create API key", "The API did not return a token.")
		return
	}

	data.Uuid = types.StringPointerValue(created.Metadata.Uuid)
	data.Token = types.StringPointerValue(created.Token)
	data.CreatedOn = types.StringValue(created.Metadata.CreatedOn)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *apiKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := data.Uuid.ValueString()
	apiKey, err := r.meshApiKeyClient.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read API key", err.Error())
		return
	}

	if apiKey == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	data.WorkspaceIdentifier = types.StringValue(apiKey.Metadata.OwnedByWorkspace)
	data.Name = types.StringValue(apiKey.Metadata.Name)
	data.CreatedOn = types.StringValue(apiKey.Metadata.CreatedOn)
	data.ExpiryDate = types.StringValue(apiKey.Spec.ExpiryDate)

	authList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Spec.Authorities)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Authorities = authList

	// Token is never returned by GET — preserve existing state value (UseStateForUnknown handles plan,
	// but we must not overwrite state here).

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *apiKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	authorities := extractAuthorities(ctx, data.Authorities, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := data.Uuid.ValueString()
	updated, err := r.meshApiKeyClient.Update(ctx, uuid, &client.MeshApiKeyCreate{
		Metadata: client.MeshApiKeyCreateMetadata{
			Name:             data.Name.ValueString(),
			OwnedByWorkspace: data.WorkspaceIdentifier.ValueString(),
		},
		Spec: client.MeshApiKeySpec{
			Authorities: authorities,
			ExpiryDate:  data.ExpiryDate.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update API key", err.Error())
		return
	}

	if updated != nil {
		data.Name = types.StringValue(updated.Metadata.Name)
		data.ExpiryDate = types.StringValue(updated.Spec.ExpiryDate)
		data.CreatedOn = types.StringValue(updated.Metadata.CreatedOn)

		authList, diags := types.ListValueFrom(ctx, types.StringType, updated.Spec.Authorities)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Authorities = authList
	}

	// Preserve token from prior state (PUT never returns it)
	var priorState apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Token = priorState.Token

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *apiKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data apiKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := data.Uuid.ValueString()
	if err := r.meshApiKeyClient.Delete(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Unable to delete API key", err.Error())
	}
}

func (r *apiKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func extractAuthorities(ctx context.Context, authList types.List, diags *diag.Diagnostics) []string {
	var authorities []string
	diags.Append(authList.ElementsAs(ctx, &authorities, false)...)
	return authorities
}
