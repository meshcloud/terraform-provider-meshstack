package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"slices"
	"time"
)

const (
	apiMeshObjectsRoot = "/api/meshobjects"
	loginEndpoint      = "/api/login"
)

var (
	errNotFound = errors.New("request failed with status Not Found (404)")
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
	Locations              *url.URL `json:"meshlocations"`
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
		Locations:              rootUrl.JoinPath(apiMeshObjectsRoot, "meshlocations"),
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

type doRequestOption func(opts *doRequestOptions)

type requestModifier func(req *http.Request)

type responseVerifier func(res *http.Response, body []byte) error

type doRequestOptions struct {
	requestPayload   any
	requestModifiers []requestModifier
	responseVerifier responseVerifier
}

func withExpectedStatusCode(statusCode int) doRequestOption {
	return func(opts *doRequestOptions) {
		opts.responseVerifier = func(res *http.Response, body []byte) error {
			if res.StatusCode == statusCode {
				return nil
			}
			return handleErrWithNotFound(fmt.Errorf("expected status %d, but got %d", statusCode, res.StatusCode), res.StatusCode, body)
		}
	}
}

func withAccept(accept string) doRequestOption {
	return withHeader("Accept", accept)
}

func withHeader(key, value string) doRequestOption {
	return func(opts *doRequestOptions) {
		opts.requestModifiers = append(opts.requestModifiers, func(req *http.Request) {
			req.Header.Set(key, value)
		})
	}
}

func withPayload(payload any, contentType string) doRequestOption {
	return func(opts *doRequestOptions) {
		// always provide Accept header with the same value as content-type,
		// as meshObject API currently does not version that differently.
		// that convention can still be overridden/broken by a later withAccept option
		withAccept(contentType)(opts)
		withHeader("Content-Type", contentType)(opts)
		opts.requestPayload = payload
	}
}

func ensureSuccessfulRequest(opts *doRequestOptions) {
	opts.responseVerifier = func(res *http.Response, body []byte) error {
		if res.StatusCode >= 200 && res.StatusCode <= 299 {
			return nil
		}
		return handleErrWithNotFound(fmt.Errorf("request failed with status %d (not 2XX successful)", res.StatusCode), res.StatusCode, body)
	}
}

func handleErrWithNotFound(err error, statusCode int, body []byte) error {
	errs := []error{err, fmt.Errorf("error body: %s", string(body))}
	if statusCode == http.StatusNotFound {
		errs = append([]error{errNotFound}, errs...)
	}
	return errors.Join(errs...)
}

func (c *MeshStackProviderClient) doAuthenticatedRequest(method string, url *url.URL, options ...doRequestOption) ([]byte, error) {
	// prepend (aka insert at 0) some default options such that given options may be overridden by caller
	options = slices.Insert(options, 0,
		withHeader("User-Agent", "meshStack Terraform Provider"),
		// by default, verify successful response
		// can be made more specific with withExpectedStatusCode option
		ensureSuccessfulRequest,
	)
	opts := doRequestOptions{}
	for _, option := range options {
		option(&opts)
	}

	var requestBody io.ReadWriter
	if opts.requestPayload != nil {
		requestBody = new(bytes.Buffer)
		if err := json.NewEncoder(requestBody).Encode(opts.requestPayload); err != nil {
			return nil, fmt.Errorf("failed to encode request body payload: %w", err)
		}
	}

	req, err := http.NewRequest(method, url.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for _, requestModifier := range opts.requestModifiers {
		requestModifier(req)
	}
	// log request before adding auth
	log.Println(req)

	// add authentication
	if err := c.ensureValidToken(); err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.token)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	log.Println(res)

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body, status code %d: %w", res.StatusCode, err)
	}
	log.Printf("Got response body with %d bytes", len(responseBody))
	// always return responseBody, even if the response is not successfully verified
	// this allows clients to investigate the responseBody even further if desirable.
	return responseBody, opts.responseVerifier(res, responseBody)
}

func (c *MeshStackProviderClient) deleteMeshObject(targetUrl *url.URL, expectedStatus int) (err error) {
	_, err = c.doAuthenticatedRequest("DELETE", targetUrl, withExpectedStatusCode(expectedStatus))
	return
}

func unmarshalBody[T any](body []byte, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	var target T
	if err := json.Unmarshal(body, &target); err != nil {
		return nil, err
	}
	return &target, nil
}

func unmarshalBodyIfPresent[T any](body []byte, err error) (*T, error) {
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	return unmarshalBody[T](body, err)
}

// paginatedResponse is a generic structure for HAL paginated responses
type paginatedResponse[T any] struct {
	Embedded map[string][]T `json:"_embedded"`
	Page     struct {
		Size          int `json:"size"`
		TotalElements int `json:"totalElements"`
		TotalPages    int `json:"totalPages"`
		Number        int `json:"number"`
	} `json:"page"`
}

// unmarshalPaginatedBody unmarshalls a paginated HAL response and extracts items using the provided key
func unmarshalPaginatedBody[T any](body []byte, err error, embeddedKey string) ([]T, *paginatedResponse[T], error) {
	if err != nil {
		return nil, nil, err
	}
	var response paginatedResponse[T]
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, nil, err
	}
	items := response.Embedded[embeddedKey]
	return items, &response, nil
}
