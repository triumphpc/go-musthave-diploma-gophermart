// Package storage describe methods for project storage
// @author Sergey Vrulin (aka Alex Versus)
package storage

import (
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
)

type Storage interface {
	// Register put new user in storage
	Register(user user.User) error
	// HasAuth check user auth in storage
	HasAuth(user user.User) bool
	// SetToken set token to user
	SetToken(user user.User, token string) error
	// UserByToken check token in storage and get user id
	UserByToken(token string) (int, error)
	// PutOrder put order in process for check status
	PutOrder(ord order.Order) error
	// SetStatus update status for order
	SetStatus(orderCode int, status int, timeout int, points int) error
	// AddPoints add points to user
	AddPoints(userID int, points int, orderCode int) error
	// Orders get all orders by user
	Orders(userID int) ([]order.Order, error)
	// OrderByCode get order by code
	OrderByCode(code int) (order.Order, error)
	// OrdersForCheck get all orders for check in loyalty machine
	OrdersForCheck() ([]order.Order, error)
}
