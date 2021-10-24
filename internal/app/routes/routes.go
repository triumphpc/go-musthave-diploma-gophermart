package routes

import (
	"github.com/gorilla/mux"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/auth"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/checker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
	"go.uber.org/zap"
	"net/http"
)

// Router define routes priority
func Router(s pg.Storage, l *zap.Logger, c checker.Executor) *mux.Router {
	rtr := mux.NewRouter()
	// Registration users
	rtr.Handle("/api/user/register", registration.New(l, s)).Methods(http.MethodPost)
	// HasAuth user
	rtr.Handle("/api/user/login", auth.New(l, s)).Methods(http.MethodPost)
	// Order register
	rtr.Handle("/api/user/orders", order.New(l, s, c)).Methods(http.MethodPost)

	return rtr
}
