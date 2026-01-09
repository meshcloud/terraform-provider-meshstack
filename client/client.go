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
	"reflect"
	"slices"
	"strings"
	"time"
	"unicode"
)

var (
	errNotFound = errors.New("request failed with status Not Found (404)")
)

type MeshStackProviderClient struct {
	BuildingBlock         MeshBuildingBlockClient
	BuildingBlockV2       MeshBuildingBlockV2Client
	Integration           MeshIntegrationClient
	LandingZone           MeshLandingZoneClient
	Location              MeshLocationClient
	PaymentMethod         MeshPaymentMethodClient
	Platform              MeshPlatformClient
	Project               MeshProjectClient
	ProjectGroupBinding   MeshProjectGroupBindingClient
	ProjectUserBinding    MeshProjectUserBindingClient
	TagDefinition         MeshTagDefinitionClient
	Tenant                MeshTenantClient
	TenantV4              MeshTenantV4Client
	Workspace             MeshWorkspaceClient
	WorkspaceGroupBinding MeshWorkspaceGroupBindingClient
	WorkspaceUserBinding  MeshWorkspaceUserBindingClient
}

func NewClient(rootUrl *url.URL, providerVersion, apiKey, apiSecret string) MeshStackProviderClient {
	// Initialize httpClient for typed clients
	c := &httpClient{
		Client:          http.Client{Timeout: 5 * time.Minute},
		RootUrl:         rootUrl,
		ProviderVersion: providerVersion,
		ApiKey:          apiKey,
		ApiSecret:       apiSecret,
	}
	return MeshStackProviderClient{
		newBuildingBlockClient(c),
		newBuildingBlockV2Client(c),
		newIntegrationClient(c),
		newLandingZoneClient(c),
		newLocationClient(c),
		newPaymentMethodClient(c),
		newPlatformClient(c),
		newProjectClient(c),
		newProjectGroupBindingClient(c),
		newProjectUserBindingClient(c),
		newTagDefinitionClient(c),
		newTenantClient(c),
		newTenantV4Client(c),
		newWorkspaceClient(c),
		newWorkspaceGroupBindingClient(c),
		newWorkspaceUserBindingClient(c),
	}
}

type httpClient struct {
	http.Client
	RootUrl         *url.URL
	ProviderVersion string

	ApiKey      string
	ApiSecret   string
	Token       string
	TokenExpiry time.Time
}

type meshObjectClient[M any] struct {
	*httpClient
	Name, ApiVersion string
	ApiUrl           *url.URL
}

func newMeshObjectClient[M any](client *httpClient, apiVersion string, explicitApiPaths ...string) meshObjectClient[M] {
	name := inferMeshObjectName[M]()

	if len(explicitApiPaths) == 0 {
		// infer API path from meshObject name by default (if nothing explicit is given)
		explicitApiPaths = []string{strings.ToLower(pluralizeName(name))}
	}
	// also prepend the root path for all meshObjects
	explicitApiPaths = slices.Insert(explicitApiPaths, 0, "/api/meshobjects")
	apiUrl := client.RootUrl.JoinPath(explicitApiPaths...)
	log.Printf("Using API at '%s' for meshObject '%s', version '%s'", apiUrl, name, apiVersion)
	return meshObjectClient[M]{client, name, apiVersion, apiUrl}
}

// inferMeshObjectName uses reflection to infer the meshObject name from the type parameter M.
// It converts the type name to camelCase (e.g., "MeshBuildingBlock" -> "meshBuildingBlock").
func inferMeshObjectName[M any]() string {
	var zero M
	typeName := reflect.TypeOf(zero).Name()
	return lowercaseFirst(typeName)
}

func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func pluralizeName(name string) string {
	return fmt.Sprintf("%ss", name)
}

func (c meshObjectClient[M]) meshObjectMimeType() string {
	return fmt.Sprintf("application/vnd.meshcloud.api.%s.%s.hal+json", c.Name, c.ApiVersion)
}

func (c meshObjectClient[M]) get(id string) (*M, error) {
	body, err := c.doAuthenticatedRequest(http.MethodGet, c.ApiUrl.JoinPath(id), withAccept(c.meshObjectMimeType()))
	if errors.Is(err, errNotFound) {
		return nil, nil
	}
	return unmarshalBody[M](body, err)
}

func (c meshObjectClient[M]) post(payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthenticatedRequest(http.MethodPost, c.ApiUrl, withPayload(payload, c.meshObjectMimeType())))
}

func (c meshObjectClient[M]) put(id string, payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthenticatedRequest(http.MethodPut, c.ApiUrl.JoinPath(id), withPayload(payload, c.meshObjectMimeType())))
}

func (c meshObjectClient[M]) delete(id string) (err error) {
	_, err = c.doAuthenticatedRequest(http.MethodDelete, c.ApiUrl.JoinPath(id), withAccept(c.meshObjectMimeType()))
	return
}

