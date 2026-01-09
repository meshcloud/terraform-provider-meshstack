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
	"time"
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
	const (
		apiMeshObjectsRoot = "/api/meshobjects"
	)
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
	loginUrl := c.url.JoinPath("/api/login")

	type loginRequest struct {
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}

	type loginResponse struct {
		Token     string `json:"access_token"`
		ExpireSec int    `json:"expires_in"`
	}

	loginResult, err := unmarshalBody[loginResponse](c.doRequest("POST", loginUrl,
		withPayload(loginRequest{ClientId: c.apiKey, ClientSecret: c.apiSecret}, "application/json")),
	)
	if err != nil {
		return fmt.Errorf("login request to %s with API Key '%s' failed: %w", loginUrl, c.apiKey, err)
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

func (c *MeshStackProviderClient) doRequest(method string, url *url.URL, options ...doRequestOption) ([]byte, error) {
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
		var err error
		url, err = url.Parse(url.String())
		if err != nil {
			panic("cloning URL failed: " + err.Error())
		}
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

func (c *MeshStackProviderClient) doAuthenticatedRequest(method string, url *url.URL, options ...doRequestOption) ([]byte, error) {
	if err := c.ensureValidToken(); err != nil {
		return nil, err
	}
	return c.doRequest(method, url, append(options,
		appendRequestModifier(func(req *http.Request) {
			// log request before adding Authorization header below
			log.Println(req)
		}),
		withHeader("Authorization", c.token),
	)...)
}

func (c *MeshStackProviderClient) doPaginatedRequest(url *url.URL, options ...doRequestOption) iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		pageNumber := 0
		for {
			body, err := c.doAuthenticatedRequest("GET", url, append(options, withUrlQuery("page", pageNumber))...)
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

func unmarshalBodyPages[T any](embeddedKey string, bodyPages iter.Seq2[[]byte, error]) (result []T, err error) {
	for bodyPage, err := range bodyPages {
		type embeddedResponse[T any] struct {
			Embedded map[string][]T `json:"_embedded"`
		}
		if response, err := unmarshalBody[embeddedResponse[T]](bodyPage, err); err != nil {
			return result, err
		} else if items, ok := response.Embedded[embeddedKey]; !ok {
			return result, fmt.Errorf("embedded key %s not found in paginated response", embeddedKey)
		} else {
			result = append(result, items...)
		}
	}
	return result, nil
}
