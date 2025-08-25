package provider

import (
	"context"
	"fmt"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &platformResource{}
	_ resource.ResourceWithConfigure   = &platformResource{}
	_ resource.ResourceWithImportState = &platformResource{}
)

// NewPlatformResource is a helper function to simplify the provider implementation.
func NewPlatformResource() resource.Resource {
	return &platformResource{}
}

// platformResource is the resource implementation.
type platformResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *platformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_platform"
}

// Configure adds the provider configured client to the resource.
func (r *platformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Schema defines the schema for the resource.
func (r *platformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Single platform by name.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Platform datatype version",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPlatform`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPlatform"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Platform metadata. Name of the target Platform must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
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
				MarkdownDescription: "Platform specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the platform.",
						Required:            true,
					},
					"platform_type": schema.StringAttribute{
						MarkdownDescription: "Type of the platform (e.g., 'azure', 'aws', 'gcp').",
						Required:            true,
					},
					"configuration": schema.MapAttribute{
						MarkdownDescription: "Platform-specific configuration parameters.",
						ElementType:         types.StringType,
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapNull(types.StringType)),
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the platform.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Optional:            true,
						Computed:            true,
						Default:             mapdefault.StaticValue(types.MapNull(types.ListType{ElemType: types.StringType})),
					},
				},
			},
		},
	}
}

// These structs use Terraform types so that we can read the plan and check for unknown/null values.
type platformCreate struct {
	ApiVersion types.String       `json:"apiVersion" tfsdk:"api_version"`
	Kind       types.String       `json:"kind" tfsdk:"kind"`
	Metadata   platformMetadata   `json:"metadata" tfsdk:"metadata"`
	Spec       platformSpec       `json:"spec" tfsdk:"spec"`
}

type platformMetadata struct {
	Name      types.String `json:"name" tfsdk:"name"`
	CreatedOn types.String `json:"createdOn" tfsdk:"created_on"`
	DeletedOn types.String `json:"deletedOn" tfsdk:"deleted_on"`
}

