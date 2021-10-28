// Package withdraw implement logic for withdrawal user points
// @author Sergey Vrulin (aka Alex Versus)
package withdraw

import (
	"encoding/json"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
	"strconv"
)

type Handler struct {
	l *zap.Logger
	s storage.Storage
}

// New constructor
func New(l *zap.Logger, s storage.Storage) *Handler {
	return &Handler{l, s}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	usr := r.Context().Value(ht.CtxUser)
	currentUser, _ := usr.(user.User)

	if currentUser.UserID == 0 {
		http.Error(w, ht.ErrNotAuth.Error(), http.StatusUnauthorized)
		return
	}

	var body []byte
	if r.Body == http.NoBody {
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	var request struct {
		Order string  `json:"order"`
		Sum   float64 `json:"sum"`
	}

	err = json.Unmarshal(body, &request)
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	orderID, err := strconv.Atoi(request.Order)
	if err != nil {
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	order, err := h.s.OrderByCode(orderID)
	if err != nil {
		http.Error(w, "", http.StatusUnprocessableEntity)
		return
	}

	// Has no points for withdraw in order
	if float64(order.Accrual) < request.Sum || currentUser.Points < request.Sum {
		http.Error(w, "", http.StatusPaymentRequired)
		return
	}

	if err := h.s.Withdraw(order, request.Sum); err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// here run some logic for withdraw
	// no implement

	w.WriteHeader(http.StatusOK)
}
