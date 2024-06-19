package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type MeshProject struct {
	ApiVersion string              `json:"apiVersion" tfsdk:"api_version"`
	Kind       string              `json:"kind" tfsdk:"kind"`
	Metadata   MeshProjectMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshProjectSpec     `json:"spec" tfsdk:"spec"`
}

type MeshProjectMetadata struct {
	Name             string  `json:"name" tfsdk:"name"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        string  `json:"createdOn" tfsdk:"created_on"`
	DeletedOn        *string `json:"deletedOn" tfsdk:"deleted_on"`
}

type MeshProjectSpec struct {
	DisplayName                       string              `json:"displayName" tfsdk:"display_name"`
	Tags                              map[string][]string `json:"tags" tfsdk:"tags"`
	PaymentMethodIdentifier           *string             `json:"paymentMethodIdentifier" tfsdk:"payment_method_identifier"`
	SubstitutePaymentMethodIdentifier *string             `json:"substitutePaymentMethodIdentifier" tfsdk:"substitute_payment_method_identifier"`
}

type MeshProjectCreate struct {
	Metadata MeshProjectCreateMetadata `json:"metadata" tfsdk:"metadata"`
	Spec     MeshProjectSpec           `json:"spec" tfsdk:"spec"`
}

type MeshProjectCreateMetadata struct {
	Name             string `json:"name" tfsdk:"name"`
	OwnedByWorkspace string `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
}

func (c *MeshStackProviderClient) urlForProject(workspace string, name string) *url.URL {
	identifier := workspace + "." + name
	return c.endpoints.Projects.JoinPath(identifier)
}

func (c *MeshStackProviderClient) ReadProject(workspace string, name string) (*MeshProject, error) {
	targetUrl := c.urlForProject(workspace, name)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_PROJECT)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 404 {
		return nil, nil
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var project MeshProject
	err = json.Unmarshal(data, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (c *MeshStackProviderClient) ReadProjects(workspaceIdentifier string, paymentMethodIdentifier *string) (*[]MeshProject, error) {
	var allProjects []MeshProject

	pageNumber := 0
	targetUrl := c.endpoints.Projects
	query := targetUrl.Query()
	query.Set("workspaceIdentifier", workspaceIdentifier)
	if paymentMethodIdentifier != nil {
		query.Set("paymentIdentifier", *paymentMethodIdentifier)
	}

	for {
		query.Set("page", fmt.Sprintf("%d", pageNumber))

		targetUrl.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", targetUrl.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", CONTENT_TYPE_PROJECT)

		res, err := c.doAuthenticatedRequest(req)
		if err != nil {
			return nil, err
		}

		defer res.Body.Close()

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
		}

		var response struct {
			Embedded struct {
				MeshProjects []MeshProject `json:"meshProjects"`
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

		allProjects = append(allProjects, response.Embedded.MeshProjects...)

		// Check if there are more pages
		if response.Page.Number >= response.Page.TotalPages-1 {
			break
		}

		pageNumber++
	}

	return &allProjects, nil
}

func (c *MeshStackProviderClient) CreateProject(project *MeshProjectCreate) (*MeshProject, error) {
	payload, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.endpoints.Projects.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PROJECT)
	req.Header.Set("Accept", CONTENT_TYPE_PROJECT)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 201 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var createdProject MeshProject
	err = json.Unmarshal(data, &createdProject)
	if err != nil {
		return nil, err
	}

	return &createdProject, nil
}

func (c *MeshStackProviderClient) UpdateProject(project *MeshProjectCreate) (*MeshProject, error) {
	targetUrl := c.urlForProject(project.Metadata.OwnedByWorkspace, project.Metadata.Name)

	payload, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetUrl.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PROJECT)
	req.Header.Set("Accept", CONTENT_TYPE_PROJECT)

	res, err := c.doAuthenticatedRequest(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var updatedProject MeshProject
	err = json.Unmarshal(data, &updatedProject)
	if err != nil {
		return nil, err
	}

	return &updatedProject, nil
}

func (c *MeshStackProviderClient) DeleteProject(workspace string, name string) error {
	targetUrl := c.urlForProject(workspace, name)
	return c.deleteMeshObject(*targetUrl, 202)
}
