package middleware

import (
	zhttp "github.com/zitadel/zitadel/internal/api/http"
	"net/http"
)

const (
	ContentTypeScim = "application/scim+json"
)

func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(zhttp.ContentType, ContentTypeScim)
		next.ServeHTTP(w, r)
	})
}
