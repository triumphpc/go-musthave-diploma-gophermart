// Code generated by mockery 2.9.0. DO NOT EDIT.

package mocks

import (
	"database/sql"
	mock "github.com/stretchr/testify/mock"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"
	"time"

	user "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
)

// MockStorage is an autogenerated mock type for the MockStorage type
type MockStorage struct {
	mock.Mock
	storage    map[string]user.User
	tokens     map[string]string
	orders     map[int]int
	ordercodes map[int]int
	userpoints map[int]int
}

// Auth provides a mock function with given fields: u
func (_m *MockStorage) HasAuth(u user.User) bool {
	if _m.storage == nil {
		_m.storage = make(map[string]user.User)
	}

	if _, ok := _m.storage[u.Login]; ok {
		return true
	}
	return false
}

// Register provides a mock function with given fields: u
func (_m *MockStorage) Register(u user.User) error {
	if _m.storage == nil {
		_m.storage = make(map[string]user.User)
	}

	if _, ok := _m.storage[u.Login]; ok {
		return pg.ErrLoginAlreadyExist
	}
	_m.storage[u.Login] = u

	return nil
}

// SetToken provides a mock function with given fields: u
func (_m *MockStorage) SetToken(u user.User, t string) error {
	if _m.tokens == nil {
		_m.tokens = make(map[string]string)
	}
	_m.tokens[u.Login] = t

	return nil
}

// UserByToken without logic
func (_m *MockStorage) UserByToken(t string) (int, error) {
	return 0, nil
}

// PutOrder put order in process for check status
func (_m *MockStorage) PutOrder(ord order.Order) error {
	if _m.orders == nil {
		_m.orders = make(map[int]int)
	}
	for _, v := range _m.orders {
		if v == ord.Code {
			return pg.ErrOrderAlreadyExist
		}
	}
	_m.orders[ord.UserID] = ord.Code

	return nil
}

// HasOrder check order in mock storage
func (_m *MockStorage) HasOrder(userID int, code int) bool {
	if _m.orders == nil {
		_m.orders = make(map[int]int)
	}
	for k, v := range _m.orders {
		if v == code && k == userID {
			return true
		}
	}

	return false
}

// SetStatus update status for order
func (_m *MockStorage) SetStatus(orderCode int, status int, timeout int, points int) error {
	if _m.ordercodes == nil {
		_m.ordercodes = make(map[int]int)
	}
	_m.ordercodes[orderCode] = status
	return nil
}

// AddPoints add points to user
func (_m *MockStorage) AddPoints(userID int, points int, orderCode int) error {
	_m.SetStatus(orderCode, order.PROCESSED, 0, 20)
	_m.userpoints[userID] += points

	return nil
}

// Orders get all orders by user
func (_m *MockStorage) Orders(userID int) ([]order.Order, error) {
	var orders []order.Order

	for k, v := range _m.orders {
		if k == userID {
			var userOrder order.Order
			userOrder.UploadedAt = jsontime.JSONTime(time.Now())
			userOrder.Accrual = 30
			userOrder.Code = v
			userOrder.CheckStatus = "PROCESSED"
			orders = append(orders, userOrder)
		}
	}

	return orders, nil
}

// OrderByCode get orde by code
func (_m *MockStorage) OrderByCode(code int) (order.Order, error) {
	userOrder := order.Order{}

	for k, v := range _m.orders {
		if v == code {
			userOrder.Code = code
			userOrder.UserID = k
			return userOrder, nil
		}
	}

	return userOrder, sql.ErrNoRows
}

func (_m *MockStorage) OrdersForCheck() ([]order.Order, error) {
	var orders []order.Order
	var userOrder order.Order
	userOrder.UploadedAt = jsontime.JSONTime(time.Now())
	userOrder.Accrual = 30
	userOrder.CheckStatus = "NEW"
	orders = append(orders, userOrder)

	return orders, nil
}
