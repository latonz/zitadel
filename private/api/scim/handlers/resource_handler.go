package handlers

import "context"

type ResourceHandler interface {
	Get(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}
