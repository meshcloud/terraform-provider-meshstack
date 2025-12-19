package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"
)

const (
	apiMeshObjectsRoot = "/api/meshobjects"
	loginEndpoint      = "/api/login"
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
	BuildingBlocks         *url.URL `json:"meshbuildingblocks"`
	Projects               *url.URL `json:"meshprojects"`
	ProjectUserBindings    *url.URL `json:"meshprojectuserbindings"`
	ProjectGroupBindings   *url.URL `json:"meshprojectgroupbindings"`
	Workspaces             *url.URL `json:"meshworkspaces"`
	WorkspaceUserBindings  *url.URL `json:"meshworkspaceuserbindings"`
	WorkspaceGroupBindings *url.URL `json:"meshworkspacegroupbindings"`
	Tenants                *url.URL `json:"meshtenants"`
	TagDefinitions         *url.URL `json:"meshtagdefinitions"`
	LandingZones           *url.URL `json:"meshlandingzones"`
	Platforms              *url.URL `json:"meshplatforms"`
	PaymentMethods         *url.URL `json:"meshpaymentmethods"`
	Integrations           *url.URL `json:"meshintegrations"`
}

type loginRequest struct {
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
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
		BuildingBlocks:         rootUrl.JoinPath(apiMeshObjectsRoot, "meshbuildingblocks"),
		Projects:               rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojects"),
		ProjectUserBindings:    rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojectbindings", "userbindings"),
		ProjectGroupBindings:   rootUrl.JoinPath(apiMeshObjectsRoot, "meshprojectbindings", "groupbindings"),
		Workspaces:             rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspaces"),
		WorkspaceUserBindings:  rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspacebindings", "userbindings"),
		WorkspaceGroupBindings: rootUrl.JoinPath(apiMeshObjectsRoot, "meshworkspacebindings", "groupbindings"),
		Tenants:                rootUrl.JoinPath(apiMeshObjectsRoot, "meshtenants"),
		TagDefinitions:         rootUrl.JoinPath(apiMeshObjectsRoot, "meshtagdefinitions"),
		LandingZones:           rootUrl.JoinPath(apiMeshObjectsRoot, "meshlandingzones"),
		Platforms:              rootUrl.JoinPath(apiMeshObjectsRoot, "meshplatforms"),
		PaymentMethods:         rootUrl.JoinPath(apiMeshObjectsRoot, "meshpaymentmethods"),
		Integrations:           rootUrl.JoinPath(apiMeshObjectsRoot, "meshintegrations"),
	}

	return client, nil
}

func (c *MeshStackProviderClient) login() error {
	loginPath, err := url.JoinPath(c.url.String(), loginEndpoint)
	if err != nil {
		return err
	}

	loginRequest := loginRequest{
		ClientId:     c.apiKey,
		ClientSecret: c.apiSecret,
	}

	payload, err := json.Marshal(loginRequest)
	if err != nil {
		return err
	}

	req, _ := http.NewRequest(http.MethodPost, loginPath, bytes.NewBuffer(payload))
	req.Header.Add("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 {
		return fmt.Errorf("login failed with status %d, check api key and secret", res.StatusCode)
	}

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
		return fmt.Errorf("cannot authenticate for delete request: %w ", err)
	}

	defer func() {
		_ = res.Body.Close()
	}()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != expectedStatus {
		return fmt.Errorf("expected status code %d, but got %d, body: '%s'", expectedStatus, res.StatusCode, string(data))
	}

	return nil
}
