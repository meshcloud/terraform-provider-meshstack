package provider

import (
	"github.com/meshcloud/terraform-provider-meshstack/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Schema helpers
// Contains IO types which are allowed for both inputs and outputs.
func buildingBlockIoMapBase(optional bool) schema.MapNestedAttribute {
	return schema.MapNestedAttribute{
		Optional: optional,
		Computed: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"value_string": schema.StringAttribute{Optional: optional, Computed: !optional},
				"value_int":    schema.Int64Attribute{Optional: optional, Computed: !optional},
				"value_bool":   schema.BoolAttribute{Optional: optional, Computed: !optional},
				"value_code":   schema.StringAttribute{MarkdownDescription: "Code value.", Optional: optional, Computed: !optional},
			},
		},
	}
}

// Contains IO types which are allowed for both user and combined inputs.
func buildingBlockInputBase(optional bool) schema.MapNestedAttribute {
	base := buildingBlockIoMapBase(optional)

	// Add input-specific attributes
	base.NestedObject.Attributes["value_single_select"] = schema.StringAttribute{
		Optional: optional,
		Computed: !optional,
	}
	base.NestedObject.Attributes["value_multi_select"] = schema.ListAttribute{
		ElementType:         types.StringType,
		Optional:            optional,
		Computed:            !optional,
		MarkdownDescription: "Multi-select value (list of strings).",
	}
	return base
}

func buildingBlockUserInputs() schema.MapNestedAttribute {
	inputs := buildingBlockInputBase(true)

	inputs.MarkdownDescription = "Building block user inputs. Each input has exactly one value. Use the value attribute that corresponds to the desired input type, e.g. `value_int` to set an integer input, and leave the remaining attributes empty."
	inputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.RequiresReplace()}

	inputs.Default = mapdefault.StaticValue(
		types.MapValueMust(
			types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"value_string":        types.StringType,
					"value_single_select": types.StringType,
					"value_multi_select":  types.ListType{ElemType: types.StringType},
					"value_int":           types.Int64Type,
					"value_bool":          types.BoolType,
					"value_code":          types.StringType,
				},
			},
			map[string]attr.Value{},
		),
	)

	// Add validator to ensure exactly one value is set
	inputs.NestedObject.Attributes["value_string"] = schema.StringAttribute{
		Optional: true,
		Computed: false,
		Validators: []validator.String{stringvalidator.ExactlyOneOf(
			path.MatchRelative().AtParent().AtName("value_string"),
			path.MatchRelative().AtParent().AtName("value_single_select"),
			path.MatchRelative().AtParent().AtName("value_multi_select"),
			path.MatchRelative().AtParent().AtName("value_int"),
			path.MatchRelative().AtParent().AtName("value_bool"),
			path.MatchRelative().AtParent().AtName("value_code"),
		)},
	}

	return inputs
}

func buildingBlockCombinedInputs() schema.MapNestedAttribute {
	inputs := buildingBlockInputBase(false)

	inputs.MarkdownDescription = "Contains all building block inputs. Each input has exactly one value attribute set according to its' type."
	inputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}

	// Add combined input-specific attributes (all are computed)
	inputs.NestedObject.Attributes["value_file"] = schema.StringAttribute{Optional: false, Computed: true}
	inputs.NestedObject.Attributes["value_list"] = schema.StringAttribute{
		MarkdownDescription: "Deprecated: use `value_code` instead. JSON encoded list of objects.",
		Optional:            false,
		Computed:            true,
	}

	return inputs
}

func buildingBlockOutputs() schema.MapNestedAttribute {
	outputs := buildingBlockIoMapBase(false)

	outputs.MarkdownDescription = "Building block outputs. Each output has exactly one value attribute set."
	outputs.PlanModifiers = []planmodifier.Map{mapplanmodifier.UseStateForUnknown()}

	return outputs
}

// Resource models and functions

type buildingBlockUserInputModel struct {
	ValueString       types.String   `tfsdk:"value_string"`
	ValueSingleSelect types.String   `tfsdk:"value_single_select"`
	ValueMultiSelect  []types.String `tfsdk:"value_multi_select"`
	ValueInt          types.Int64    `tfsdk:"value_int"`
	ValueBool         types.Bool     `tfsdk:"value_bool"`
	ValueCode         types.String   `tfsdk:"value_code"`
}

type buildingBlockOutputModel struct {
	ValueString types.String `tfsdk:"value_string"`
	ValueInt    types.Int64  `tfsdk:"value_int"`
	ValueBool   types.Bool   `tfsdk:"value_bool"`
	ValueCode   types.String `tfsdk:"value_code"`
}

type buildingBlockIoModel struct {
	ValueString       types.String   `tfsdk:"value_string"`
	ValueSingleSelect types.String   `tfsdk:"value_single_select"`
	ValueMultiSelect  []types.String `tfsdk:"value_multi_select"`
	ValueFile         types.String   `tfsdk:"value_file"`
	ValueInt          types.Int64    `tfsdk:"value_int"`
	ValueBool         types.Bool     `tfsdk:"value_bool"`
	ValueList         types.String   `tfsdk:"value_list"`
	ValueCode         types.String   `tfsdk:"value_code"`
}

func (input *buildingBlockIoModel) toOutputModel() buildingBlockOutputModel {
	return buildingBlockOutputModel{
		ValueString: input.ValueString,
		ValueInt:    input.ValueInt,
		ValueBool:   input.ValueBool,
		ValueCode:   input.ValueCode,
	}
}

func (io *buildingBlockUserInputModel) extractIoValue() (any, string) {
	if !io.ValueBool.IsNull() && !io.ValueBool.IsUnknown() {
		return io.ValueBool.ValueBool(), client.MESH_BUILDING_BLOCK_IO_TYPE_BOOLEAN
	}
	if !io.ValueInt.IsNull() && !io.ValueInt.IsUnknown() {
		return io.ValueInt.ValueInt64(), client.MESH_BUILDING_BLOCK_IO_TYPE_INTEGER
	}
	if !io.ValueSingleSelect.IsNull() && !io.ValueSingleSelect.IsUnknown() {
		return io.ValueSingleSelect.ValueString(), client.MESH_BUILDING_BLOCK_IO_TYPE_SINGLE_SELECT
	}
	// Note: this only works as long as we don't allow empty lists
	if len(io.ValueMultiSelect) != 0 {
		values := make([]string, 0)
		for _, value := range io.ValueMultiSelect {
			values = append(values, value.ValueString())
		}
		return values, client.MESH_BUILDING_BLOCK_IO_TYPE_MULTI_SELECT
	}
	if !io.ValueString.IsNull() && !io.ValueString.IsUnknown() {
		return io.ValueString.ValueString(), client.MESH_BUILDING_BLOCK_IO_TYPE_STRING
	}
	if !io.ValueCode.IsNull() && !io.ValueCode.IsUnknown() {
		return io.ValueCode.ValueString(), client.MESH_BUILDING_BLOCK_IO_TYPE_CODE
	}
	return nil, "No value present."
}
