package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	eschema "github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

const apiKeyPrivateStateKey = "api_key"

var (
	_ ephemeral.EphemeralResource              = &apiKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithConfigure = &apiKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithRenew     = &apiKeyEphemeralResource{}
	_ ephemeral.EphemeralResourceWithClose     = &apiKeyEphemeralResource{}
)

type apiKeyEphemeralResource struct {
	meshApiKeyClient client.MeshApiKeyClient
}

type apiKeyEphemeralModel struct {
	WorkspaceIdentifier types.String `tfsdk:"workspace_identifier"`
	Name                types.String `tfsdk:"name"`
	Authorities         types.List   `tfsdk:"authorities"`
	ExpiryDate          types.String `tfsdk:"expiry_date"`

	Uuid      types.String `tfsdk:"uuid"`
	Token     types.String `tfsdk:"token"`
	CreatedOn types.String `tfsdk:"created_on"`
}

type apiKeyEphemeralPrivateState struct {
	Uuid                string   `json:"uuid"`
	WorkspaceIdentifier string   `json:"workspace_identifier"`
	Name                string   `json:"name"`
	Authorities         []string `json:"authorities"`
	ExpiryDate          string   `json:"expiry_date"`
}

type privateStateReader interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}

func NewApiKeyEphemeralResource() ephemeral.EphemeralResource {
	return &apiKeyEphemeralResource{}
}

func (r *apiKeyEphemeralResource) Metadata(_ context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *apiKeyEphemeralResource) Configure(_ context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(providerClient client.Client) {
		r.meshApiKeyClient = providerClient.ApiKey
	})...)
}

func (r *apiKeyEphemeralResource) Schema(_ context.Context, _ ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = eschema.Schema{
		MarkdownDescription: "Creates an ephemeral API key and exposes its token during a Terraform run.",
		Attributes: map[string]eschema.Attribute{
			"workspace_identifier": eschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Identifier of the workspace that owns the API key.",
			},
			"name": eschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name of the API key.",
			},
			"authorities": eschema.ListAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Authorities assigned to the API key.",
			},
			"expiry_date": eschema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Expiry date of the API key in RFC3339 format.",
			},
			"uuid": eschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "UUID of the created API key.",
			},
			"token": eschema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Token of the created API key.",
			},
			"created_on": eschema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Creation timestamp of the API key.",
			},
		},
	}
}

func (r *apiKeyEphemeralResource) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	if r.meshApiKeyClient == nil {
		resp.Diagnostics.AddError(
			"Missing API key client",
			"The provider API key client is not configured.",
		)
		return
	}

	var data apiKeyEphemeralModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	expiryDate, err := time.Parse(time.RFC3339, data.ExpiryDate.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid expiry_date",
			fmt.Sprintf("Expected expiry_date in RFC3339 format, got %q: %s", data.ExpiryDate.ValueString(), err),
		)
		return
	}

	var authorities []string
	resp.Diagnostics.Append(data.Authorities.ElementsAs(ctx, &authorities, false)...)
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
		resp.Diagnostics.AddError(
			"Unable to create ephemeral API key",
			err.Error(),
		)
		return
	}

	if created == nil || created.Metadata.Uuid == nil || *created.Metadata.Uuid == "" {
		resp.Diagnostics.AddError(
			"Unable to create ephemeral API key",
			"The API did not return a UUID for the created API key.",
		)
		return
	}

	if created.Token == nil || *created.Token == "" {
		resp.Diagnostics.AddError(
			"Unable to create ephemeral API key",
			"The API did not return a token for the created API key.",
		)
		return
	}

	data.Uuid = types.StringPointerValue(created.Metadata.Uuid)
	data.Token = types.StringPointerValue(created.Token)
	data.CreatedOn = types.StringValue(created.Metadata.CreatedOn)

	resp.Diagnostics.Append(resp.Result.Set(ctx, data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	privateState := apiKeyEphemeralPrivateState{
		Uuid:                *created.Metadata.Uuid,
		WorkspaceIdentifier: data.WorkspaceIdentifier.ValueString(),
		Name:                data.Name.ValueString(),
		Authorities:         authorities,
		ExpiryDate:          data.ExpiryDate.ValueString(),
	}

	privateStateBytes, err := json.Marshal(privateState)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to persist ephemeral API key state",
			err.Error(),
		)
		return
	}

	if resp.Private == nil {
		resp.Diagnostics.AddError(
			"Unable to persist ephemeral API key state",
			"Private state is not initialized.",
		)
		return
	}

	resp.Diagnostics.Append(resp.Private.SetKey(ctx, apiKeyPrivateStateKey, privateStateBytes)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.RenewAt = expiryDate.Add(-time.Minute)
}

func (r *apiKeyEphemeralResource) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
	if r.meshApiKeyClient == nil {
		resp.Diagnostics.AddError(
			"Missing API key client",
			"The provider API key client is not configured.",
		)
		return
	}

	privateState := loadApiKeyPrivateState(ctx, req.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || privateState == nil {
		return
	}

	_, err := r.meshApiKeyClient.Update(ctx, privateState.Uuid, &client.MeshApiKeyCreate{
		Metadata: client.MeshApiKeyCreateMetadata{
			Name:             privateState.Name,
			OwnedByWorkspace: privateState.WorkspaceIdentifier,
		},
		Spec: client.MeshApiKeySpec{
			Authorities: privateState.Authorities,
			ExpiryDate:  privateState.ExpiryDate,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to renew ephemeral API key",
			err.Error(),
		)
		return
	}

	expiryDate, err := time.Parse(time.RFC3339, privateState.ExpiryDate)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to renew ephemeral API key",
			fmt.Sprintf("Stored expiry_date %q is invalid: %s", privateState.ExpiryDate, err),
		)
		return
	}

	nextRenewAt := expiryDate.Add(-time.Minute)
	if nextRenewAt.After(time.Now()) {
		resp.RenewAt = nextRenewAt
	}
}

func (r *apiKeyEphemeralResource) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	if r.meshApiKeyClient == nil {
		resp.Diagnostics.AddError(
			"Missing API key client",
			"The provider API key client is not configured.",
		)
		return
	}

	privateState := loadApiKeyPrivateState(ctx, req.Private, &resp.Diagnostics)
	if resp.Diagnostics.HasError() || privateState == nil {
		return
	}

	if err := r.meshApiKeyClient.Delete(ctx, privateState.Uuid); err != nil {
		resp.Diagnostics.AddError(
			"Unable to close ephemeral API key",
			err.Error(),
		)
	}
}

func loadApiKeyPrivateState(ctx context.Context, privateStateReader privateStateReader, diags *diag.Diagnostics) *apiKeyEphemeralPrivateState {
	if privateStateReader == nil {
		return nil
	}

	rawPrivateState, privateStateDiags := privateStateReader.GetKey(ctx, apiKeyPrivateStateKey)
	diags.Append(privateStateDiags...)
	if diags.HasError() || len(rawPrivateState) == 0 {
		return nil
	}

	var privateState apiKeyEphemeralPrivateState
	if err := json.Unmarshal(rawPrivateState, &privateState); err != nil {
		diags.AddError(
			"Unable to decode ephemeral API key state",
			err.Error(),
		)
		return nil
	}

	if privateState.Uuid == "" {
		diags.AddError(
			"Unable to decode ephemeral API key state",
			"Missing UUID in ephemeral API key state.",
		)
		return nil
	}

	return &privateState
}
