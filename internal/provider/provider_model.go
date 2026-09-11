package provider

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/types"
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

// ExtraDescription is for what is true of Terraform alone. A fact about the setting itself belongs in
// the meshStack CLI's declaration, so that the two front ends cannot describe it differently.
var modelAttributes = func() (result map[string]providerModelAttribute) {
	result = map[string]providerModelAttribute{
		"endpoint":  {Setting: setting.Endpoint},
		"profile":   {Setting: setting.Profile, ExtraDescription: "A block holding only `profile` is a complete configuration."},
		"workspace": {Setting: setting.Workspace},
		"apikey":    {Setting: setting.ApiKeyClientId, ExtraDescription: fmt.Sprintf("Setting this here with the secret in `%s` keeps the secret out of the configuration and out of state.", setting.ApiKeyClientSecret.EnvKey())},
		"apisecret": {Setting: setting.ApiKeyClientSecret, Sensitive: true, ExtraDescription: "Always set this through the environment: a value set here is written to Terraform state. The warning about a stored secret winning is a log record, which `TF_LOG=WARN` shows."},
		"apitoken":  {Setting: setting.ApiBearerToken, Sensitive: true, ExtraDescription: "Always set this through the environment: a value set here is written to Terraform state."},
	}
	for field := range reflect.TypeFor[MeshStackProviderModel]().Fields() {
		tagValue := field.Tag.Get("tfsdk")
		if _, found := result[tagValue]; !found {
			panic(fmt.Sprintf("cannot find %s in model attributes", tagValue))
		}
	}
	return
}()

func (m MeshStackProviderModel) Lookup(settingKey string) (string, error) {
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

func (m MeshStackProviderModel) Describe(settingKey string) setting.SourceDescription {
	attributeKey, hasAttribute := getAttributeKey(settingKey)
	if !hasAttribute {
		return setting.SourceDescription{}
	}
	return setting.SourceDescription{
		Type:    "provider block attribute",
		Details: attributeKey,
	}
}

// getAttributeKey reports no attribute for a setting the block cannot express, such as
// MESHSTACK_NO_INPUT.
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
	ExtraDescription string
	Sensitive        bool
}

func (a providerModelAttribute) MarkdownDescription() string {
	if a.ExtraDescription == "" {
		return a.Help()
	}
	return a.Help() + "\n\n" + a.ExtraDescription
}
