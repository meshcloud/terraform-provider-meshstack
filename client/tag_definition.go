package client

import (
	"context"

	"github.com/meshcloud/terraform-provider-meshstack/client/internal"
)

const API_VERSION_TAG_DEFINITION = "v1"

type MeshTagDefinition struct {
	ApiVersion string                    `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                    `json:"kind" tfsdk:"kind"`
	Metadata   MeshTagDefinitionMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshTagDefinitionSpec     `json:"spec" tfsdk:"spec"`
}

type MeshTagDefinitionMetadata struct {
	Name string `json:"name" tfsdk:"name"`
}

type MeshTagDefinitionSpec struct {
	TargetKind     string                     `json:"targetKind" tfsdk:"target_kind"`
	Key            string                     `json:"key" tfsdk:"key"`
	ValueType      MeshTagDefinitionValueType `json:"valueType" tfsdk:"value_type"`
	Description    string                     `json:"description" tfsdk:"description"`
	DisplayName    string                     `json:"displayName" tfsdk:"display_name"`
	SortOrder      int64                      `json:"sortOrder" tfsdk:"sort_order"`
	Mandatory      bool                       `json:"mandatory" tfsdk:"mandatory"`
	Immutable      bool                       `json:"immutable" tfsdk:"immutable"`
	Restricted     bool                       `json:"restricted" tfsdk:"restricted"`
	ReplicationKey *string                    `json:"replicationKey,omitempty" tfsdk:"replication_key"`
}

type MeshTagDefinitionValueType struct {
	String       *TagValueString       `json:"string,omitempty" tfsdk:"string"`
	Email        *TagValueEmail        `json:"email,omitempty" tfsdk:"email"`
	Integer      *TagValueInteger      `json:"integer,omitempty" tfsdk:"integer"`
	Number       *TagValueNumber       `json:"number,omitempty" tfsdk:"number"`
	SingleSelect *TagValueSingleSelect `json:"singleSelect,omitempty" tfsdk:"single_select"`
	MultiSelect  *TagValueMultiSelect  `json:"multiSelect,omitempty" tfsdk:"multi_select"`
}

type TagValueString struct {
	DefaultValue    *string `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex *string `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type TagValueEmail struct {
	DefaultValue    *string `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex *string `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type TagValueInteger struct {
	DefaultValue *int64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueNumber struct {
	DefaultValue *float64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueSingleSelect struct {
	Options      []string `json:"options,omitempty" tfsdk:"options"`
	DefaultValue *string  `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueMultiSelect struct {
	Options      []string  `json:"options,omitempty" tfsdk:"options"`
	DefaultValue *[]string `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type MeshTagDefinitionClient struct {
	meshObject internal.MeshObjectClient[MeshTagDefinition]
}

func newTagDefinitionClient(ctx context.Context, httpClient *internal.HttpClient) MeshTagDefinitionClient {
	return MeshTagDefinitionClient{
		meshObject: internal.NewMeshObjectClient[MeshTagDefinition](ctx, httpClient, "v1"),
	}
}

func (c MeshTagDefinitionClient) List(ctx context.Context) ([]MeshTagDefinition, error) {
	return c.meshObject.List(ctx)
}

func (c MeshTagDefinitionClient) Read(ctx context.Context, name string) (*MeshTagDefinition, error) {
	return c.meshObject.Get(ctx, name)
}

func (c MeshTagDefinitionClient) Create(ctx context.Context, tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	return c.meshObject.Post(ctx, tagDefinition)
}

func (c MeshTagDefinitionClient) Update(ctx context.Context, tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	return c.meshObject.Put(ctx, tagDefinition.Metadata.Name, tagDefinition)
}

func (c MeshTagDefinitionClient) Delete(ctx context.Context, name string) error {
	return c.meshObject.Delete(ctx, name)
}