type platformSpec struct {
	DisplayName   types.String `json:"displayName" tfsdk:"display_name"`
	PlatformType  types.String `json:"platformType" tfsdk:"platform_type"`
	Configuration types.Map    `json:"configuration" tfsdk:"configuration"`
	Tags          types.Map    `json:"tags" tfsdk:"tags"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *platformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan platformCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuration := make(map[string]interface{})
	if !plan.Spec.Configuration.IsNull() {
		var configMap map[string]string
		diags = plan.Spec.Configuration.ElementsAs(ctx, &configMap, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range configMap {
			configuration[k] = v
		}
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		diags = plan.Spec.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	create := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{
			Name: plan.Metadata.Name.ValueString(),
		},
		Spec: client.MeshPlatformSpec{
			DisplayName:   plan.Spec.DisplayName.ValueString(),
			PlatformType:  plan.Spec.PlatformType.ValueString(),
			Configuration: configuration,
			Tags:          tags,
		},
	}

	platform, err := r.client.CreatePlatform(&create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating platform",
			"Could not create platform, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert configuration back to string map for state
	configurationMap := make(map[string]string)
	for k, v := range platform.Spec.Configuration {
		if strVal, ok := v.(string); ok {
			configurationMap[k] = strVal
		} else {
			configurationMap[k] = fmt.Sprintf("%v", v)
		}
	}

	planResult := platformCreate{
		ApiVersion: types.StringValue(platform.ApiVersion),
		Kind:       types.StringValue(platform.Kind),
		Metadata: platformMetadata{
			Name:      types.StringValue(platform.Metadata.Name),
			CreatedOn: types.StringValue(platform.Metadata.CreatedOn),
			DeletedOn: types.StringNull(),
		},
		Spec: platformSpec{
			DisplayName:  types.StringValue(platform.Spec.DisplayName),
			PlatformType: types.StringValue(platform.Spec.PlatformType),
		},
	}

	if len(configurationMap) > 0 {
		configMap, diags := types.MapValueFrom(ctx, types.StringType, configurationMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		planResult.Spec.Configuration = configMap
	} else {
		planResult.Spec.Configuration = types.MapNull(types.StringType)
	}

	if len(platform.Spec.Tags) > 0 {
		tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, platform.Spec.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		planResult.Spec.Tags = tagsMap
	} else {
		planResult.Spec.Tags = types.MapNull(types.ListType{ElemType: types.StringType})
	}

	if platform.Metadata.DeletedOn != nil {
		planResult.Metadata.DeletedOn = types.StringValue(*platform.Metadata.DeletedOn)
	}

	diags = resp.State.Set(ctx, planResult)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *platformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state platformCreate

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	platform, err := r.client.ReadPlatform(state.Metadata.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading platform",
			"Could not read platform, unexpected error: "+err.Error(),
		)
		return
	}

	if platform == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// Convert configuration back to string map for state
	configurationMap := make(map[string]string)
	for k, v := range platform.Spec.Configuration {
		if strVal, ok := v.(string); ok {
			configurationMap[k] = strVal
		} else {
			configurationMap[k] = fmt.Sprintf("%v", v)
		}
	}

	updatedState := platformCreate{
		ApiVersion: types.StringValue(platform.ApiVersion),
		Kind:       types.StringValue(platform.Kind),
		Metadata: platformMetadata{
			Name:      types.StringValue(platform.Metadata.Name),
			CreatedOn: types.StringValue(platform.Metadata.CreatedOn),
			DeletedOn: types.StringNull(),
		},
		Spec: platformSpec{
			DisplayName:  types.StringValue(platform.Spec.DisplayName),
			PlatformType: types.StringValue(platform.Spec.PlatformType),
		},
	}

	if len(configurationMap) > 0 {
		configMap, diags := types.MapValueFrom(ctx, types.StringType, configurationMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updatedState.Spec.Configuration = configMap
	} else {
		updatedState.Spec.Configuration = types.MapNull(types.StringType)
	}

	if len(platform.Spec.Tags) > 0 {
		tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, platform.Spec.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		updatedState.Spec.Tags = tagsMap
	} else {
		updatedState.Spec.Tags = types.MapNull(types.ListType{ElemType: types.StringType})
	}

	if platform.Metadata.DeletedOn != nil {
		updatedState.Metadata.DeletedOn = types.StringValue(*platform.Metadata.DeletedOn)
	}

	diags = resp.State.Set(ctx, updatedState)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *platformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan platformCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	configuration := make(map[string]interface{})
	if !plan.Spec.Configuration.IsNull() {
		var configMap map[string]string
		diags = plan.Spec.Configuration.ElementsAs(ctx, &configMap, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		for k, v := range configMap {
			configuration[k] = v
		}
	}

	tags := make(map[string][]string)
	if !plan.Spec.Tags.IsNull() {
		diags = plan.Spec.Tags.ElementsAs(ctx, &tags, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	update := client.MeshPlatformCreate{
		Metadata: client.MeshPlatformCreateMetadata{
			Name: plan.Metadata.Name.ValueString(),
		},
		Spec: client.MeshPlatformSpec{
			DisplayName:   plan.Spec.DisplayName.ValueString(),
			PlatformType:  plan.Spec.PlatformType.ValueString(),
			Configuration: configuration,
			Tags:          tags,
		},
	}

	platform, err := r.client.UpdatePlatform(plan.Metadata.Name.ValueString(), &update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating platform",
			"Could not update platform, unexpected error: "+err.Error(),
		)
		return
	}

	// Convert configuration back to string map for state
	configurationMap := make(map[string]string)
	for k, v := range platform.Spec.Configuration {
		if strVal, ok := v.(string); ok {
			configurationMap[k] = strVal
		} else {
			configurationMap[k] = fmt.Sprintf("%v", v)
		}
	}

	planResult := platformCreate{
		ApiVersion: types.StringValue(platform.ApiVersion),
		Kind:       types.StringValue(platform.Kind),
		Metadata: platformMetadata{
			Name:      types.StringValue(platform.Metadata.Name),
			CreatedOn: types.StringValue(platform.Metadata.CreatedOn),
			DeletedOn: types.StringNull(),
		},
		Spec: platformSpec{
			DisplayName:  types.StringValue(platform.Spec.DisplayName),
			PlatformType: types.StringValue(platform.Spec.PlatformType),
		},
	}

	if len(configurationMap) > 0 {
		configMap, diags := types.MapValueFrom(ctx, types.StringType, configurationMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		planResult.Spec.Configuration = configMap
	} else {
		planResult.Spec.Configuration = types.MapNull(types.StringType)
	}

	if len(platform.Spec.Tags) > 0 {
		tagsMap, diags := types.MapValueFrom(ctx, types.ListType{ElemType: types.StringType}, platform.Spec.Tags)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		planResult.Spec.Tags = tagsMap
	} else {
		planResult.Spec.Tags = types.MapNull(types.ListType{ElemType: types.StringType})
	}

	if platform.Metadata.DeletedOn != nil {
		planResult.Metadata.DeletedOn = types.StringValue(*platform.Metadata.DeletedOn)
	}

	diags = resp.State.Set(ctx, planResult)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *platformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state platformCreate

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeletePlatform(state.Metadata.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting platform",
			"Could not delete platform, unexpected error: "+err.Error(),
		)
		return
	}
}

func (r *platformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// The ID should be the platform name
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("metadata").AtName("name"), req.ID)...)
}