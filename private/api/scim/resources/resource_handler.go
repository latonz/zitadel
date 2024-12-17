package resources

import (
	"context"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"time"
)

type ResourceHandler[T ResourceHolder] interface {
	Get(ctx context.Context, id string) (T, error)
	Delete(ctx context.Context, id string) error
}

type Resource struct {
	Schemas []schemas.ScimSchemaType `json:"schemas"`
	Meta    *ResourceMeta            `json:"meta"`
}

type ResourceMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      time.Time `json:"created"`
	LastModified time.Time `json:"lastModified"`
	Version      string    `json:"version"`
	Location     string    `json:"location"`
}

type ResourceHolder interface {
	GetResource() *Resource
}
