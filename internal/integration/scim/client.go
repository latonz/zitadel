package scim

import (
	"context"
	"encoding/json"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
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

func (c *ResourceClient) Delete(ctx context.Context, authHeader string, id string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseUrl+"/"+id, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", authHeader)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusNoContent {
		scimErr := new(ScimError)
		err = json.NewDecoder(resp.Body).Decode(scimErr)
		logging.OnError(err).Panic("Failed decoding scim error")
		return scimErr
	}

	return nil
}

func (err *ScimError) Error() string {
	return "scim error: " + err.Detail
}
