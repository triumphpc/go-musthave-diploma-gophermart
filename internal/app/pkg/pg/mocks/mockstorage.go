// Code generated by mockery 2.9.0. DO NOT EDIT.

package mocks

import (
	"context"
	"database/sql"
	mock "github.com/stretchr/testify/mock"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"
	"strconv"
	"time"
)

// MockStorage is an autogenerated mock type for the MockStorage type
type MockStorage struct {
	mock.Mock
	storage    map[string]models.User
	tokens     map[string]string
	orders     map[int]string
	ordercodes map[int]int
	userpoints map[int]float64
}

// Auth provides a mock function with given fields: u
func (_m *MockStorage) HasAuth(ctx context.Context, u models.User) (bool, error) {
	if _m.storage == nil {
		_m.storage = make(map[string]models.User)
	}

	if _, ok := _m.storage[u.Login]; ok {
		return true, nil
	}
	return false, nil
}

// Register provides a mock function with given fields: u
func (_m *MockStorage) Register(ctx context.Context, u models.User) error {
	if _m.storage == nil {
		_m.storage = make(map[string]models.User)
	}

	if _, ok := _m.storage[u.Login]; ok {
		return pg.ErrLoginAlreadyExist
	}
	_m.storage[u.Login] = u

	return nil
}

// SetToken provides a mock function with given fields: u
func (_m *MockStorage) SetToken(ctx context.Context, u models.User, t string) error {
	if _m.tokens == nil {
		_m.tokens = make(map[string]string)
	}
	_m.tokens[u.Login] = t

	return nil
}

// UserByToken without logic
func (_m *MockStorage) UserByToken(ctx context.Context, token string) (models.User, error) {
	usr := models.User{}
	for k, v := range _m.tokens {
		if v == token {
			usr.Login = k
		}
	}
	return usr, nil
}

// PutOrder put order in process for check status
func (_m *MockStorage) PutOrder(ctx context.Context, ord models.Order) error {
	if _m.orders == nil {
		_m.orders = make(map[int]string)
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
func (_m *MockStorage) HasOrder(ctx context.Context, userID int, code int) bool {
	if _m.orders == nil {
		_m.orders = make(map[int]string)
	}
	for k, v := range _m.orders {
		if v == strconv.Itoa(code) && k == userID {
			return true
		}
	}

	return false
}

// SetStatus update status for order
func (_m *MockStorage) SetStatus(ctx context.Context, orderCode int, status, timeout int, points float64) error {
	if _m.ordercodes == nil {
		_m.ordercodes = make(map[int]int)
	}
	_m.ordercodes[orderCode] = status
	return nil
}

// AddPoints add points to user
func (_m *MockStorage) AddPoints(ctx context.Context, userID int, points float64, orderCode int) error {
	_m.SetStatus(ctx, orderCode, models.PROCESSED, 0, 20)
	_m.userpoints[userID] += points

	return nil
}

// Orders get all orders by user
func (_m *MockStorage) Orders(ctx context.Context, userID int) ([]models.Order, error) {
	var orders []models.Order

	for k, v := range _m.orders {
		if k == userID {
			var userOrder models.Order
			userOrder.UploadedAt = jsontime.JSONTime(time.Now())
			userOrder.Accrual = 30
			userOrder.Code = v
			userOrder.CheckStatus = "PROCESSED"
			orders = append(orders, userOrder)
		}
	}

	return orders, nil
}

// OrderByCode get order by code
func (_m *MockStorage) OrderByCode(ctx context.Context, code int) (models.Order, error) {
	userOrder := models.Order{}

	for k, v := range _m.orders {
		if v == strconv.Itoa(code) {
			userOrder.Code = strconv.Itoa(code)
			userOrder.UserID = k
			return userOrder, nil
		}
	}

	return userOrder, sql.ErrNoRows
}

// OrdersForCheck implement check orders mock
func (_m *MockStorage) OrdersForCheck(ctx context.Context) ([]models.Order, error) {
	var orders []models.Order
	var userOrder models.Order
	userOrder.UploadedAt = jsontime.JSONTime(time.Now())
	userOrder.Accrual = 30
	userOrder.CheckStatus = "NEW"
	orders = append(orders, userOrder)

	return orders, nil
}

func (_m *MockStorage) Withdraw(ctx context.Context, ord models.Order, points float64) error {
	return nil
}

func (_m *MockStorage) AddWithdraw(ctx context.Context, ord models.Order, points float64) error {
	return nil
}

func (_m *MockStorage) ActiveWithdrawals(ctx context.Context) ([]models.Withdraw, error) {
	var wds []models.Withdraw
	var wd models.Withdraw
	wds = append(wds, wd)

	return wds, nil
}

func (_m *MockStorage) WithdrawsByUserID(ctx context.Context, userID int) ([]models.Withdraw, error) {
	var wds []models.Withdraw
	var wd models.Withdraw
	wds = append(wds, wd)

	return wds, nil
}

func (_m *MockStorage) Close() {
	return
}
