package scim

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/zitadel/logging"
	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/middleware"
	"github.com/zitadel/zitadel/private/api/scim/resources"
	"github.com/zitadel/zitadel/private/api/scim/serrors"
	"net/http"
)

const (
	HandlerPrefix = "/scim/v2"
)

func NewServer(
	command *command.Commands,
	query *query.Queries,
	verifier *authz.ApiTokenVerifier,
	middlewares ...func(next http.Handler) http.Handler) http.Handler {
	verifier.RegisterServer("SCIM-V2", "scim/v2", AuthMapping)
	return buildHandler(command, query, middlewares)
}

func buildHandler(command *command.Commands, query *query.Queries, middlewares []func(next http.Handler) http.Handler) http.Handler {
	router := mux.NewRouter()
	for _, m := range middlewares {
		router.Use(m)
	}

	router.Use(middleware.ContentTypeMiddleware)
	mapResource[*resources.ScimUser](router, resources.UserResourceNamePlural, resources.NewUsersHandler(command, query))
	return router
}

func mapResource[T resources.ResourceHolder](router *mux.Router, resourceName string, handler resources.ResourceHandler[T]) {
	adapter := resources.NewResourceHandlerAdapter[T](handler)
	resourceRouter := router.PathPrefix("/" + resourceName).Subrouter()
	resourceRouter.HandleFunc("/{id}", serrors.ErrorHandlerMiddleware(handleResourceResponse(adapter.Get))).Methods(http.MethodGet)
	resourceRouter.HandleFunc("/{id}", serrors.ErrorHandlerMiddleware(adapter.Delete)).Methods(http.MethodDelete)
}

func handleResourceResponse[T resources.ResourceHolder](next func(w http.ResponseWriter, r *http.Request) (T, error)) func(http.ResponseWriter, *http.Request) error {
	return func(w http.ResponseWriter, r *http.Request) error {
		entity, err := next(w, r)
		if err != nil {
			return err
		}

		w.Header().Set("Location", entity.GetResource().Meta.Location)
		w.Header().Set("ETag", entity.GetResource().Meta.Version)

		err = json.NewEncoder(w).Encode(entity)
		logging.OnError(err).Warn("scim error encoding failed")
		return nil
	}
}
