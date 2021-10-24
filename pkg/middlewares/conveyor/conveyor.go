package conveyor

import "net/http"

type Middleware func(http.Handler) http.Handler

// Conveyor service handlers
func Conveyor(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, middleware := range middlewares {
		h = middleware(h)
	}
	return h
}
