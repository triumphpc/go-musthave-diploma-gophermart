package order

import (
	"context"
	"database/sql"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

type Handler struct {
	lgr *zap.Logger
	stg storage.Storage
	pub broker.Publisher
	ckr checker.Controller
}

// New constructor
func New(lgr *zap.Logger, stg storage.Storage, pub broker.Publisher, ckr checker.Controller) *Handler {
	return &Handler{lgr, stg, pub, ckr}
}

// Register order
func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var currentUser models.User
	if token, err := r.Cookie(ht.CookieUserIDName); err == nil {
		currentUser, _ = h.stg.UserByToken(r.Context(), token.Value)
	}

	if currentUser.UserID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	orderCode, err := ht.IsValidOrder(r)
	if err != nil {
		if errors.Is(err, ht.ErrBadRequest) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	order, err := h.stg.OrderByCode(r.Context(), orderCode)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			h.lgr.Info("Error in get order", zap.Error(err))
			http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if order.UserID == currentUser.UserID {
			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "", http.StatusConflict)
		return
	}

	order.UserID = currentUser.UserID
	order.Code = strconv.Itoa(orderCode)

	// Create order
	if err := h.stg.PutOrder(r.Context(), order); err != nil {
		// If someone already added code
		if errors.Is(err, pg.ErrOrderAlreadyExist) {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		h.lgr.Info("Put order error", zap.Error(err))
		http.Error(w, ht.ErrInternalError.Error(), http.StatusInternalServerError)
		return
	}

	task := h.ckr.PrepareTask(context.Background(), order)
	err = h.pub.Publish(task)

	if err != nil {
		h.lgr.Info("Error handler", zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
