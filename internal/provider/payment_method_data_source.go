package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/meshcloud/terraform-provider-meshstack/client"
)

var (
	_ datasource.DataSource              = &paymentMethodDataSource{}
	_ datasource.DataSourceWithConfigure = &paymentMethodDataSource{}
)

func NewPaymentMethodDataSource() datasource.DataSource {
	return &paymentMethodDataSource{}
}

type paymentMethodDataSource struct {
	MeshPaymentMethod client.MeshPaymentMethodClient
}

func (d *paymentMethodDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_payment_method"
}

func (d *paymentMethodDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Read a single payment method by workspace and identifier.",

		Attributes: map[string]schema.Attribute{
			"api_version": schema.StringAttribute{
				MarkdownDescription: "Payment method API version.",
				Computed:            true,
			},
			"kind": schema.StringAttribute{
				MarkdownDescription: "meshObject type, always `meshPaymentMethod`.",
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.OneOf([]string{"meshPaymentMethod"}...),
				},
			},

			"metadata": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Payment method identifier.",
						Required:            true,
					},
					"owned_by_workspace": schema.StringAttribute{
						MarkdownDescription: "Identifier of the workspace that owns this payment method.",
						Required:            true,
					},
					"created_on": schema.StringAttribute{
						MarkdownDescription: "Creation date of the payment method.",
						Computed:            true,
					},
					"deleted_on": schema.StringAttribute{
						MarkdownDescription: "Deletion date of the payment method.",
						Computed:            true,
					},
				},
			},

			"spec": schema.SingleNestedAttribute{
				Computed: true,
				Attributes: map[string]schema.Attribute{
					"display_name": schema.StringAttribute{
						MarkdownDescription: "Display name of the payment method.",
						Computed:            true,
					},
					"expiration_date": schema.StringAttribute{
						MarkdownDescription: "Expiration date of the payment method (ISO 8601 format).",
						Computed:            true,
					},
					"amount": schema.Int64Attribute{
						MarkdownDescription: "Amount associated with the payment method.",
						Computed:            true,
					},
					"tags": schema.MapAttribute{
						MarkdownDescription: "Tags of the payment method.",
						ElementType:         types.ListType{ElemType: types.StringType},
						Computed:            true,
					},
				},
			},
		},
	}
}

func (d *paymentMethodDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.MeshPaymentMethod = client.PaymentMethod
	})...)
}

func (d *paymentMethodDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var workspace, name string

	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("owned_by_workspace"), &workspace)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &name)...)

	if resp.Diagnostics.HasError() {
		return
	}

	paymentMethod, err := d.MeshPaymentMethod.Read(ctx, workspace, name)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Could not read payment method '%s' in workspace '%s'", name, workspace),
			err.Error(),
		)
		return
	}

	if paymentMethod == nil {
		resp.Diagnostics.AddError(
			"Payment method not found",
			fmt.Sprintf("The requested payment method '%s' in workspace '%s' was not found.", name, workspace),
		)
		return
	}

	if paymentMethod.Spec.Tags == nil {
		paymentMethod.Spec.Tags = make(map[string][]string)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, paymentMethod)...)
}