func (c meshObjectClient[M]) list(options ...doRequestOption) ([]M, error) {
	var result []M
	embeddedKey := pluralizeName(c.Name)
	pageNumber := 0
	for {
		body, err := c.doAuthenticatedRequest(http.MethodGet, c.ApiUrl, append(options,
			withAccept(c.meshObjectMimeType()),
			withUrlQuery("page", pageNumber),
		)...)
		if err != nil {
			return result, fmt.Errorf("cannot fetch page %d: %w", pageNumber, err)
		}
		type paginatedResponse struct {
			Embedded map[string][]M `json:"_embedded"`
			Page     struct {
				TotalPages int `json:"totalPages"`
				Number     int `json:"number"`
			} `json:"page"`
		}
		response, err := unmarshalBody[paginatedResponse](body, err)
		if err != nil {
			return result, fmt.Errorf("cannot unmarshal paginated response, page %d: %w", pageNumber, err)
		} else if items, ok := response.Embedded[embeddedKey]; !ok {
			return result, fmt.Errorf("embedded key %s not found in paginated response", embeddedKey)
		} else {
			result = append(result, items...)
		}
		// check if we've reached the end of pagination
		if response.Page.Number >= response.Page.TotalPages-1 {
			return result, nil
		}
		pageNumber++
	}
}

func (c *httpClient) login() error {
	loginApiUrl := c.RootUrl.JoinPath("/api/login")

	type loginRequest struct {
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}

	type loginResponse struct {
		Token     string `json:"access_token"`
		ExpireSec int    `json:"expires_in"`
	}

	loginResult, err := unmarshalBody[loginResponse](c.doRequest("POST", loginApiUrl,
		withPayload(loginRequest{ClientId: c.ApiKey, ClientSecret: c.ApiSecret}, "application/json")),
	)
	if err != nil {
		return fmt.Errorf("login request to %s with API Key '%s' failed: %w", loginApiUrl, c.ApiKey, err)
	}

	c.Token = fmt.Sprintf("Bearer %s", loginResult.Token)
	c.TokenExpiry = time.Now().Add(time.Second * time.Duration(loginResult.ExpireSec))
	return nil
}

func (c *httpClient) ensureValidToken() error {
	if c.Token == "" || time.Now().Add(30*time.Second).After(c.TokenExpiry) {
		return c.login()
	}
	return nil
}

type doRequestOption func(opts *doRequestOptions)

type requestModifier func(req *http.Request)

type doRequestOptions struct {
	urlQueryParams   map[string]string
	requestPayload   any
	requestModifiers []requestModifier
}

func appendRequestModifier(modifier requestModifier) doRequestOption {
	return func(opts *doRequestOptions) {
		opts.requestModifiers = append(opts.requestModifiers, modifier)
	}
}

func withUrlQuery(key string, value any) doRequestOption {
	return func(opts *doRequestOptions) {
		var valueStr string
		if stringerValue, ok := value.(fmt.Stringer); ok {
			valueStr = stringerValue.String()
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}
		if opts.urlQueryParams == nil {
			opts.urlQueryParams = map[string]string{key: valueStr}
		} else {
			opts.urlQueryParams[key] = valueStr
		}
	}
}

func withAccept(accept string) doRequestOption {
	return withHeader("Accept", accept)
}

func withHeader(key, value string) doRequestOption {
	return appendRequestModifier(func(req *http.Request) {
		req.Header.Set(key, value)
	})
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

func (c *httpClient) doRequest(method string, url *url.URL, options ...doRequestOption) ([]byte, error) {
	// prepend (aka insert at 0) some default options such that given options may be overridden by caller
	options = slices.Insert(options, 0,
		withHeader("User-Agent", fmt.Sprintf("terraform-provider-meshstack/%s", c.ProviderVersion)),
	)
	opts := doRequestOptions{}
	for _, option := range options {
		option(&opts)
	}
	req, err := c.buildRequest(method, *url, opts)
	if err != nil {
		return nil, err
	}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()
	log.Println(res)
	return c.readBodyAndCheckSuccess(res)
}

func (c *httpClient) readBodyAndCheckSuccess(res *http.Response) ([]byte, error) {
	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("cannot read response body, status code %d: %w", res.StatusCode, err)
	}
	log.Printf("Got response body with %d bytes", len(responseBody))

	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return responseBody, nil
	}
	var errs []error
	if res.StatusCode == http.StatusNotFound {
		errs = append(errs, errNotFound)
	}
	errs = append(errs,
		fmt.Errorf("request failed with status %d (not 2XX successful)", res.StatusCode),
		fmt.Errorf("error response: %s", string(responseBody)),
	)
	// always return responseBody, even if the response is not successfully verified
	// this allows clients to investigate the responseBody even further if desirable.
	return responseBody, errors.Join(errs...)
}

func (c *httpClient) buildRequest(method string, url url.URL, opts doRequestOptions) (*http.Request, error) {
	if len(opts.urlQueryParams) > 0 {
		query := url.Query()
		for k, v := range opts.urlQueryParams {
			query.Set(k, v)
		}
		// Note: url is not a pointer here,
		// so we can safely update that struct field without propagating such change to the caller!
		url.RawQuery = query.Encode()
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
	return req, err
}

func (c *httpClient) doAuthenticatedRequest(method string, url *url.URL, options ...doRequestOption) ([]byte, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, err
	}
	return c.doRequest(method, url, append(options,
		appendRequestModifier(func(req *http.Request) {
			// log request before adding Authorization header below
			log.Println(req)
		}),
		withHeader("Authorization", c.Token),
	)...)
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
