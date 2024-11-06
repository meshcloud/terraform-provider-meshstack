package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                   = &tagDefinitionResource{}
	_ resource.ResourceWithConfigure      = &tagDefinitionResource{}
	_ resource.ResourceWithValidateConfig = &tagDefinitionResource{}
	_ resource.ResourceWithImportState    = &tagDefinitionResource{}
)

var targetKinds = []string{
	"meshProject",
	"meshWorkspace",
	"meshLandingZone",
	"meshPaymentMethod",
	"meshBuildingBlockDefinition",
}

// NewTagDefinitionResource is a helper function to simplify the provider implementation.
func NewTagDefinitionResource() resource.Resource {
	return &tagDefinitionResource{}
}

// tagDefinitionResource is the resource implementation.
type tagDefinitionResource struct {
	client *client.MeshStackProviderClient
}

// Metadata returns the resource type name.
func (r *tagDefinitionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tag_definition"
}

// Configure adds the provider configured client to the resource.
func (r *tagDefinitionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *tagDefinitionResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config tagDefinitionCreate

	diags := req.Config.Get(ctx, &config)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that metadata.name is equal to spec.target_kind.spec.key
	expectedName := fmt.Sprintf("%s.%s", config.Spec.TargetKind.ValueString(), config.Spec.Key.ValueString())
	if config.Metadata.Name.ValueString() != expectedName {
		resp.Diagnostics.AddError(
			"Invalid Name",
			fmt.Sprintf("metadata.name must be equal to spec.target_kind.spec.key. Expected: %s, Got: %s", expectedName, config.Metadata.Name.ValueString()),
		)
		return
	}

	// Validate that value_type only contains one of the value types
	valueType := config.Spec.ValueType
	count := 0
	if valueType.String != nil {
		count++
	}
	if valueType.Email != nil {
		count++
	}
	if valueType.Integer != nil {
		count++
	}
	if valueType.Number != nil {
		count++
	}
	if valueType.SingleSelect != nil {
		count++
	}
	if valueType.MultiSelect != nil {
		count++
	}

	// Check if exactly one value type is specified
	if count != 1 {
		resp.Diagnostics.AddError(
			"Invalid value type",
			"Exactly one value type must be specified: string, email, integer, number, single_select, multi_select",
		)
	}
}

// Schema defines the schema for the resource.
func (r *tagDefinitionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage tag definitions",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Tag definition datatype version",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshTagDefinition`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshTagDefinition"}...),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},

			"metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Tag definition metadata. Name of the target tag definition must be set here.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-zA-Z0-9_-]+\.[a-zA-Z0-9_-]+$`),
								"Name must be in the format 'target_kind.key'",
							),
						},
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				MarkdownDescription: "Tag definition specification.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"target_kind": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
						Validators: []validator.String{
							stringvalidator.OneOf(targetKinds...),
						},
					},
					"key": schema.StringAttribute{
						Required:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
					},
					"value_type": schema.SingleNestedAttribute{
						// TODO: this should not require a replace if fields of a value type are changed (e.g. if string.default_value is changed)
						PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplace()},
						Required:      true,
						Attributes: map[string]schema.Attribute{
							"string": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.StringAttribute{
										Optional: true,
									},
									"validation_regex": schema.StringAttribute{
										Optional: true,
									},
								},
							},
							"email": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.StringAttribute{
										Optional: true,
									},
									"validation_regex": schema.StringAttribute{
										Optional: true,
									},
								},
							},
							"integer": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.Int64Attribute{
										Optional: true,
									},
								},
							},
							"number": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.Float64Attribute{
										Optional: true,
									},
								},
							},
							"single_select": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"options": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
									"default_value": schema.StringAttribute{
										Optional: true,
									},
								},
							},
							"multi_select": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"options": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
									"default_value": schema.ListAttribute{
										ElementType: types.StringType,
										Optional:    true,
									},
								},
							},
						},
					},
					"display_name": schema.StringAttribute{Required: true},
					"description": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString(""),
					},
					"sort_order": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(0)},
					"mandatory":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"immutable":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"restricted": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
				},
			},
		},
	}
}

