package scim

import (
	"context"
	"encoding/json"
	"github.com/zitadel/logging"
	zhttp "github.com/zitadel/zitadel/internal/api/http"
	"github.com/zitadel/zitadel/private/api/scim/middleware"
	"github.com/zitadel/zitadel/private/api/scim/resources"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"google.golang.org/grpc/metadata"
	"io"
	"net/http"
)

type Client struct {
	Users *ResourceClient
}

type ResourceClient struct {
	client  *http.Client
	baseUrl string
}

type ScimError struct {
	Schemas  []string `json:"schemas"`
	ScimType string   `json:"scimType"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
}

func NewScimClient(target string) *Client {
	target = "http://" + target + schemas.HandlerPrefix
	client := &http.Client{}
	return &Client{
		Users: &ResourceClient{
			client:  client,
			baseUrl: target + "/Users",
		},
	}
}

func (c *ResourceClient) Create(ctx context.Context, user *resources.ScimUser) (*resources.ScimUser, error) {
	responseUser := new(resources.ScimUser)
	err := c.doWithJsonBody(ctx, http.MethodPost, "", user, responseUser)
	return responseUser, err
}

func (c *ResourceClient) CreateRaw(ctx context.Context, body io.Reader) (*resources.ScimUser, error) {
	user := new(resources.ScimUser)
	err := c.doWithBody(ctx, http.MethodPost, "", body, user)
	return user, err
}

func (c *ResourceClient) Delete(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/"+id)
}

func (c *ResourceClient) do(ctx context.Context, method, url string) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+url, nil)
	if err != nil {
		return err
	}

	return c.doReq(req, nil)
}

func (c *ResourceClient) doWithJsonBody(ctx context.Context, method, url string, body interface{}, responseEntity interface{}) error {
	rawBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return c.doWithJsonBody(ctx, method, url, rawBody, responseEntity)
}

func (c *ResourceClient) doWithBody(ctx context.Context, method, url string, body io.Reader, responseEntity interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+url, body)
	if err != nil {
		return err
	}

	req.Header.Set(zhttp.ContentType, middleware.ContentTypeScim)
	return c.doReq(req, responseEntity)
}

func (c *ResourceClient) doReq(req *http.Request, responseEntity interface{}) error {
	addTokenAsHeader(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if (resp.StatusCode % 100) != 2 {
		return readScimError(resp)
	}

	if responseEntity == nil {
		return nil
	}

	err = readJson(responseEntity, resp)
	return err
}

func addTokenAsHeader(req *http.Request) {
	md, ok := metadata.FromOutgoingContext(req.Context())
	if !ok {
		return
	}

	req.Header.Set("Authorization", md.Get("Authorization")[0])
}

func readJson(entity interface{}, resp *http.Response) error {
	defer func(body io.ReadCloser) {
		err := body.Close()
		logging.OnError(err).Panic("Failed to close response body")
	}(resp.Body)

	err := json.NewDecoder(resp.Body).Decode(entity)
	logging.OnError(err).Panic("Failed decoding entity")
	return err
}

func readScimError(resp *http.Response) error {
	scimErr := new(ScimError)
	return readJson(scimErr, resp)
}

func (err *ScimError) Error() string {
	return "scim error: " + err.Detail
}
