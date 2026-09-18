package provider

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/meshcloud/meshstack-cli/pkg/auth"
	"github.com/meshcloud/meshstack-cli/pkg/setting"
)

type MeshStackProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	Profile   types.String `tfsdk:"profile"`
	Workspace types.String `tfsdk:"workspace"`
	ApiKey    types.String `tfsdk:"apikey"`
	ApiSecret types.String `tfsdk:"apisecret"`
	ApiToken  types.String `tfsdk:"apitoken"`
}

var modelAttributes = func() (result map[string]providerModelAttribute) {
	result = map[string]providerModelAttribute{
		"endpoint":  {Setting: setting.Endpoint},
		"profile":   {Setting: setting.Profile},
		"workspace": {Setting: setting.Workspace, ExtraDescription: "Leaving this empty takes the only workspace a profile's browser login reaches, else that profile's default workspace. The provider cannot ask, so a login reaching several workspaces without a default needs a value here."},
		"apikey":    {Setting: setting.ApiKeyClientId, ExtraDescription: fmt.Sprintf("Setting this here with the secret in `%s` keeps the secret out of the configuration and out of state.", setting.ApiKeyClientSecret.EnvKey())},
		"apisecret": {Setting: setting.ApiKeyClientSecret, Sensitive: true, ExtraDescription: "Always set this through the environment: a value set here is written to Terraform state. The warning about a stored secret winning is a log record, which `TF_LOG=WARN` shows."},
		"apitoken":  {Setting: setting.ApiToken, Sensitive: true, ExtraDescription: "Always set this through the environment: a value set here is written to Terraform state."},
	}
	for field := range reflect.TypeFor[MeshStackProviderModel]().Fields() {
		tagValue := field.Tag.Get("tfsdk")
		if _, found := result[tagValue]; !found {
			panic(fmt.Sprintf("cannot find %s in model attributes", tagValue))
		}
	}
	return
}()

func (m MeshStackProviderModel) Lookup(_ context.Context, settingKey string) (string, error) {
	attributeKey, hasAttribute := getAttributeKey(settingKey)
	if !hasAttribute {
		return "", nil
	}
	for field, value := range reflect.ValueOf(m).Fields() {
		if field.Tag.Get("tfsdk") == attributeKey {
			switch v := value.Interface().(type) {
			case types.String:
				switch {
				case v.IsUnknown():
					return "", fmt.Errorf("cannot use unknown value at configuration time")
				default:
					return v.ValueString(), nil
				}
			default:
				panic(fmt.Sprintf("attribute %s is not a string but %T, support not implemented", attributeKey, v))
			}
		}
	}
	panic(fmt.Sprintf("cannot look up setting %s / attribute %s", settingKey, attributeKey))
}

func (m MeshStackProviderModel) Describe(settingKey string) string {
	attributeKey, hasAttribute := getAttributeKey(settingKey)
	if !hasAttribute {
		return ""
	}
	return fmt.Sprintf("provider block attribute '%s'", attributeKey)
}

// getAttributeKey reports no attribute for a setting the block cannot express, such as
// MESHSTACK_USER_AGENT.
func getAttributeKey(settingKey string) (string, bool) {
	for key, attribute := range modelAttributes {
		if attribute.EnvKey() == settingKey {
			return key, true
		}
	}
	return "", false
}

type providerModelAttribute struct {
	setting.Setting
	// ExtraDescription is for what is true of Terraform alone. A fact about the setting itself belongs in
	// the meshStack CLI's declaration, so that the two front ends cannot describe it differently.
	ExtraDescription string
	Sensitive        bool
}

func (a providerModelAttribute) MarkdownDescription() string {
	markdown := a.HelpMarkdown()
	if a.ExtraDescription != "" {
		markdown += "\n\n" + a.ExtraDescription
	}
	// tfplugindocs renders the description into a bullet list item, where a following line stays in
	// that item only while it is indented to the item's content column.
	return strings.ReplaceAll(markdown, "\n", "\n  ")
}

// newWorkspaceSource picks the workspace a browser login acts in, where nothing above it named one.
// It is the meshStack CLI's workspace selection minus the prompt, and a fallback source for the
// same reason: a profile's stored default ranks below it, because only a login changes that default.
//
// It looks nothing up for an API key or an API token. Both resolve their workspace without reaching
// the backend, and a credential holding no permission to list workspaces would fail there.
//
// It never fails: a configuration that resolves today must not start failing over a lookup.
func newWorkspaceSource() setting.FallbackSource {
	return setting.FallbackLookupSource(setting.Workspace.EnvKey(), "the workspace of the profile's login",
		func(ctx context.Context) (string, error) {
			method, err := auth.MethodFromContext(ctx)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Looking up no workspace, because the credential is unknown: %v", err))
				return "", nil
			}
			if method != auth.OidcLoginMethod {
				return "", nil
			}
			workspaces, err := setting.WorkspacesFromContext(ctx)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Looking up no workspace, because the login's workspaces are unknown: %v", err))
				return "", nil
			}
			selected := workspaces.Single(ctx)
			if selected == nil {
				selected = workspaces.ProfileDefault(ctx)
			}
			if selected == nil {
				return "", nil
			}
			return string(selected.Name()), nil
		})
}