// These structs use Terraform types so that we can read the plan and check for unknown/null values.
type tagDefinitionCreate struct {
	ApiVersion types.String          `json:"apiVersion" tfsdk:"api_version"`
	Kind       types.String          `json:"kind" tfsdk:"kind"`
	Metadata   tagDefinitionMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       tagDefinitionSpec     `json:"spec" tfsdk:"spec"`
}

type tagDefinitionMetadata struct {
	Name types.String `json:"name" tfsdk:"name"`
}

type tagDefinitionSpec struct {
	TargetKind  types.String           `json:"targetKind" tfsdk:"target_kind"`
	Key         types.String           `json:"key" tfsdk:"key"`
	ValueType   tagDefinitionValueType `json:"valueType" tfsdk:"value_type"`
	Description types.String           `json:"description" tfsdk:"description"`
	DisplayName types.String           `json:"displayName" tfsdk:"display_name"`
	SortOrder   types.Int64            `json:"sortOrder" tfsdk:"sort_order"`
	Mandatory   types.Bool             `json:"mandatory" tfsdk:"mandatory"`
	Immutable   types.Bool             `json:"immutable" tfsdk:"immutable"`
	Restricted  types.Bool             `json:"restricted" tfsdk:"restricted"`
}

type tagDefinitionValueType struct {
	String       *tagValueString       `json:"string,omitempty" tfsdk:"string"`
	Email        *tagValueEmail        `json:"email,omitempty" tfsdk:"email"`
	Integer      *tagValueInteger      `json:"integer,omitempty" tfsdk:"integer"`
	Number       *tagValueNumber       `json:"number,omitempty" tfsdk:"number"`
	SingleSelect *tagValueSingleSelect `json:"singleSelect,omitempty" tfsdk:"single_select"`
	MultiSelect  *tagValueMultiSelect  `json:"multiSelect,omitempty" tfsdk:"multi_select"`
}

