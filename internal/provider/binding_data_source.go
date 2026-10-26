package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/meshcloud/meshstack-cli/client"
)

var (
	_ datasource.DataSourceWithConfigure        = &bindingDataSource[workspaceBindingModel]{}
	_ datasource.DataSourceWithConfigValidators = &bindingDataSource[workspaceBindingModel]{}
)

type bindingDataSource[M any] struct {
	typeName   string
	kind       string
	attributes map[string]schema.Attribute
	// lookups starts with metadata.name, which reads the binding directly.
	lookups      []bindingLookup[M]
	selectClient func(client.Client) bindingClient[M]
	client       bindingClient[M]
}

type bindingLookup[M any] struct {
	path  path.Path
	value func(M) string
}

type bindingClient[M any] struct {
	read func(ctx context.Context, name string) (*M, error)
	list func(ctx context.Context, filter bindingFilter) ([]M, error)
}

// bindingFilter holds the configured lookup values by attribute path, such as "target_ref.name".
type bindingFilter map[string]*string

func newBindingClient[B, M any](
	read func(context.Context, string) (*B, error),
	list func(context.Context, bindingFilter) ([]B, error),
	model func(B) M,
) bindingClient[M] {
	return bindingClient[M]{
		read: func(ctx context.Context, name string) (*M, error) {
			binding, err := read(ctx, name)
			if binding == nil {
				return nil, err
			}
			return new(model(*binding)), err
		},
		list: func(ctx context.Context, filter bindingFilter) ([]M, error) {
			listed, err := list(ctx, filter)
			models := make([]M, len(listed))
			for i, binding := range listed {
				models[i] = model(binding)
			}
			return models, err
		},
	}
}

func (d *bindingDataSource[M]) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + d.typeName
}

func (d *bindingDataSource[M]) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	var names []string
	for _, lookup := range d.lookups {
		names = append(names, "`"+lookup.path.String()+"`")
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: fmt.Sprintf("Read the one %s matching the lookup attributes you set: %s. Fails unless exactly one binding matches.",
			d.kind, strings.Join(names, ", ")),
		Attributes: d.attributes,
	}
}

func (d *bindingDataSource[M]) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	var paths []path.Expression
	for _, lookup := range d.lookups {
		paths = append(paths, lookup.path.Expression())
	}
	return []datasource.ConfigValidator{datasourcevalidator.AtLeastOneOf(paths...)}
}

func (d *bindingDataSource[M]) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.client = d.selectClient(client)
	})...)
}

func (d *bindingDataSource[M]) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	filter := bindingFilter{}
	for _, lookup := range d.lookups {
		var value *string
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, lookup.path, &value)...)
		filter[lookup.path.String()] = value
	}
	if resp.Diagnostics.HasError() {
		return
	}

	candidates := d.candidates(ctx, filter, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var matches []M
	for _, candidate := range candidates {
		if d.matches(filter, candidate) {
			matches = append(matches, candidate)
		}
	}
	switch len(matches) {
	case 1:
		resp.Diagnostics.Append(resp.State.Set(ctx, &matches[0])...)
	case 0:
		resp.Diagnostics.AddError(capitalize(d.kind)+" not found", fmt.Sprintf("No %s matches %s.", d.kind, d.describe(filter)))
	default:
		names := make([]string, len(matches))
		for i, match := range matches {
			names[i] = d.lookups[0].value(match)
		}
		resp.Diagnostics.AddError(capitalize(d.kind)+" is ambiguous",
			fmt.Sprintf("%d %ss match %s: %s. Set `metadata.name` to pick one.", len(matches), d.kind, d.describe(filter), strings.Join(names, ", ")))
	}
}

func (d *bindingDataSource[M]) candidates(ctx context.Context, filter bindingFilter, diags *diag.Diagnostics) []M {
	name := filter[d.lookups[0].path.String()]
	if name == nil {
		bindings, err := d.client.list(ctx, filter)
		if err != nil {
			diags.AddError("Unable to list "+d.kind+"s", err.Error())
		}
		return bindings
	}
	binding, err := d.client.read(ctx, *name)
	if err != nil {
		diags.AddError("Unable to read "+d.kind, err.Error())
	}
	if binding == nil {
		return nil
	}
	return []M{*binding}
}

func (d *bindingDataSource[M]) matches(filter bindingFilter, binding M) bool {
	for _, lookup := range d.lookups {
		if want := filter[lookup.path.String()]; want != nil && lookup.value(binding) != *want {
			return false
		}
	}
	return true
}

func (d *bindingDataSource[M]) describe(filter bindingFilter) string {
	var set []string
	for _, lookup := range d.lookups {
		if value := filter[lookup.path.String()]; value != nil {
			set = append(set, fmt.Sprintf("%s = %q", lookup.path, *value))
		}
	}
	return strings.Join(set, " and ")
}

func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}

// bindingExpired follows meshStack, which treats the expiry date itself as still effective.
func bindingExpired(expiryDate *string) bool {
	return expiryDate != nil && time.Now().UTC().Format(time.DateOnly) > *expiryDate
}

func bindingDataSourceAttributes(role, roleExample, subject, subjectNameDescription string, targetRef schema.SingleNestedAttribute, targetLookups ...string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"metadata": asFilter(schema.SingleNestedAttribute{
			Computed: true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Unique name of the binding.",
					Computed:            true,
					Validators:          []validator.String{stringvalidator.LengthBetween(1, 45)},
				},
			},
		}, "name"),
		"role_ref": asFilter(schema.SingleNestedAttribute{
			MarkdownDescription: capitalize(role) + " role assigned by this binding.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: "Name of the " + role + " role, for example " + roleExample + ".",
					Computed:            true,
				},
			},
		}, "name"),
		"target_ref": asFilter(targetRef, targetLookups...),
		"subject": asFilter(schema.SingleNestedAttribute{
			MarkdownDescription: "The " + subject + " assigned by this binding.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{
					MarkdownDescription: subjectNameDescription,
					Computed:            true,
				},
			},
		}, "name"),
		"expiry_date": schema.StringAttribute{
			MarkdownDescription: "Last day the binding is effective (`YYYY-MM-DD`); null if it never expires.",
			Computed:            true,
		},
		"expired": schema.BoolAttribute{
			MarkdownDescription: "True after the expiry date has passed.",
			Computed:            true,
		},
	}
}
