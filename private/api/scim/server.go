package scim

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/api/authz"
	api_http "github.com/zitadel/zitadel/internal/api/http"
	zhttp_middlware "github.com/zitadel/zitadel/internal/api/http/middleware"
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
	middlewares ...zhttp_middlware.MiddlewareWithErrorFunc) http.Handler {
	verifier.RegisterServer("SCIM-V2", schemas.HandlerPrefix, AuthMapping)
	return buildHandler(command, query, userCodeAlg, config, middlewares...)
}

func buildHandler(
	command *command.Commands,
	query *query.Queries,
	userCodeAlg crypto.EncryptionAlgorithm,
	cfg *scim_config.Config,
	middlewares ...zhttp_middlware.MiddlewareWithErrorFunc) http.Handler {

	router := mux.NewRouter()

	// handle non-error related middleware
	// TODO org in path
	router.Use(middleware.ContentTypeMiddleware)

	scimMiddleware := zhttp_middlware.ChainedWithErrorHandler(serrors.ErrorHandler, middlewares...)
	mapResource(router, scimMiddleware, resources.NewUsersHandler(command, query, userCodeAlg, cfg))
	return router
}

func mapResource[T resources.ResourceHolder](router *mux.Router, mw zhttp_middlware.ErrorHandlerFunc, handler resources.ResourceHandler[T]) {
	adapter := resources.NewResourceHandlerAdapter[T](handler)
	resourceRouter := router.PathPrefix("/" + string(handler.ResourceNamePlural())).Subrouter()

	resourceRouter.Handle("", mw(handleResourceCreatedResponse(adapter.Create))).Methods(http.MethodPost)
	resourceRouter.Handle("", mw(handleJsonResponse(adapter.List))).Methods(http.MethodGet)
	resourceRouter.Handle("/.search", mw(handleJsonResponse(adapter.List))).Methods(http.MethodPost)
	resourceRouter.Handle("/{id}", mw(handleResourceResponse(adapter.Get))).Methods(http.MethodGet)
	resourceRouter.Handle("/{id}", mw(handleResourceResponse(adapter.Replace))).Methods(http.MethodPut)
	resourceRouter.Handle("/{id}", mw(handleEmptyResponse(adapter.Delete))).Methods(http.MethodDelete)
}

func handleJsonResponse[T any](next func(r *http.Request) (T, error)) zhttp_middlware.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim json response encoding failed")
		return nil
	}
}

func handleResourceCreatedResponse[T resources.ResourceHolder](next func(*http.Request) (T, error)) zhttp_middlware.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		resource := entity.GetResource()
		w.Header().Set(api_http.Location, resource.Meta.Location)
		w.WriteHeader(http.StatusCreated)

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim json response encoding failed")
		return nil
	}
}

func handleResourceResponse[T resources.ResourceHolder](next func(*http.Request) (T, error)) zhttp_middlware.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(r)
		if err != nil {
			return err
		}

		resource := entity.GetResource()
		w.Header().Set(api_http.ContentLocation, resource.Meta.Location)

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim json response encoding failed")
		return nil
	}
}

func handleEmptyResponse(next func(*http.Request) error) zhttp_middlware.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := next(r)
		if err != nil {
			return err
		}

		w.WriteHeader(http.StatusNoContent)
		return nil
	}
}
