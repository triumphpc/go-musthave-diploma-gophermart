package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	"github.com/triumphpc/go-musthave-diploma-gophermart/migrations"
	"go.uber.org/zap"
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
	PutOrder(userID int, code int) error
	// HasOrder check order from current user
	HasOrder(userID int, code int) bool
	// SetStatus update status for order
	SetStatus(orderCode int, status int) error
	// AddPoints add points to user
	AddPoints(userID int, points int, orderCode int) error
}

// Pg storage
type Pg struct {
	db *sql.DB
	l  *zap.Logger
}

// ErrLoginAlreadyExist if login already exist in storage
var ErrLoginAlreadyExist = errors.New("already exist")

// ErrUserNotFound if not found user by code
var ErrUserNotFound = errors.New("user not exist")

// ErrOrderAlreadyExist if found order code
var ErrOrderAlreadyExist = errors.New("order exists")

// sqlNewRecord for new record in db
const sqlNewUser = "INSERT INTO users (id, login, password) VALUES (default, $1, $2)"

// sqlGetUser check user
const sqlGetUser = "SELECT 1 FROM users WHERE login=$1 AND password=$2"

// sqlUpdateToken for set delete flag
const sqlUpdateToken = "UPDATE users SET auth_token=$1 WHERE login=$2"

// sqlCheckToken get user id by token
const sqlCheckToken = "SELECT id FROM users WHERE auth_token=$1"

// sqlNewOrder create new order
const sqlNewOrder = "INSERT INTO orders (id, user_id, code, check_status) VALUES (default, $1, $2, $3)"

// sqlGetCode check code
const sqlGetCode = "SELECT 1 FROM orders WHERE user_id=$1 AND code=$2"

// sqlUpdateStatus update status order
const sqlUpdateStatus = "UPDATE orders SET check_status=$1 WHERE code=$2"

// sqlAddPoints update user points
const sqlAddPoints = "UPDATE users SET points=points+$1 WHERE id=$2"

// New New new Pg with not null fields
func New(ctx context.Context, l *zap.Logger, e *env.Env) (*Pg, error) {
	// Database init
	connect, err := sql.Open("postgres", e.DatabaseDsn)
	if err != nil {
		return nil, err
	}
	// Ping
	if err := connect.PingContext(ctx); err != nil {
		return nil, err
	}
	// Run migrations
	goose.SetBaseFS(migrations.EmbedMigrations)
	if err := goose.Up(connect, "."); err != nil {
		panic(err)
	}

	return &Pg{connect, l}, nil
}

// Close connection
func (s *Pg) Close() {
	err := s.db.Close()
	if err != nil {
		s.l.Info("Closing don't close")
	}
}

// Register register new user in storage
func (s *Pg) Register(u user.User) error {
	if _, err := s.db.Exec(sqlNewUser, u.Login, u.HexPassword()); err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == pgerrcode.UniqueViolation {
				return ErrLoginAlreadyExist
			}
			return err
		}
	}

	return nil
}

// HasAuth search user in storage
func (s *Pg) HasAuth(u user.User) bool {
	return s.rowExists(sqlGetUser, u.Login, u.HexPassword())
}

// SetToken update token to user
func (s *Pg) SetToken(u user.User, t string) error {
	_, err := s.db.Exec(sqlUpdateToken, t, u.Login)

	return err
}

// UserByToken check if token exist
func (s *Pg) UserByToken(t string) (int, error) {
	var userID int
	err := s.db.QueryRow(sqlCheckToken, t).Scan(&userID)
	if err != nil {
		return userID, ErrUserNotFound
	}

	return userID, nil
}

// PutOrder put order in storage
func (s *Pg) PutOrder(id int, code int) error {
	if _, err := s.db.Exec(sqlNewOrder, id, code, order.NEW); err != nil {
		if err, ok := err.(*pq.Error); ok {
			if err.Code == pgerrcode.UniqueViolation {
				return ErrOrderAlreadyExist
			}
			return err
		}
	}

	return nil
}

// HasOrder check order exist
func (s *Pg) HasOrder(userID int, code int) bool {
	return s.rowExists(sqlGetCode, userID, code)
}

// SetStatus update status to order by code
func (s *Pg) SetStatus(orderCode int, status int) error {
	_, err := s.db.Exec(sqlUpdateStatus, status, orderCode)

	return err
}

// Check if exist record by query
func (s *Pg) rowExists(query string, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := s.db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return false
	}

	return exists
}

// AddPoints add point sto user
func (s *Pg) AddPoints(userID int, points int, orderCode int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	_, err = s.db.Exec(sqlUpdateStatus, order.PROCESSED, orderCode)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(sqlAddPoints, points, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
