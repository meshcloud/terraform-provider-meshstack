package provider

import (
	"context"
	"fmt"
	"slices"

	"github.com/meshcloud/terraform-provider-meshstack/client"
	"github.com/meshcloud/terraform-provider-meshstack/internal/modifiers/tagdefinitionmodifier"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	var spec tagDefinitionSpec

	diags := req.Config.GetAttribute(ctx, path.Root("spec"), &spec)

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that value_type only contains one of the value types
	valueType := spec.ValueType
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

	if valueType.SingleSelect != nil && !valueType.SingleSelect.DefaultValue.IsNull() && !valueType.SingleSelect.DefaultValue.IsUnknown() {
		defaultValue := valueType.SingleSelect.DefaultValue.ValueString()
		options := extractStringValues(valueType.SingleSelect.Options)
		if !slices.Contains(options, defaultValue) {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("value_type").AtName("single_select").AtName("default_value"),
				"Invalid default value",
				fmt.Sprintf("Default value %v must be one of the available options: %v", defaultValue, options),
			)
		}
	}

	if valueType.MultiSelect != nil && valueType.MultiSelect.DefaultValue != nil && len(valueType.MultiSelect.DefaultValue) > 0 {
		defaultValues := extractStringValues(valueType.MultiSelect.DefaultValue)
		options := extractStringValues(valueType.MultiSelect.Options)

		for _, dv := range defaultValues {
			if !slices.Contains(options, dv) {
				resp.Diagnostics.AddAttributeError(
					path.Root("spec").AtName("value_type").AtName("multi_select").AtName("default_value"),
					"Invalid default value",
					fmt.Sprintf("All default values %v must be from the available options: %v", defaultValues, options),
				)
			}
		}
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
				MarkdownDescription: "Tag definition metadata. Name of the target tag definition must be `target_kind.key` and will be set automatically.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Computed:      true,
						PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
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
						PlanModifiers: []planmodifier.Object{objectplanmodifier.RequiresReplaceIf(
							tagdefinitionmodifier.ReplaceIfValueTypeChanges,
							"resource will be replaced if value_type changes (e.g. integer to string), but not when key values change (e.g. integer.default_value = 3 to integer.default_value = 5)",
							"resource will be replaced if value_type changes (e.g. integer to string), but not when key values change (e.g. integer.default_value = 3 to integer.default_value = 5)",
						)},
						Required: true,
						Attributes: map[string]schema.Attribute{
							"string": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value":    schema.StringAttribute{Optional: true},
									"validation_regex": schema.StringAttribute{Optional: true},
								},
							},
							"email": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value":    schema.StringAttribute{Optional: true},
									"validation_regex": schema.StringAttribute{Optional: true},
								},
							},
							"integer": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.Int64Attribute{Optional: true},
								},
							},
							"number": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"default_value": schema.Float64Attribute{Optional: true},
								},
							},
							"single_select": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"options": schema.ListAttribute{
										ElementType: types.StringType,
										Required:    true,
										Validators: []validator.List{
											listvalidator.SizeAtLeast(1),
										},
									},
									"default_value": schema.StringAttribute{Optional: true},
								},
							},
							"multi_select": schema.SingleNestedAttribute{
								Optional: true,
								Attributes: map[string]schema.Attribute{
									"options": schema.ListAttribute{
										ElementType: types.StringType,
										Required:    true,
										Validators: []validator.List{
											listvalidator.SizeAtLeast(1),
										},
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
					"sort_order": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(0),
					},
					"mandatory": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
					"immutable": schema.BoolAttribute{
						Optional: true, Computed: true,
						Default: booldefault.StaticBool(false),
					},
					"restricted": schema.BoolAttribute{
						Optional: true,
						Computed: true,
						Default:  booldefault.StaticBool(false),
					},
					"replication_key": schema.StringAttribute{
						Optional: true,
					},
				},
			},
		},
	}
}

