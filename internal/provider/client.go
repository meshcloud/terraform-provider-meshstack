package provider

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	apiMeshObjectsRoot = "/api/meshobjects"
	loginEndpoint      = "/api/login"

	ERROR_GENERIC_CLIENT_ERROR   = "client error"
	ERROR_GENERIC_API_ERROR      = "api error"
	ERROR_AUTHENTICATION_FAILURE = "not authorized. check api key and secret."
	ERROR_ENDPOINT_LOOKUP        = "could not fetch endpoints for meshStack."
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
	BuildingBlocks string `json:"meshbuildingblocks"`
}

type loginResponse struct {
	Token     string `json:"access_token"`
	ExpireSec int    `json:"expires_in"`
}

func NewClient(url *url.URL, apiKey string, apiSecret string) (*MeshStackProviderClient, error) {
	client := &MeshStackProviderClient{
		url: url,
		httpClient: &http.Client{
			Timeout: time.Minute * 5,
		},
		apiKey:    apiKey,
		apiSecret: apiSecret,
		token:     "",
	}

	if err := client.lookUpEndpoints(); err != nil {
		return nil, errors.New(ERROR_ENDPOINT_LOOKUP)
	}

	return client, nil
}

func (c *MeshStackProviderClient) login() error {
	loginPath, err := url.JoinPath(c.url.String(), loginEndpoint)
	if err != nil {
		return err
	}
	loginUrl, _ := url.Parse(loginPath)

	res, _ := c.httpClient.Do(
		&http.Request{
			URL:    loginUrl,
			Method: "POST",
			Header: http.Header{
				"client_id":     {c.apiKey},
				"client_secret": {c.apiSecret},
				"grant_type":    {"client_credentials"},
			},
		},
	)

	if err != nil {
		return errors.New(ERROR_GENERIC_CLIENT_ERROR)
	}

	defer res.Body.Close()

	if err != nil || res.StatusCode != 200 {
		return errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var loginResult loginResponse
	json.Unmarshal(data, &loginResult)
	c.token = loginResult.Token
	c.tokenExpiry = time.Now().Add(time.Second * time.Duration(loginResult.ExpireSec))

	return nil
}

func (c *MeshStackProviderClient) ensureValidToken() error {
	if c.token == "" || time.Now().Add(time.Second*30).After(c.tokenExpiry) {
		return c.login()
	}
	return nil
}

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
	json.Unmarshal(data, &endpoints)
	c.endpoints = endpoints

	return nil
}

func (c *MeshStackProviderClient) ReadBuildingBlock(uuid string) (*MeshBuildingBlock, error) {
	if c.ensureValidToken() != nil {
		return nil, errors.New(ERROR_AUTHENTICATION_FAILURE)
	}

	targetPath, err := url.JoinPath(c.url.String(), c.endpoints.BuildingBlocks, uuid)
	if err != nil {
		return nil, err
	}

	targetUrl, _ := url.Parse(targetPath)
	res, err := c.httpClient.Do(
		&http.Request{
			URL:    targetUrl,
			Method: "GET",
			Header: http.Header{
				"Authorization": {c.token},
			},
		},
	)

	if err != nil {
		return nil, errors.New(ERROR_GENERIC_CLIENT_ERROR)
	}

	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, errors.New(ERROR_GENERIC_API_ERROR)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	
	var bb MeshBuildingBlock
	json.Unmarshal(data, &bb)
	return &bb, nil
}
