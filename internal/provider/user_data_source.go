package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/datasourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/meshcloud/meshstack-cli/client"

	"github.com/meshcloud/terraform-provider-meshstack/internal/types/generic"
)

var (
	_ datasource.DataSource                     = &userDataSource{}
	_ datasource.DataSourceWithConfigure        = &userDataSource{}
	_ datasource.DataSourceWithConfigValidators = &userDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	meshUserClient client.MeshUserClient
}

type userFilter struct {
	Name  *string
	Uuid  *string
	Email *string
	Euid  *string
}

type userModel struct {
	Ref      client.NamedRef         `tfsdk:"ref"`
	Metadata client.MeshUserMetadata `tfsdk:"metadata"`
	Spec     client.MeshUserSpec     `tfsdk:"spec"`
	Status   client.MeshUserStatus   `tfsdk:"status"`
}

func newUserModel(user client.MeshUser) userModel {
	return userModel{
		Ref:      client.NamedRef{Kind: client.MeshObjectKind.User, Name: user.Metadata.Name},
		Metadata: user.Metadata,
		Spec:     user.Spec,
		Status:   user.Status,
	}
}

func userAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ref": meshRefByName(meshRefOptions{
			Kind:        client.MeshObjectKind.User,
			Description: "Reference to this user, can be used as `user_ref` in `meshstack_workspace_user_group_member`.",
			Output:      true,
		}),
		"metadata": schema.SingleNestedAttribute{
			MarkdownDescription: "Identifiers of the user.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"name": computedString("Username of the user. A workspace user group lists its members by it, and a user binding names the user by it in `subject.name`."),
				"uuid": computedString("UUID of the user."),
			},
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "Profile of the user.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"email":      computedString("Email address of the user."),
				"first_name": computedString("First name of the user."),
				"last_name":  computedString("Last name of the user."),
				"euid":       computedString("External user ID, which meshStack uses to find the user in the cloud platforms."),
				"tags":       tagsAttribute(tagsOptions{Kind: client.MeshObjectKind.User, Output: true}),
			},
		},
		"status": schema.SingleNestedAttribute{
			MarkdownDescription: "Information meshStack keeps about the user.",
			Computed:            true,
			Attributes: map[string]schema.Attribute{
				"lifecycle": schema.SingleNestedAttribute{
					MarkdownDescription: "Lifecycle of the user.",
					Computed:            true,
					Attributes: map[string]schema.Attribute{
						"state": computedString("`ACTIVE`, or `DEACTIVATED` for a user your identity provider has deactivated."),
					},
				},
				"last_login": computedString("When the user last signed in to meshPanel. Null for a user who has never signed in."),
			},
		},
	}
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	resp.Diagnostics.Append(configureProviderClient(req.ProviderData, func(client client.Client) {
		d.meshUserClient = client.User
	})...)
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attributes := userAttributes()
	attributes["metadata"] = asFilter(attributes["metadata"], "name", "uuid")
	attributes["spec"] = asFilter(attributes["spec"], "email", "euid")

	resp.Schema = schema.Schema{
		MarkdownDescription: "Read the one meshUser that matches what you set in `metadata` and `spec`, such as `spec.email`. " +
			"Set at least one of `metadata.name`, `metadata.uuid`, `spec.email` and `spec.euid`. " +
			"It fails unless exactly one user matches, so you can use it to check that a user exists. " +
			"Deleted users are not found.",
		Attributes: attributes,
	}
}

func (d *userDataSource) ConfigValidators(_ context.Context) []datasource.ConfigValidator {
	return []datasource.ConfigValidator{
		datasourcevalidator.AtLeastOneOf(
			path.MatchRoot("metadata").AtName("name"),
			path.MatchRoot("metadata").AtName("uuid"),
			path.MatchRoot("spec").AtName("email"),
			path.MatchRoot("spec").AtName("euid"),
		),
	}
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var filter userFilter
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("name"), &filter.Name)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("metadata").AtName("uuid"), &filter.Uuid)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("email"), &filter.Email)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("euid"), &filter.Euid)...)
	if resp.Diagnostics.HasError() {
		return
	}

	users, err := d.meshUserClient.List(ctx, filter.listFilter())
	if err != nil {
		resp.Diagnostics.AddError("Unable to list users", err.Error())
		return
	}

	user := onlyMatch("user", filter, users, filter.matches, func(user client.MeshUser) string {
		return fmt.Sprintf("%s (%s)", user.Metadata.Uuid, user.Metadata.Name)
	}, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(generic.Set(ctx, &resp.State, newUserModel(user))...)
}

func (filter userFilter) listFilter() client.MeshUserFilter {
	return client.MeshUserFilter{
		Username: valueOrEmpty(filter.Name),
		Uuid:     valueOrEmpty(filter.Uuid),
		Email:    valueOrEmpty(filter.Email),
		Euid:     valueOrEmpty(filter.Euid),
	}
}

func (filter userFilter) matches(user client.MeshUser) bool {
	return (filter.Name == nil || user.Metadata.Name == *filter.Name) &&
		(filter.Uuid == nil || user.Metadata.Uuid == *filter.Uuid) &&
		(filter.Email == nil || strings.EqualFold(user.Spec.Email, *filter.Email)) &&
		(filter.Euid == nil || user.Spec.Euid != nil && *user.Spec.Euid == *filter.Euid)
}

func (filter userFilter) String() string {
	var set []string
	for _, attribute := range []struct {
		name  string
		value *string
	}{
		{"metadata.name", filter.Name},
		{"metadata.uuid", filter.Uuid},
		{"spec.email", filter.Email},
		{"spec.euid", filter.Euid},
	} {
		if attribute.value != nil {
			set = append(set, fmt.Sprintf("%s = %q", attribute.name, *attribute.value))
		}
	}
	if len(set) == 0 {
		return "an empty filter"
	}
	return strings.Join(set, " and ")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
