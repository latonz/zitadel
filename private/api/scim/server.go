package scim

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/api/authz"
	api_http "github.com/zitadel/zitadel/internal/api/http"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/crypto"
	"github.com/zitadel/zitadel/internal/query"
	scim_config "github.com/zitadel/zitadel/private/api/scim/config"
	"github.com/zitadel/zitadel/private/api/scim/middleware"
	"github.com/zitadel/zitadel/private/api/scim/resources"
	"github.com/zitadel/zitadel/private/api/scim/schemas"
	"github.com/zitadel/zitadel/private/api/scim/serrors"
	"net/http"
)

func NewServer(
	command *command.Commands,
	query *query.Queries,
	verifier *authz.ApiTokenVerifier,
	userCodeAlg crypto.EncryptionAlgorithm,
	config *scim_config.Config,
	middlewares ...func(next http.Handler) http.Handler) http.Handler {
	verifier.RegisterServer("SCIM-V2", schemas.HandlerPrefix, AuthMapping)
	return buildHandler(command, query, userCodeAlg, config, middlewares)
}

func buildHandler(
	command *command.Commands,
	query *query.Queries,
	userCodeAlg crypto.EncryptionAlgorithm,
	cfg *scim_config.Config,
	middlewares []func(next http.Handler) http.Handler) http.Handler {
	router := mux.NewRouter()
	for _, m := range middlewares {
		router.Use(m)
	}

	router.Use(middleware.ContentTypeMiddleware)
	mapResource(router, resources.NewUsersHandler(command, query, userCodeAlg, cfg))
	return router
}

func mapResource[T resources.ResourceHolder](router *mux.Router, handler resources.ResourceHandler[T]) {
	adapter := resources.NewResourceHandlerAdapter[T](handler)
	resourceRouter := router.PathPrefix("/" + handler.ResourceNamePlural()).Subrouter()

	resourceRouter.HandleFunc("", handleResourceCreatedResponse(adapter.Create)).Methods(http.MethodPost)
	resourceRouter.HandleFunc("", handleJsonResponse(adapter.List)).Methods(http.MethodGet)
	resourceRouter.HandleFunc("/.search", handleJsonResponse(adapter.List)).Methods(http.MethodPost)
	resourceRouter.HandleFunc("/{id}", handleResourceResponse(adapter.Get)).Methods(http.MethodGet)
	resourceRouter.HandleFunc("/{id}", handleEmptyResponse(adapter.Delete)).Methods(http.MethodDelete)
}

func handleJsonResponse[T any](next func(r *http.Request) (T, error)) func(w http.ResponseWriter, r *http.Request) {
	return serrors.ErrorHandlerMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim error encoding failed")
		return nil
	})
}

func handleResourceCreatedResponse[T resources.ResourceHolder](next func(r *http.Request) (T, error)) func(http.ResponseWriter, *http.Request) {
	return serrors.ErrorHandlerMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		resource := entity.GetResource()
		w.Header().Set(api_http.Location, resource.Meta.Location)
		w.Header().Set(api_http.Etag, resource.Meta.Version)
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim error encoding failed")
		return nil
	})
}

func handleResourceResponse[T resources.ResourceHolder](next func(r *http.Request) (T, error)) func(http.ResponseWriter, *http.Request) {
	return serrors.ErrorHandlerMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		resource := entity.GetResource()
		if r.Header.Get(api_http.IfNoneMatch) == resource.Meta.Version {
			w.WriteHeader(http.StatusNotModified)
			return nil
		}

		w.Header().Set(api_http.ContentLocation, resource.Meta.Location)
		w.Header().Set(api_http.Etag, resource.Meta.Version)

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim error encoding failed")
		return nil
	})
}

func handleEmptyResponse(next func(r *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return serrors.ErrorHandlerMiddleware(func(w http.ResponseWriter, r *http.Request) error {
		err := next(r)
		if err != nil {
			return err
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	})
}
