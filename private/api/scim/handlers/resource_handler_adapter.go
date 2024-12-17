package handlers

import (
	"github.com/gorilla/mux"
	"net/http"
)

type ResourceHandlerAdapter struct {
	handler ResourceHandler
}

func NewResourceHandlerAdapter(handler ResourceHandler) *ResourceHandlerAdapter {
	return &ResourceHandlerAdapter{
		handler,
	}
}

func (adapter *ResourceHandlerAdapter) Get(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["id"]
	err := adapter.handler.Get(r.Context(), id)
	return err
}

func (adapter *ResourceHandlerAdapter) Delete(w http.ResponseWriter, r *http.Request) error {
	id := mux.Vars(r)["id"]
	if err := adapter.handler.Delete(r.Context(), id); err != nil {
		return err
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
}
