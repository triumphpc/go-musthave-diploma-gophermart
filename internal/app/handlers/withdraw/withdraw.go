// Package withdraw implement logic for withdrawal user points
// @author Sergey Vrulin (aka Alex Versus)
package withdraw

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Handler struct {
	lgr *zap.Logger
	stg storage.Storage
}

// New constructor
func New(l *zap.Logger, s storage.Storage) *Handler {
	return &Handler{l, s}
}

// request on withdraw
type request struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var currentUser models.User
	if token, err := r.Cookie(ht.CookieUserIDName); err == nil {
		currentUser, _ = h.stg.UserByToken(r.Context(), token.Value)
	}

	if currentUser.UserID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	var body []byte
	if r.Body == http.NoBody {
		h.lgr.Error("No body from request")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	req := request{}
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = strconv.Atoi(req.Order)
	if err != nil {
		h.lgr.Error("Can't convert orderID")
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	// Has no points for withdraw in order
	if currentUser.Points < req.Sum {
		http.Error(w, "", http.StatusPaymentRequired)
		return
	}

	// Order for withdraw
	order := models.Order{
		Code: req.Order,
		ID:   req.Order,
	}

	h.lgr.Info("Add to withdraw", zap.Reflect("order", order), zap.Reflect("request", req))
	if err := h.stg.AddWithdraw(r.Context(), order, req.Sum); err != nil {
		h.lgr.Error("Don't add withdraw", zap.Error(err))
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// here run some logic for withdraw
	// no implement

	w.WriteHeader(http.StatusOK)
}