// These structs use Terraform types so that we can read the plan and check for unknown/null values.
type tagDefinitionSpec struct {
	TargetKind     types.String           `json:"targetKind" tfsdk:"target_kind"`
	Key            types.String           `json:"key" tfsdk:"key"`
	ValueType      tagDefinitionValueType `json:"valueType" tfsdk:"value_type"`
	Description    types.String           `json:"description" tfsdk:"description"`
	DisplayName    types.String           `json:"displayName" tfsdk:"display_name"`
	SortOrder      types.Int64            `json:"sortOrder" tfsdk:"sort_order"`
	Mandatory      types.Bool             `json:"mandatory" tfsdk:"mandatory"`
	Immutable      types.Bool             `json:"immutable" tfsdk:"immutable"`
	Restricted     types.Bool             `json:"restricted" tfsdk:"restricted"`
	ReplicationKey types.String           `json:"replicationKey" tfsdk:"replication_key"`
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
	var spec tagDefinitionSpec

	diags := req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueType := buildValueType(spec.ValueType)

	name := spec.TargetKind.ValueString() + "." + spec.Key.ValueString()

	create := client.MeshTagDefinition{
		ApiVersion: client.API_VERSION_TAG_DEFINITION,
		Kind:       "meshTagDefinition",
		Metadata: client.MeshTagDefinitionMetadata{
			Name: name,
		},
		Spec: client.MeshTagDefinitionSpec{
			TargetKind:     spec.TargetKind.ValueString(),
			Key:            spec.Key.ValueString(),
			ValueType:      valueType,
			Description:    spec.Description.ValueString(),
			DisplayName:    spec.DisplayName.ValueString(),
			SortOrder:      spec.SortOrder.ValueInt64(),
			Mandatory:      spec.Mandatory.ValueBool(),
			Immutable:      spec.Immutable.ValueBool(),
			Restricted:     spec.Restricted.ValueBool(),
			ReplicationKey: spec.ReplicationKey.ValueStringPointer(),
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
			ValidationRegex: valueType.String.ValidationRegex.ValueStringPointer(),
			DefaultValue:    valueType.String.DefaultValue.ValueStringPointer(),
		}
	}

	if valueType.Email != nil {
		result.Email = &client.TagValueEmail{
			ValidationRegex: valueType.Email.ValidationRegex.ValueStringPointer(),
			DefaultValue:    valueType.Email.DefaultValue.ValueStringPointer(),
		}
	}

	if valueType.Integer != nil {
		result.Integer = &client.TagValueInteger{
			DefaultValue: valueType.Integer.DefaultValue.ValueInt64Pointer(),
		}
	}

	if valueType.Number != nil {
		result.Number = &client.TagValueNumber{
			DefaultValue: valueType.Number.DefaultValue.ValueFloat64Pointer(),
		}
	}

	if valueType.SingleSelect != nil {
		result.SingleSelect = &client.TagValueSingleSelect{
			Options:      extractStringValues(valueType.SingleSelect.Options),
			DefaultValue: valueType.SingleSelect.DefaultValue.ValueStringPointer(),
		}
	}

	if valueType.MultiSelect != nil {
		result.MultiSelect = &client.TagValueMultiSelect{
			Options: extractStringValues(valueType.MultiSelect.Options),
		}
		if valueType.MultiSelect.DefaultValue != nil {
			v := extractStringValues(valueType.MultiSelect.DefaultValue)
			result.MultiSelect.DefaultValue = &v
		}
	}

	return result
}

// Read refreshes the Terraform state with the latest data.
func (r *tagDefinitionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var name types.String

	diags := req.State.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tagDefinition, err := r.client.ReadTagDefinition(name.ValueString())
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
	var spec tagDefinitionSpec

	diags := req.Plan.GetAttribute(ctx, path.Root("spec"), &spec)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	valueType := buildValueType(spec.ValueType)
	name := spec.TargetKind.ValueString() + "." + spec.Key.ValueString()

	update := client.MeshTagDefinition{
		ApiVersion: client.API_VERSION_TAG_DEFINITION,
		Kind:       "meshTagDefinition",
		Metadata: client.MeshTagDefinitionMetadata{
			Name: name,
		},
		Spec: client.MeshTagDefinitionSpec{
			TargetKind:     spec.TargetKind.ValueString(),
			Key:            spec.Key.ValueString(),
			ValueType:      valueType,
			Description:    spec.Description.ValueString(),
			DisplayName:    spec.DisplayName.ValueString(),
			SortOrder:      spec.SortOrder.ValueInt64(),
			Mandatory:      spec.Mandatory.ValueBool(),
			Immutable:      spec.Immutable.ValueBool(),
			Restricted:     spec.Restricted.ValueBool(),
			ReplicationKey: spec.ReplicationKey.ValueStringPointer(),
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