type tagValueString struct {
	DefaultValue    types.String `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex types.String `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type tagValueEmail struct {
	DefaultValue    types.String `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex types.String `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type tagValueInteger struct {
	DefaultValue types.Int64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type tagValueNumber struct {
	DefaultValue types.Float64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type tagValueSingleSelect struct {
	Options      []types.String `json:"options,omitempty" tfsdk:"options"`
	DefaultValue types.String   `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type tagValueMultiSelect struct {
	Options      []types.String `json:"options,omitempty" tfsdk:"options"`
	DefaultValue []types.String `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

// Create creates the resource and sets the initial Terraform state.
func (r *tagDefinitionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tagDefinitionCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueType := buildValueType(plan.Spec.ValueType)

	create := client.MeshTagDefinition{
		ApiVersion: plan.ApiVersion.ValueString(),
		Kind:       "meshTagDefinition",
		Metadata: client.MeshTagDefinitionMetadata{
			Name: plan.Metadata.Name.ValueString(),
		},
		Spec: client.MeshTagDefinitionSpec{
			TargetKind:  plan.Spec.TargetKind.ValueString(),
			Key:         plan.Spec.Key.ValueString(),
			ValueType:   valueType,
			Description: plan.Spec.Description.ValueString(),
			DisplayName: plan.Spec.DisplayName.ValueString(),
			SortOrder:   plan.Spec.SortOrder.ValueInt64(),
			Mandatory:   plan.Spec.Mandatory.ValueBool(),
			Immutable:   plan.Spec.Immutable.ValueBool(),
			Restricted:  plan.Spec.Restricted.ValueBool(),
		},
	}

	tagDefinition, err := r.client.CreateTagDefinition(&create)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating tag definition",
			"Could not create tag definition, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, tagDefinition)
	resp.Diagnostics.Append(diags...)
}

func extractStringValues(values []basetypes.StringValue) []string {
	result := make([]string, len(values))
	for i, v := range values {
		result[i] = v.ValueString()
	}
	return result
}

func buildValueType(valueType tagDefinitionValueType) client.MeshTagDefinitionValueType {
	var result client.MeshTagDefinitionValueType

	if valueType.String != nil {
		result.String = &client.TagValueString{
			DefaultValue:    valueType.String.DefaultValue.ValueString(),
			ValidationRegex: valueType.String.ValidationRegex.ValueString(),
		}
	}

	if valueType.Email != nil {
		result.Email = &client.TagValueEmail{
			DefaultValue:    valueType.Email.DefaultValue.ValueString(),
			ValidationRegex: valueType.Email.ValidationRegex.ValueString(),
		}
	}

	if valueType.Integer != nil {
		result.Integer = &client.TagValueInteger{
			DefaultValue: valueType.Integer.DefaultValue.ValueInt64(),
		}
	}

	if valueType.Number != nil {
		result.Number = &client.TagValueNumber{
			DefaultValue: valueType.Number.DefaultValue.ValueFloat64(),
		}
	}

	if valueType.SingleSelect != nil {
		result.SingleSelect = &client.TagValueSingleSelect{
			Options:      extractStringValues(valueType.SingleSelect.Options),
			DefaultValue: valueType.SingleSelect.DefaultValue.ValueString(),
		}
	}

	if valueType.MultiSelect != nil {
		result.MultiSelect = &client.TagValueMultiSelect{
			Options:      extractStringValues(valueType.MultiSelect.Options),
			DefaultValue: extractStringValues(valueType.MultiSelect.DefaultValue),
		}
	}

	return result
}

// Read refreshes the Terraform state with the latest data.
func (r *tagDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state client.MeshTagDefinition

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagDefinition, err := r.client.ReadTagDefinition(state.Metadata.Name)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read tag definition", err.Error())
	}

	if tagDefinition == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, tagDefinition)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tagDefinitionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan tagDefinitionCreate

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that metadata.name is equal to spec.target_kind.spec.key
	expectedName := fmt.Sprintf("%s.%s", plan.Spec.TargetKind.ValueString(), plan.Spec.Key.ValueString())
	if plan.Metadata.Name.ValueString() != expectedName {
		resp.Diagnostics.AddError(
			"Invalid Name",
			fmt.Sprintf("metadata.name must be equal to spec.target_kind.spec.key. Expected: %s, Got: %s", expectedName, plan.Metadata.Name.ValueString()),
		)
		return
	}

	valueType := buildValueType(plan.Spec.ValueType)

	update := client.MeshTagDefinition{
		ApiVersion: plan.ApiVersion.ValueString(),
		Kind:       "meshTagDefinition",
		Metadata: client.MeshTagDefinitionMetadata{
			Name: plan.Metadata.Name.ValueString(),
		},
		Spec: client.MeshTagDefinitionSpec{
			TargetKind:  plan.Spec.TargetKind.ValueString(),
			Key:         plan.Spec.Key.ValueString(),
			ValueType:   valueType,
			Description: plan.Spec.Description.ValueString(),
			DisplayName: plan.Spec.DisplayName.ValueString(),
			SortOrder:   plan.Spec.SortOrder.ValueInt64(),
			Mandatory:   plan.Spec.Mandatory.ValueBool(),
			Immutable:   plan.Spec.Immutable.ValueBool(),
			Restricted:  plan.Spec.Restricted.ValueBool(),
		},
	}

	tagDefinition, err := r.client.UpdateTagDefinition(&update)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating tag definition",
			"Could not update tag definition, unexpected error: "+err.Error(),
		)
		return
	}

	diags = resp.State.Set(ctx, tagDefinition)
	resp.Diagnostics.Append(diags...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *tagDefinitionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state client.MeshTagDefinition

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTagDefinition(state.Metadata.Name)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error deleting tag definition",
			"Could not delete tag definition, unexpected error: "+err.Error(),
		)
		return
	}
}

// ImportState imports the resource state.
func (r *tagDefinitionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Use the resource ID as the name of the tag definition
	tagDefinitionName := req.ID

	// Read the tag definition from the provider
	tagDefinition, err := r.client.ReadTagDefinition(tagDefinitionName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing tag definition",
			"Could not import tag definition, unexpected error: "+err.Error(),
		)
		return
	}

	if tagDefinition == nil {
		resp.Diagnostics.AddError(
			"Error importing tag definition",
			"Tag definition not found",
		)
		return
	}

	// Set the state with the imported tag definition
	diags := resp.State.Set(ctx, tagDefinition)
	resp.Diagnostics.Append(diags...)
}
