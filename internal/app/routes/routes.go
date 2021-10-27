package routes

import (
	"github.com/gorilla/mux"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/auth"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/balance"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/orderslist"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"net/http"
)

// Router define routes priority
func Router(s storage.Storage, l *zap.Logger, c broker.QueueBroker) *mux.Router {
	rtr := mux.NewRouter()
	// Registration users
	rtr.Handle("/api/user/register", registration.New(l, s)).Methods(http.MethodPost)
	// HasAuth user
	rtr.Handle("/api/user/login", auth.New(l, s)).Methods(http.MethodPost)
	// Order register
	rtr.Handle("/api/user/orders", order.New(l, s, c)).Methods(http.MethodPost)
	// Order list
	rtr.Handle("/api/user/orders", orderslist.New(l, s)).Methods(http.MethodGet)
	// Get user balance
	rtr.Handle("/api/user/balance", balance.New(l, s)).Methods(http.MethodGet)

	return rtr
}
