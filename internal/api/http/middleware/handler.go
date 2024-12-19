package middleware

import "net/http"

type HandlerFuncWithError func(w http.ResponseWriter, r *http.Request) error
