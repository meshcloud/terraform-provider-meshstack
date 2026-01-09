package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
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

func NewClient(rootUrl *url.URL, apiKey string, apiSecret string) MeshStackProviderClient {
	// Initialize httpClient for typed clients
	c := &httpClient{
		Client:    http.Client{Timeout: 5 * time.Minute},
		RootUrl:   rootUrl,
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
	}
	return MeshStackProviderClient{
		MeshBuildingBlockClient{newMeshObjectClient[MeshBuildingBlock](c, "meshBuildingBlock", "v1")},
		MeshBuildingBlockV2Client{newMeshObjectClient[MeshBuildingBlockV2](c, "meshBuildingBlock", "v2-preview")},
		MeshIntegrationClient{newMeshObjectClient[MeshIntegration](c, "meshIntegration", "v1-preview")},
		MeshLandingZoneClient{newMeshObjectClient[MeshLandingZone](c, "meshLandingZone", "v1-preview")},
		MeshLocationClient{newMeshObjectClient[MeshLocation](c, "meshLocation", "v1-preview")},
		MeshPaymentMethodClient{newMeshObjectClient[MeshPaymentMethod](c, "meshPaymentMethod", "v2")},
		MeshPlatformClient{newMeshObjectClient[MeshPlatform](c, "meshPlatform", "v2-preview")},
		MeshProjectClient{newMeshObjectClient[MeshProject](c, "meshProject", "v2")},
		MeshProjectGroupBindingClient{newMeshObjectClient[MeshProjectBinding](c, "meshProjectGroupBinding", "v3", "meshprojectbindings", "groupbindings")},
		MeshProjectUserBindingClient{newMeshObjectClient[MeshProjectBinding](c, "meshProjectUserBinding", "v3", "meshprojectbindings", "userbindings")},
		MeshTagDefinitionClient{newMeshObjectClient[MeshTagDefinition](c, "meshTagDefinition", "v1")},
		MeshTenantClient{newMeshObjectClient[MeshTenant](c, "meshTenant", "v3")},
		MeshTenantV4Client{newMeshObjectClient[MeshTenantV4](c, "meshTenant", "v4-preview")},
		MeshWorkspaceClient{newMeshObjectClient[MeshWorkspace](c, "meshWorkspace", "v2")},
		MeshWorkspaceGroupBindingClient{newMeshObjectClient[MeshWorkspaceBinding](c, "meshWorkspaceGroupBinding", "v2", "meshworkspacebindings", "groupbindings")},
		MeshWorkspaceUserBindingClient{newMeshObjectClient[MeshWorkspaceBinding](c, "meshWorkspaceUserBinding", "v2", "meshworkspacebindings", "userbindings")},
	}
}

type httpClient struct {
	http.Client
	RootUrl     *url.URL
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

func newMeshObjectClient[M any](client *httpClient, name, apiVersion string, explicitApiPaths ...string) meshObjectClient[M] {
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

func pluralizeName(name string) string {
	return fmt.Sprintf("%ss", name)
}

func (o meshObjectClient[M]) mediaType() string {
	return fmt.Sprintf("application/vnd.meshcloud.api.%s.%s.hal+json", o.Name, o.ApiVersion)
}

func (c meshObjectClient[M]) get(id string) (*M, error) {
	return unmarshalBodyIfPresent[M](c.doAuthenticatedRequest(http.MethodGet, c.ApiUrl.JoinPath(id), withAccept(c.mediaType())))
}

func (c meshObjectClient[M]) list(options ...doRequestOption) ([]M, error) {
	return unmarshalBodyPages[M](pluralizeName(c.Name), c.doPaginatedRequest(c.ApiUrl, append(options, withAccept(c.mediaType()))...))
}

func (c meshObjectClient[M]) post(payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthenticatedRequest(http.MethodPost, c.ApiUrl, withPayload(payload, c.mediaType())))
}

func (c meshObjectClient[M]) put(id string, payload any) (*M, error) {
	return unmarshalBody[M](c.doAuthenticatedRequest(http.MethodPut, c.ApiUrl.JoinPath(id), withPayload(payload, c.mediaType())))
}

func (c meshObjectClient[M]) delete(id string) (err error) {
	_, err = c.doAuthenticatedRequest(http.MethodDelete, c.ApiUrl.JoinPath(id), withAccept(c.mediaType()))
	return
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

type urlModifier func(url *url.URL)

type requestModifier func(req *http.Request)

type doRequestOptions struct {
	urlModifiers     []urlModifier
	requestPayload   any
	requestModifiers []requestModifier
}

func appendUrlModifier(modifier urlModifier) doRequestOption {
	return func(opts *doRequestOptions) {
		opts.urlModifiers = append(opts.urlModifiers, modifier)
	}
}

func appendRequestModifier(modifier requestModifier) doRequestOption {
	return func(opts *doRequestOptions) {
		opts.requestModifiers = append(opts.requestModifiers, modifier)
	}
}

func withUrlQuery(key string, value any) doRequestOption {
	return appendUrlModifier(func(url *url.URL) {
		var valueStr string
		if stringerValue, ok := value.(fmt.Stringer); ok {
			valueStr = stringerValue.String()
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}
		query := url.Query()
		query.Set(key, valueStr)
		url.RawQuery = query.Encode()
	})
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
		withHeader("User-Agent", "meshStack Terraform Provider"),
	)
	opts := doRequestOptions{}
	for _, option := range options {
		option(&opts)
	}

	if len(opts.urlModifiers) > 0 {
		// clone url to prevent modifiers edit the given URL (it's sad that this is a pointer actually)
		// ignoring the error is fine as this always succeeds parsing from String()
		url, _ = url.Parse(url.String())
		for _, modifier := range opts.urlModifiers {
			modifier(url)
		}
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

	res, err := c.Do(req)
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

func (c *httpClient) doPaginatedRequest(url *url.URL, options ...doRequestOption) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		pageNumber := 0
		for {
			body, err := c.doAuthenticatedRequest(http.MethodGet, url, append(options, withUrlQuery("page", pageNumber))...)
			if err != nil {
				yield(body, fmt.Errorf("cannot fetch page %d: %w", pageNumber, err))
				return
			}
			if !yield(body, nil) {
				// consumer wants to stop
				return
			}
			// Check if there are more pages to fetch
			type paginatedResponse struct {
				Page struct {
					TotalPages int `json:"totalPages"`
					Number     int `json:"number"`
				} `json:"page"`
			}
			response, err := unmarshalBody[paginatedResponse](body, err)
			if err != nil {
				yield(body, fmt.Errorf("cannot unmarshal paginated response, page %d: %w", pageNumber, err))
				return
			}
			if response.Page.Number >= response.Page.TotalPages-1 {
				return
			}
			pageNumber++
		}
	}
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

func unmarshalBodyPages[T any](embeddedKey string, bodyPages iter.Seq2[[]byte, error]) ([]T, error) {
	var result []T
	for bodyPage, pageErr := range bodyPages {
		type embeddedResponse[T any] struct {
			Embedded map[string][]T `json:"_embedded"`
		}
		if response, err := unmarshalBody[embeddedResponse[T]](bodyPage, pageErr); err != nil {
			return result, err
		} else if items, ok := response.Embedded[embeddedKey]; !ok {
			return result, fmt.Errorf("embedded key %s not found in paginated response", embeddedKey)
		} else {
			result = append(result, items...)
		}
	}
	return result, nil
}
