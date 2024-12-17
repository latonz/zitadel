package middleware

import "net/http"

const (
	ContentTypeHeader = "Content-Type"
	ContentTypeScim   = "application/scim+json"
)

func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(ContentTypeHeader, ContentTypeScim)
		next.ServeHTTP(w, r)
	})
}
