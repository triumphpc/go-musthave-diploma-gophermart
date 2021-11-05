// Package http helper for client request parser
package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/encoder"
	"io/ioutil"
	"net/http"
	"strconv"
)

// ErrBadRequest bad request error
var ErrBadRequest = errors.New("bad request")

// ErrInternalError some internal error
var ErrInternalError = errors.New("internal error")

// ErrInvalidOrder invalid order id
var ErrInvalidOrder = errors.New("invalid order")

// ErrNotAuth user not auth
var ErrNotAuth = errors.New("not auth")

// CookieUserIDName cookie name
const CookieUserIDName = "user_id"

// ParseJSONReq parse JSON request and convert to struct by point
func ParseJSONReq(r *http.Request, s interface{}) error {
	if r.Body == http.NoBody {
		return ErrBadRequest
	}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ErrBadRequest
	}
	err = json.Unmarshal(body, &s)
	if err != nil {
		return ErrBadRequest
	}

	return nil
}

// AuthUser auth current user and save cookie token
func AuthUser(w http.ResponseWriter) string {
	encoded := encoder.RandomString(50)

	cookie := &http.Cookie{
		Name:  CookieUserIDName,
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, cookie)

	return encoded
}

// IsValidOrder validate orders for check
func IsValidOrder(r *http.Request) (int, error) {
	if r.Body == http.NoBody {
		return 0, ErrBadRequest
	}
	orderID, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return 0, ErrBadRequest
	}
	id, err := strconv.Atoi(string(orderID))
	if err != nil {
		return id, ErrBadRequest
	}

	fmt.Println(id)
	// Check for Luhn

	// @todo temp disable
	//if !luhn.Valid(id) {
	//	return id, ErrInvalidOrder
	//}

	return id, nil
}
