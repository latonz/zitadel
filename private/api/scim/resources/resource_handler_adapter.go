package resources

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
)

type ResourceHandlerAdapter[T ResourceHolder] struct {
	handler ResourceHandler[T]
}

func NewResourceHandlerAdapter[T ResourceHolder](handler ResourceHandler[T]) *ResourceHandlerAdapter[T] {
	return &ResourceHandlerAdapter[T]{
		handler,
	}
}

func (adapter *ResourceHandlerAdapter[T]) Create(_ http.ResponseWriter, r *http.Request) (T, error) {
	// TODO validate according to schema

	entity := adapter.handler.NewResource()
	json.NewDecoder(r.Body).Decode(entity)
	return adapter.handler.Create(r.Context(), entity)
}

func (adapter *ResourceHandlerAdapter[T]) Get(_ http.ResponseWriter, r *http.Request) (T, error) {
	id := mux.Vars(r)["id"]
	return adapter.handler.Get(r.Context(), id)
}

func (adapter *ResourceHandlerAdapter[T]) Delete(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["id"]
	if err := adapter.handler.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
