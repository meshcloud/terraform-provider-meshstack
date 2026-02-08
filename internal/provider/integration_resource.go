package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/client/types/enum"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
	"github.com/meshcloud/terraform-provider-meshstack/internal/types/secret"
)

var (
	_ resource.Resource                = &integrationResource{}
	_ resource.ResourceWithConfigure   = &integrationResource{}
	_ resource.ResourceWithImportState = &integrationResource{}
	_ resource.ResourceWithModifyPlan  = &integrationResource{}
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

type integrationModel struct {
	client.MeshIntegration
	Ref struct {
		Kind string `tfsdk:"kind"`
		Uuid string `tfsdk:"uuid"`
	} `tfsdk:"ref"`
}

var IntegrationConfigTypeToBBDImplType = map[enum.Entry[client.MeshIntegrationConfigType]]enum.Entry[client.MeshBuildingBlockImplementationType]{
	client.MeshIntegrationConfigTypeGithub:      client.MeshBuildingBlockImplementationTypeGithubWorkflows,
	client.MeshIntegrationConfigTypeGitlab:      client.MeshBuildingBlockImplementationTypeGitlabPipeline,
	client.MeshIntegrationConfigTypeAzureDevops: client.MeshBuildingBlockImplementationTypeAzureDevOpsPipeline,
}

func (model integrationModel) ToClientDto() client.MeshIntegration {
	setRunnerRefIfNotNil := func(configType enum.Entry[client.MeshIntegrationConfigType], runnerRef **client.BuildingBlockRunnerRef) {
		if *runnerRef == nil {
			*runnerRef = getSharedBuildingBlockRunnerRef(IntegrationConfigTypeToBBDImplType[configType])
		}
	}
	switch configType := model.Spec.Config.InferTypeFromNonNilField(); configType {
	case client.MeshIntegrationConfigTypeGithub:
		setRunnerRefIfNotNil(configType, &model.Spec.Config.Github.RunnerRef)
	case client.MeshIntegrationConfigTypeGitlab:
		setRunnerRefIfNotNil(configType, &model.Spec.Config.Gitlab.RunnerRef)
	case client.MeshIntegrationConfigTypeAzureDevops:
		setRunnerRefIfNotNil(configType, &model.Spec.Config.AzureDevops.RunnerRef)
	}
	return model.MeshIntegration
}

func (model *integrationModel) SetFromClientDto(dto *client.MeshIntegration) {
	model.MeshIntegration = *dto
	model.Ref.Kind = "meshIntegration"
	model.Ref.Uuid = *dto.Metadata.Uuid
}

func (r *integrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan := generic.Get[integrationModel](ctx, req.Plan, &resp.Diagnostics,
		secret.WithConverterSupport(ctx, req.Config, req.Plan, nil).Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	createdDto, err := r.integrationClient.Create(ctx, plan.ToClientDto())
	if err != nil {
		resp.Diagnostics.AddError("Error creating meshIntegration", err.Error())
		return
	}
	plan.SetFromClientDto(createdDto)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, secret.WithConverterSupport(ctx, req.Config, req.Plan, nil)...)...)
}

func (r *integrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
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
	var state integrationModel
	state.SetFromClientDto(readDto)
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, state, secret.WithConverterSupport(ctx, nil, nil, req.State)...)...)
}

func (r *integrationResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		// do nothing in case of delete
		return
	}
	secret.WalkSecretPathsIn(req.Plan.Raw, &resp.Diagnostics, func(attributePath path.Path, diags *diag.Diagnostics) {
		secret.SetHashToUnknownIfVersionChanged(ctx, req.Plan, req.State, &resp.Plan)(attributePath, diags)
	})
}

func (r *integrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan := generic.Get[integrationModel](ctx, req.Plan, &resp.Diagnostics,
		secret.WithConverterSupport(ctx, req.Config, req.Plan, req.State).Append(generic.WithSetUnknownValueToZero())...)
	if resp.Diagnostics.HasError() {
		return
	}
	updatedDto, err := r.integrationClient.Update(ctx, plan.ToClientDto())
	if err != nil {
		resp.Diagnostics.AddError("Error updating meshIntegration", fmt.Sprintf("Updating integration '%s' failed: %s", *plan.Metadata.Uuid, err.Error()))
		return
	}
	plan.MeshIntegration = *updatedDto
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, plan, secret.WithConverterSupport(ctx, req.Config, req.Plan, nil)...)...)
}

func (r *integrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	uuid := generic.GetAttribute[string](ctx, req.State, path.Root("metadata").AtName("uuid"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.integrationClient.Delete(ctx, uuid); err != nil {
		resp.Diagnostics.AddError("Error deleting meshIntegration", fmt.Sprintf("Deleting integration '%s' failed: %s", uuid, err.Error()))
		return
	}
}

func (r *integrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("metadata").AtName("uuid"), req, resp)
}
