package client

import (
	"fmt"
	"net/url"
)

const API_VERSION_TAG_DEFINITION = "v1"
const CONTENT_TYPE_TAG_DEFINITION = "application/vnd.meshcloud.api.meshtagdefinition.v1.hal+json"

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

func (c *MeshStackProviderClient) urlForTagDefinition(name string) *url.URL {
	return c.endpoints.TagDefinitions.JoinPath(name)
}

func (c *MeshStackProviderClient) ReadTagDefinitions() (*[]MeshTagDefinition, error) {
	var all []MeshTagDefinition

	pageNumber := 0
	targetUrl := c.endpoints.TagDefinitions
	query := targetUrl.Query()

	for {
		query.Set("page", fmt.Sprintf("%d", pageNumber))
		targetUrl.RawQuery = query.Encode()

		body, err := c.doAuthenticatedRequest("GET", targetUrl,
			withAccept(CONTENT_TYPE_TAG_DEFINITION),
		)
		items, response, err := unmarshalPaginatedBody[MeshTagDefinition](body, err, "meshTagDefinitions")
		if err != nil {
			return nil, err
		}

		all = append(all, items...)

		// Check if there are more pages
		if response.Page.Number >= response.Page.TotalPages-1 {
			break
		}

		pageNumber++
	}

	return &all, nil
}

func (c *MeshStackProviderClient) ReadTagDefinition(name string) (*MeshTagDefinition, error) {
	return unmarshalBody[MeshTagDefinition](c.doAuthenticatedRequest("GET", c.urlForTagDefinition(name),
		withAccept(CONTENT_TYPE_TAG_DEFINITION),
	))
}

func (c *MeshStackProviderClient) CreateTagDefinition(tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	result, err := unmarshalBody[MeshTagDefinition](c.doAuthenticatedRequest("POST", c.endpoints.TagDefinitions,
		withPayload(tagDefinition, CONTENT_TYPE_TAG_DEFINITION),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to do authenticated request: %w", err)
	}
	return result, nil
}

func (c *MeshStackProviderClient) UpdateTagDefinition(tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	result, err := unmarshalBody[MeshTagDefinition](c.doAuthenticatedRequest("PUT", c.urlForTagDefinition(tagDefinition.Metadata.Name),
		withPayload(tagDefinition, CONTENT_TYPE_TAG_DEFINITION),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to do authenticated request: %w", err)
	}
	return result, nil
}

func (c *MeshStackProviderClient) DeleteTagDefinition(name string) error {
	_, err := c.doAuthenticatedRequest("DELETE", c.urlForTagDefinition(name),
		withAccept(CONTENT_TYPE_TAG_DEFINITION),
		withExpectedStatusCode(204),
	)
	if err != nil {
		return fmt.Errorf("failed to do authenticated request: %w", err)
	}

	return nil
}
