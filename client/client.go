package client

import (
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
)

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
	BuildingBlocks       *url.URL `json:"meshbuildingblocks"`
	Projects             *url.URL `json:"meshprojects"`
	ProjectUserBindings  *url.URL `json:"meshprojectuserbindings"`
	ProjectGroupBindings *url.URL `json:"meshprojectgroupbindings"`
	Workspaces           *url.URL `json:"meshworkspaces"`
	WorkspaceGroupBindings *url.URL `json:"meshworkspacegroupbindings"`
	Tenants              *url.URL `json:"meshtenants"`
	TagDefinitions       *url.URL `json:"meshtagdefinitions"`
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
		BuildingBlocks:       rootUrl.JoinPath(apiMeshObjectsRoot, "meshbuildingblocks"),
		Projects:             rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojects"),
		ProjectUserBindings:  rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojectbindings", "userbindings"),
		ProjectGroupBindings: rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojectbindings", "groupbindings"),
		Workspaces:           rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspaces"),
		WorkspaceUserBindings: rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspacebindings", "userbindings"),
		WorkspaceGroupBindings: rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspacebindings", "groupbindings"),
		Tenants:              rootUrl.JoinPath(apiMeshObjectsRoot, "meshtenants"),
		TagDefinitions:       rootUrl.JoinPath(apiMeshObjectsRoot, "meshtagdefinitions"),
	}

	return client, nil
}

func (c *MeshStackProviderClient) login() error {
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

	if err != nil {
		return err
	} else if res.StatusCode != 200 {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

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
	if c.token == "" || time.Now().Add(time.Second*30).After(c.tokenExpiry) {
		return c.login()
	}
	return nil
}

// nolint: unused
func (c *MeshStackProviderClient) lookUpEndpoints() error {
	if c.ensureValidToken() != nil {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

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
	req.Header.Set("User-Agent", "meshStack Terraform Provider")

	// log request before adding auth
	log.Println(req)

	// add authentication
	err := c.ensureValidToken()
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	log.Println(res)

	return res, nil
}

func (c *MeshStackProviderClient) deleteMeshObject(targetUrl url.URL, expectedStatus int) error {
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

	if res.StatusCode != expectedStatus {
		return fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, data)
	}

	return nil
}
