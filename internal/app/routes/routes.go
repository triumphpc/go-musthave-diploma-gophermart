package routes

import (
	"github.com/gorilla/mux"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/auth"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/balance"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/orderslist"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/withdraw"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/withdrawallist"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"go.uber.org/zap"
	"net/http"
)

// Router define routes priority
func Router(stg storage.Storage, lgr *zap.Logger, pub broker.Publisher) *mux.Router {
	rtr := mux.NewRouter()
	// Registration users
	rtr.Handle("/api/user/register", registration.New(lgr, stg)).Methods(http.MethodPost)
	// HasAuth user
	rtr.Handle("/api/user/login", auth.New(lgr, stg)).Methods(http.MethodPost)
	// Order register
	rtr.Handle("/api/user/orders", order.New(lgr, stg, pub)).Methods(http.MethodPost)
	// Order list
	rtr.Handle("/api/user/orders", orderslist.New(lgr, stg)).Methods(http.MethodGet)
	// Get user balance
	rtr.Handle("/api/user/balance", balance.New(lgr, stg)).Methods(http.MethodGet)
	// Withdraw request
	rtr.Handle("/api/user/balance/withdraw", withdraw.New(lgr, stg)).Methods(http.MethodPost)
	// Get withdrawals statuses
	rtr.Handle("/api/user/balance/withdrawals", withdrawallist.New(lgr, stg)).Methods(http.MethodGet)

	return rtr
}
