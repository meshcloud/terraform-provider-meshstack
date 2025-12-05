package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const CONTENT_TYPE_INTEGRATION = "application/vnd.meshcloud.api.meshintegration.v1-preview.hal+json"

type MeshIntegration struct {
	ApiVersion string                  `json:"apiVersion" tfsdk:"api_version"`
	Kind       string                  `json:"kind" tfsdk:"kind"`
	Metadata   MeshIntegrationMetadata `json:"metadata" tfsdk:"metadata"`
	Spec       MeshIntegrationSpec     `json:"spec" tfsdk:"spec"`
	Status     *MeshIntegrationStatus  `json:"status,omitempty" tfsdk:"status"`
}

type MeshIntegrationMetadata struct {
	Uuid             *string `json:"uuid,omitempty" tfsdk:"uuid"`
	OwnedByWorkspace string  `json:"ownedByWorkspace" tfsdk:"owned_by_workspace"`
	CreatedOn        *string `json:"createdOn,omitempty" tfsdk:"created_on"`
}

type MeshIntegrationSpec struct {
	DisplayName string                `json:"displayName" tfsdk:"display_name"`
	Config      MeshIntegrationConfig `json:"config" tfsdk:"config"`
}

type MeshIntegrationStatus struct {
	IsBuiltIn                  bool                            `json:"isBuiltIn" tfsdk:"is_built_in"`
	WorkloadIdentityFederation *MeshWorkloadIdentityFederation `json:"workloadIdentityFederation,omitempty" tfsdk:"workload_identity_federation"`
}

// Integration Config wrapper with type discrimination
type MeshIntegrationConfig struct {
	Type        string                                `json:"type" tfsdk:"type"`
	Github      *MeshGithubIntegrationProperties      `json:"github,omitempty" tfsdk:"github"`
	Gitlab      *MeshGitlabIntegrationProperties      `json:"gitlab,omitempty" tfsdk:"gitlab"`
	AzureDevops *MeshAzureDevopsIntegrationProperties `json:"azuredevops,omitempty" tfsdk:"azuredevops"`
}

// GitHub Integration
type MeshGithubIntegrationProperties struct {
	Owner         string                 `json:"owner" tfsdk:"owner"`
	BaseUrl       string                 `json:"baseUrl" tfsdk:"base_url"`
	AppId         string                 `json:"appId" tfsdk:"app_id"`
	AppPrivateKey string                 `json:"appPrivateKey" tfsdk:"app_private_key"`
	RunnerRef     BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// GitLab Integration
type MeshGitlabIntegrationProperties struct {
	BaseUrl   string                 `json:"baseUrl" tfsdk:"base_url"`
	RunnerRef BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// Azure DevOps Integration
type MeshAzureDevopsIntegrationProperties struct {
	BaseUrl             string                 `json:"baseUrl" tfsdk:"base_url"`
	Organization        string                 `json:"organization" tfsdk:"organization"`
	PersonalAccessToken string                 `json:"personalAccessToken" tfsdk:"personal_access_token"`
	RunnerRef           BuildingBlockRunnerRef `json:"runnerRef" tfsdk:"runner_ref"`
}

// Building Block Runner Reference
type BuildingBlockRunnerRef struct {
	Uuid string `json:"uuid" tfsdk:"uuid"`
	Kind string `json:"kind,omitempty" tfsdk:"kind"`
}

// Workload Identity Federation
type MeshWorkloadIdentityFederation struct {
	Issuer  string              `json:"issuer" tfsdk:"issuer"`
	Subject string              `json:"subject" tfsdk:"subject"`
	Gcp     *MeshWifProvider    `json:"gcp,omitempty" tfsdk:"gcp"`
	Aws     *MeshAwsWifProvider `json:"aws,omitempty" tfsdk:"aws"`
	Azure   *MeshWifProvider    `json:"azure,omitempty" tfsdk:"azure"`
}

type MeshWifProvider struct {
	Audience string `json:"audience" tfsdk:"audience"`
}

type MeshAwsWifProvider struct {
	Audience   string `json:"audience" tfsdk:"audience"`
	Thumbprint string `json:"thumbprint" tfsdk:"thumbprint"`
}

func (c *MeshStackProviderClient) urlForIntegration(workspace string, uuid string) *url.URL {
	return c.endpoints.Integrations.JoinPath(workspace, uuid)
}

func (c *MeshStackProviderClient) ReadIntegration(workspace string, uuid string) (*MeshIntegration, error) {
	targetUrl := c.urlForIntegration(workspace, uuid)
	req, err := http.NewRequest("GET", targetUrl.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", CONTENT_TYPE_INTEGRATION)

	res, err := c.doAuthenticatedRequest(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if !isSuccessHTTPStatus(res) {
		return nil, fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	var integration MeshIntegration
	err = json.Unmarshal(data, &integration)
	if err != nil {
		return nil, err
	}

	return &integration, nil
}

func (c *MeshStackProviderClient) ReadIntegrations(workspaceIdentifier string) (*[]MeshIntegration, error) {
	var allIntegrations []MeshIntegration

	pageNumber := 0
	targetUrl := c.endpoints.Integrations
	query := targetUrl.Query()
	query.Set("workspaceIdentifier", workspaceIdentifier)

	for {
		query.Set("page", fmt.Sprintf("%d", pageNumber))
		targetUrl.RawQuery = query.Encode()

		req, err := http.NewRequest("GET", targetUrl.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("Accept", CONTENT_TYPE_INTEGRATION)

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
				MeshIntegrations []MeshIntegration `json:"meshIntegrations"`
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

		allIntegrations = append(allIntegrations, response.Embedded.MeshIntegrations...)

		// Check if there are more pages
		if response.Page.Number >= response.Page.TotalPages-1 {
			break
		}

		pageNumber++
	}

	return &allIntegrations, nil
}
