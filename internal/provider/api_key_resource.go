package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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
	DisplayName         types.String `tfsdk:"display_name"`
	Authorities         types.List   `tfsdk:"authorities"`
	ExpiresAt           types.String `tfsdk:"expires_at"`

	Uuid  types.String `tfsdk:"uuid"`
	Token types.String `tfsdk:"token"`
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
			"The API key token is returned in `status.token` after creation and after updates that " +
			"rotate the secret (e.g. changing `expires_at`). It is stored (sensitive) in the Terraform state.",

		Attributes: map[string]schema.Attribute{
			"workspace_identifier": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Identifier of the workspace that owns the API key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"display_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name of the API key.",
			},
			"authorities": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Authorities (permission shortcodes) assigned to the API key. " +
					"Valid values: " + client.WorkspacePermissions.Markdown() + " (workspace-scoped), " +
					client.AdminPermissions.Markdown() + " (admin-scoped).",
			},
			"expires_at": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Expiry date of the API key (ISO date, e.g. `2025-12-31`). Changing this rotates the secret and a new token is returned.",
			},
			"uuid": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the API key (Keycloak clientId).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Secret token of the API key. Available after creation and after secret rotation (when `expires_at` changes).",
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

	if diags := validateAuthorities(authorities); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	expiresAt := data.ExpiresAt.ValueString()
	created, err := r.meshApiKeyClient.Create(ctx, &client.MeshApiKeyCreate{
		Metadata: client.MeshApiKeyCreateMetadata{
			OwnedByWorkspace: data.WorkspaceIdentifier.ValueString(),
		},
		Spec: client.MeshApiKeySpec{
			DisplayName: data.DisplayName.ValueString(),
			Authorities: authorities,
			ExpiresAt:   &expiresAt,
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

	data.Uuid = types.StringPointerValue(created.Metadata.Uuid)
	data.Token = extractToken(created)

	// Sync back server-authoritative fields
	data.DisplayName = types.StringValue(created.Spec.DisplayName)
	if created.Spec.ExpiresAt != nil {
		data.ExpiresAt = types.StringValue(*created.Spec.ExpiresAt)
	}
	authList, diags := types.ListValueFrom(ctx, types.StringType, created.Spec.Authorities)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Authorities = authList

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
	data.DisplayName = types.StringValue(apiKey.Spec.DisplayName)
	if apiKey.Spec.ExpiresAt != nil {
		data.ExpiresAt = types.StringValue(*apiKey.Spec.ExpiresAt)
	}

	authList, diags := types.ListValueFrom(ctx, types.StringType, apiKey.Spec.Authorities)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Authorities = authList

	// Token is never returned by GET — preserve existing state value.

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

	if diags := validateAuthorities(authorities); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	uuid := data.Uuid.ValueString()
	expiresAt := data.ExpiresAt.ValueString()
	updated, err := r.meshApiKeyClient.Update(ctx, uuid, &client.MeshApiKeyCreate{
		Metadata: client.MeshApiKeyCreateMetadata{
			Uuid:             &uuid,
			OwnedByWorkspace: data.WorkspaceIdentifier.ValueString(),
		},
		Spec: client.MeshApiKeySpec{
			DisplayName: data.DisplayName.ValueString(),
			Authorities: authorities,
			ExpiresAt:   &expiresAt,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Unable to update API key", err.Error())
		return
	}

	if updated != nil {
		data.DisplayName = types.StringValue(updated.Spec.DisplayName)
		if updated.Spec.ExpiresAt != nil {
			data.ExpiresAt = types.StringValue(*updated.Spec.ExpiresAt)
		}

		authList, diags := types.ListValueFrom(ctx, types.StringType, updated.Spec.Authorities)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Authorities = authList

		// If the API rotated the secret (expiresAt changed), capture the new token.
		token := extractToken(updated)
		if !token.IsNull() {
			data.Token = token
		} else {
			// Preserve token from prior state when no rotation occurred.
			var priorState apiKeyResourceModel
			resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Token = priorState.Token
		}
	}

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

func extractToken(apiKey *client.MeshApiKey) types.String {
	if apiKey.Status != nil && apiKey.Status.Token != nil && *apiKey.Status.Token != "" {
		return types.StringValue(*apiKey.Status.Token)
	}
	return types.StringNull()
}

func validateAuthorities(authorities []string) diag.Diagnostics {
	var diags diag.Diagnostics
	validPermissions := client.AllApiKeyPermissions()
	validSet := make(map[string]bool, len(validPermissions))
	for _, p := range validPermissions {
		validSet[p] = true
	}

	var invalid []string
	for _, a := range authorities {
		if !validSet[a] {
			invalid = append(invalid, a)
		}
	}

	if len(invalid) > 0 {
		diags.AddError(
			"Invalid authorities",
			fmt.Sprintf(
				"The following authorities are not valid API key permissions: %s. "+
					"Valid permissions are: %s",
				strings.Join(invalid, ", "),
				strings.Join(validPermissions, ", "),
			),
		)
	}
	return diags
}
