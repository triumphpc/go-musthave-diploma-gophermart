package orderslist

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mods "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	mocks2 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage/mocks"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/jsontime"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name: "Check order list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
				test:   0,
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
			name: "Check order list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
				test:   1,
			},
			want: want{
				code:        http.StatusNoContent,
				contentType: "",
			},
			server: server{
				path:     "/api/user/orders",
				withAuth: true,
			},
		},
		{
			name: "Check order list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
				test:   2,
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json; charset=utf-8",
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

				var orders []mods.Order

				var userOrder mods.Order
				userOrder.UploadedAt = jsontime.JSONTime(time.Now())
				userOrder.Accrual = 30
				userOrder.Code = 12345
				userOrder.CheckStatus = "PROCESSED"
				orders = append(orders, userOrder)

				storage.
					On("UserByToken", mock.Anything, mock.Anything).Return(
					mods.User{
						Login:  "test",
						UserID: 123,
					}, nil).
					On("Orders", mock.Anything, mock.MatchedBy(func(userID int) bool {
						return tt.request.test == 1
					})).Return([]mods.Order{}, nil).
					On("Orders", mock.Anything, mock.MatchedBy(func(userID int) bool {
						return userID == 123 && tt.request.test == 2
					})).Return(orders, nil)
			}

			handler := New(zap.NewNop(), &storage)

			// Create new recorder
			w := httptest.NewRecorder()
			// Init handler
			rtr := mux.NewRouter()
			rtr.Handle(tt.server.path, handler)

			// Create server
			rtr.ServeHTTP(w, req)
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
