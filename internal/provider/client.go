package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiMeshObjectsRoot = "/api/meshobjects"
	loginEndpoint      = "/api/login"

	ERROR_GENERIC_CLIENT_ERROR   = "client error"
	ERROR_GENERIC_API_ERROR      = "api error"
	ERROR_AUTHENTICATION_FAILURE = "Not authorized. Check api key and secret."
	ERROR_ENDPOINT_LOOKUP        = "Could not fetch endpoints for meshStack."

	CONTENT_TYPE_PROJECT = "application/vnd.meshcloud.api.meshproject.v2.hal+json"
)

// TODO this will be an abstraction that does the login call, get a token and then use this token in the Auth header.
type MeshStackProviderClient struct {
	url         *url.URL
	httpClient  *http.Client
	apiKey      string
	apiSecret   string
	token       string
	tokenExpiry time.Time
	endpoints   endpoints
}

type endpoints struct {
	BuildingBlocks *url.URL `json:"meshbuildingblocks"`
	Projects       *url.URL `json:"meshprojects"`
}

type loginResponse struct {
	Token     string `json:"access_token"`
	ExpireSec int    `json:"expires_in"`
}

func NewClient(rootUrl *url.URL, apiKey string, apiSecret string) (*MeshStackProviderClient, error) {
	client := &MeshStackProviderClient{
		url: rootUrl,
		httpClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		apiKey:    apiKey,
		apiSecret: apiSecret,
		token:     "",
	}

	// TODO: lookup endpoints
	client.endpoints = endpoints{
		BuildingBlocks: rootUrl.JoinPath(apiMeshObjectsRoot, "meshbuildingblocks"),
		Projects:       rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojects"),
	}

	return client, nil
}

func (c *MeshStackProviderClient) login() error {
	log.Println("login")
	loginPath, err := url.JoinPath(c.url.String(), loginEndpoint)
	if err != nil {
		return err
	}

	formData := url.Values{}
	formData.Set("client_id", c.apiKey)
	formData.Set("client_secret", c.apiSecret)
	formData.Set("grant_type", "client_credentials")

	req, _ := http.NewRequest(http.MethodPost, loginPath, strings.NewReader(formData.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.httpClient.Do(req)

	if err != nil || res.StatusCode != 200 {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	log.Println(res)

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var loginResult loginResponse
	err = json.Unmarshal(data, &loginResult)
	if err != nil {
		return err
	}

	c.token = fmt.Sprintf("Bearer %s", loginResult.Token)
	c.tokenExpiry = time.Now().Add(time.Second * time.Duration(loginResult.ExpireSec))

	return nil
}

func (c *MeshStackProviderClient) ensureValidToken() error {
	log.Printf("current token: %s", c.token)
	if c.token == "" || time.Now().Add(time.Second*30).After(c.tokenExpiry) {
		return c.login()
	}
	return nil
}

// nolint: unused
func (c *MeshStackProviderClient) lookUpEndpoints() error {
	log.Println("lookUpEndpoints")
	if c.ensureValidToken() != nil {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}
	log.Printf("new token: %s", c.token)

	meshObjectsPath, err := url.JoinPath(c.url.String(), apiMeshObjectsRoot)
	if err != nil {
		return err
	}
	meshObjects, _ := url.Parse(meshObjectsPath)

	res, err := c.httpClient.Do(
		&http.Request{
			URL:    meshObjects,
			Method: "GET",
			Header: http.Header{
				"Authorization": {c.token},
			},
		},
	)

	if err != nil {
		return errors.New(ERROR_GENERIC_CLIENT_ERROR)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var endpoints endpoints
	err = json.Unmarshal(data, &endpoints)
	if err != nil {
		return err
	}

	c.endpoints = endpoints
	return nil
}

func (c *MeshStackProviderClient) doAuthenticatedRequest(req *http.Request) (*http.Response, error) {
	// ensure that headeres are initialized
	if req.Header == nil {
		req.Header = map[string][]string{}
	}

	// add authentication
	if c.ensureValidToken() != nil {
		return nil, errors.New(ERROR_AUTHENTICATION_FAILURE)
	}
	req.Header.Set("Authorization", c.token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c *MeshStackProviderClient) ReadBuildingBlock(uuid string) (*MeshBuildingBlock, error) {
	if c.ensureValidToken() != nil {
		return nil, errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	targetPath := c.endpoints.BuildingBlocks.JoinPath(uuid)
	req, err := http.NewRequest("GET", targetPath.String(), nil)
	if err != nil {
		return nil, err
	}

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

	var bb MeshBuildingBlock
	err = json.Unmarshal(data, &bb)
	if err != nil {
		return nil, err
	}

	return &bb, nil
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

	var project MeshProject
	err = json.Unmarshal(data, &project)
	if err != nil {
		return nil, err
	}

	return &project, nil
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
	targetPath := c.urlForProject(project.Metadata.OwnedByWorkspace, project.Metadata.Name)

	payload, err := json.Marshal(project)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", targetPath.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", CONTENT_TYPE_PROJECT)

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

	req, err := http.NewRequest("DELETE", targetUrl.String(), nil)
	if err != nil {
		return err
	}

	res, err := c.doAuthenticatedRequest(req)

	if err != nil {
		return errors.New(ERROR_GENERIC_CLIENT_ERROR)
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != 202 {
		return fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	return nil
}
