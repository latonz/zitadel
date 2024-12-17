package scim

import (
	"github.com/gorilla/mux"
	"github.com/zitadel/zitadel/internal/api/authz"
	"github.com/zitadel/zitadel/internal/command"
	"github.com/zitadel/zitadel/internal/query"
	"github.com/zitadel/zitadel/private/api/scim/handlers"
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

	router.Use(ContentTypeMiddleware)
	mapResource(router, "Users", handlers.NewUsersHandler(command, query))
	return router
}

func mapResource(router *mux.Router, resourceName string, handler handlers.ResourceHandler) {
	adapter := handlers.NewResourceHandlerAdapter(handler)
	resourceRouter := router.PathPrefix("/" + resourceName).Subrouter()
	resourceRouter.HandleFunc("/{id}", serrors.ErrorHandlerMiddleware(adapter.Get)).Methods(http.MethodGet)
	resourceRouter.HandleFunc("/{id}", serrors.ErrorHandlerMiddleware(adapter.Delete)).Methods(http.MethodDelete)
}
