package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	TargetKind  string                     `json:"targetKind" tfsdk:"target_kind"`
	Key         string                     `json:"key" tfsdk:"key"`
	ValueType   MeshTagDefinitionValueType `json:"valueType" tfsdk:"value_type"`
	Description string                     `json:"description" tfsdk:"description"`
	DisplayName string                     `json:"displayName" tfsdk:"display_name"`
	SortOrder   int64                      `json:"sortOrder" tfsdk:"sort_order"`
	Mandatory   bool                       `json:"mandatory" tfsdk:"mandatory"`
	Immutable   bool                       `json:"immutable" tfsdk:"immutable"`
	Restricted  bool                       `json:"restricted" tfsdk:"restricted"`
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
	DefaultValue    string `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex string `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type TagValueEmail struct {
	DefaultValue    string `json:"defaultValue,omitempty" tfsdk:"default_value"`
	ValidationRegex string `json:"validationRegex,omitempty" tfsdk:"validation_regex"`
}

type TagValueInteger struct {
	DefaultValue int64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueNumber struct {
	DefaultValue float64 `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueSingleSelect struct {
	Options      []string `json:"options,omitempty" tfsdk:"options"`
	DefaultValue string   `json:"defaultValue,omitempty" tfsdk:"default_value"`
}

type TagValueMultiSelect struct {
	Options      []string `json:"options,omitempty" tfsdk:"options"`
	DefaultValue []string `json:"defaultValue,omitempty" tfsdk:"default_value"`
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

		req, err := http.NewRequest("GET", targetUrl.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", CONTENT_TYPE_TAG_DEFINITION)

		res, err := c.doAuthenticatedRequest(req)
		if err != nil {
			return nil, err
		}

		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if !isSuccessHTTPStatus(res) {
			return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
		}

		var response struct {
			Embedded struct {
				MeshTagDefinitions []MeshTagDefinition `json:"meshTagDefinitions"`
			} `json:"_embedded"`
			Page struct {
				Size          int `json:"size"`
				TotalElements int `json:"totalElements"`
				TotalPages    int `json:"totalPages"`
				Number        int `json:"number"`
			} `json:"page"`
		}

		err = json.Unmarshal(data, &response)
		if err != nil {
			return nil, err
		}

		all = append(all, response.Embedded.MeshTagDefinitions...)

		// Check if there are more pages
		if response.Page.Number >= response.Page.TotalPages-1 {
			break
		}

		pageNumber++
	}

	return &all, nil
}

func (c *MeshStackProviderClient) ReadTagDefinition(name string) (*MeshTagDefinition, error) {
	targetUrl := c.urlForTagDefinition(name)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", CONTENT_TYPE_TAG_DEFINITION)

	resp, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !isSuccessHTTPStatus(resp) {
		return nil, fmt.Errorf("failed to read tag definition: %s", resp.Status)
	}

	var tagDefinition MeshTagDefinition
	if err := json.NewDecoder(resp.Body).Decode(&tagDefinition); err != nil {
		return nil, err
	}

	return &tagDefinition, nil
}

func (c *MeshStackProviderClient) CreateTagDefinition(tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	targetUrl := c.endpoints.TagDefinitions
	data, err := json.Marshal(tagDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tag definition: %w", err)
	}

	fmt.Printf("JSON Payload: %s\n", string(data))

	req, err := http.NewRequest("POST", targetUrl.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", CONTENT_TYPE_TAG_DEFINITION)
	req.Header.Set("Accept", CONTENT_TYPE_TAG_DEFINITION)

	resp, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do authenticated request: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessHTTPStatus(resp) {
		return nil, fmt.Errorf("failed to create tag definition: %s", resp.Status)
	}

	var createdTagDefinition MeshTagDefinition
	if err := json.NewDecoder(resp.Body).Decode(&createdTagDefinition); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdTagDefinition, nil
}

func (c *MeshStackProviderClient) UpdateTagDefinition(tagDefinition *MeshTagDefinition) (*MeshTagDefinition, error) {
	targetUrl := c.urlForTagDefinition(tagDefinition.Metadata.Name)
	data, err := json.Marshal(tagDefinition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tag definition: %w", err)
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", CONTENT_TYPE_TAG_DEFINITION)
	req.Header.Set("Accept", CONTENT_TYPE_TAG_DEFINITION)

	resp, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do authenticated request: %w", err)
	}
	defer resp.Body.Close()

	if !isSuccessHTTPStatus(resp) {
		return nil, fmt.Errorf("failed to update tag definition: %s", resp.Status)
	}

	var updatedTagDefinition MeshTagDefinition
	if err := json.NewDecoder(resp.Body).Decode(&updatedTagDefinition); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedTagDefinition, nil
}

func (c *MeshStackProviderClient) DeleteTagDefinition(name string) error {
	targetUrl := c.urlForTagDefinition(name)
	req, err := http.NewRequest("DELETE", targetUrl.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", CONTENT_TYPE_TAG_DEFINITION)

	resp, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return fmt.Errorf("failed to do authenticated request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("failed to delete tag definition: %s", resp.Status)
	}

	return nil
}
