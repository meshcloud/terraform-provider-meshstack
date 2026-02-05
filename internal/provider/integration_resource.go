package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
)

func NewIntegrationResource() resource.Resource {
	return &integrationResource{}
}

type integrationResource struct {
	integrationClient client.MeshIntegrationClient
}

func (r *integrationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *integrationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		r.integrationClient = client.Integration
	})...)
}

func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integration
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createRequest := plan.ToClientDto(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	createdDto, err := r.integrationClient.Create(ctx, createRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error creating meshIntegration", err.Error())
		return
	}
	plan.SetFromClientDto(createdDto, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integration
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Metadata.Uuid.Get(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	readDto, err := r.integrationClient.Read(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read integration", fmt.Sprintf("Reading integration '%s' failed: %s", uuid, err.Error()))
		return
	} else if readDto == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	state.SetFromClientDto(readDto, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan integration
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateRequest := plan.ToClientDto(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	updatedDto, err := r.integrationClient.Update(ctx, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError("Error updating meshIntegration", fmt.Sprintf("Updating integration '%s' failed: %s", *updateRequest.Metadata.Uuid, err.Error()))
		return
	}
	plan.SetFromClientDto(updatedDto, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integration
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.Metadata.Uuid.Get(&resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.integrationClient.Delete(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting meshIntegration", fmt.Sprintf("Deleting integration '%s' failed: %s", uuid, err.Error()))
		return
	}
}

func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
