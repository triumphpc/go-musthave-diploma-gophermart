package order

import (
	"context"
	"database/sql"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	mocks4 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker/mocks"
	mocks2 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage/mocks"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/conveyor"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_ServeHTTP(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
	}

	type request struct {
		method string
		target string
		body   string
		path   string
		test   int
	}

	type server struct {
		path     string
		withAuth bool
	}

	broker := &mocks4.Publisher{}

	broker.On("Push", mock.Anything).Return(nil)

	broker.On("Run", mock.MatchedBy(func(ctx context.Context) bool {
		// no implement
		return true
	})).Return(func(ctx context.Context) error {
		return nil
	}, nil)

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name: "Check order",
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "",
			},
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: false,
			},
		},
		{
			name: "Check order",
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "",
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: true,
			},
		},
		{
			name: "Check order",
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
			},
			want: want{
				code:        http.StatusAccepted,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: true,
			},
		},
		{
			name: "Check order",
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
				test:   1,
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: true,
			},
		},
		{
			name: "Check order",
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
				test:   2,
			},
			want: want{
				code:        http.StatusConflict,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r io.Reader
			if len(tt.request.body) > 0 {
				r = strings.NewReader(tt.request.body)
			} else {
				r = nil
			}

			storage := mocks2.Storage{}
			req := httptest.NewRequest(tt.request.method, tt.request.target, r)

			if tt.server.withAuth {
				cookie := &http.Cookie{
					Name:  ht.CookieUserIDName,
					Value: "test",
					Path:  "/",
				}
				req.AddCookie(cookie)

				storage.
					On("UserByToken", mock.Anything, mock.Anything).Return(
					models.User{
						UserID:    1,
						Withdrawn: 100,
						Points:    100,
					}, nil).
					On("OrderByCode", mock.Anything, mock.MatchedBy(func(code int) bool {
						return code == 12345674 && tt.request.test == 0
					})).Return(models.Order{
					Code:   "12345674",
					UserID: 123,
				}, sql.ErrNoRows).
					On("OrderByCode", mock.Anything, mock.MatchedBy(func(code int) bool {
						return code == 12345674 && tt.request.test == 1
					})).Return(models.Order{
					Code:   "12345674",
					UserID: 1,
				}, nil).
					On("OrderByCode", mock.Anything, mock.MatchedBy(func(code int) bool {
						return code == 12345674 && tt.request.test == 2
					})).Return(models.Order{
					Code:   "12345674",
					UserID: 12345,
				}, nil).
					On("PutOrder", mock.Anything, mock.MatchedBy(func(ord models.Order) bool {
						return ord.Code == "12345674"
					})).Return(nil)

			}

			handler := Handler{
				lgr: zap.NewNop(),
				stg: &storage,
				pub: broker,
			}

			// Create new recorder
			w := httptest.NewRecorder()
			// Init handler
			rtr := mux.NewRouter()
			rtr.Handle(tt.server.path, handler)

			h := conveyor.Conveyor(
				rtr,
			)

			// Create server
			h.ServeHTTP(w, req)
			res := w.Result()

			// Check code
			assert.Equal(t, tt.want.code, res.StatusCode, "code incorrect")

			// check body
			defer res.Body.Close()
			resBody, err := ioutil.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			readLine := strings.TrimSuffix(string(resBody), "\n")
			// equal response
			if tt.want.response != "" {
				assert.Equal(t, tt.want.response, readLine)

			}

			if tt.want.code > 0 {
				assert.Equal(t, tt.want.code, res.StatusCode)
			}

			if tt.want.contentType != "" {
				assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
			}
		})
	}
}
