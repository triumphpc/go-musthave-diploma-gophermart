package orderslist

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg/mocks"
	mocks3 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/gobroker/mocks"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/logger"
	mocks2 "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/authchecker/mocks"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/conveyor"
	"io"
	"io/ioutil"
	"log"
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
	}

	type server struct {
		path   string
		userID int
	}

	lgr, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	storage := &mocks.MockStorage{}
	ckr := &mocks3.Executor{}

	orderHandler := order.New(lgr, storage, ckr)
	regHandler := registration.New(lgr, storage)
	handler := New(lgr, storage)

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name:    "Check order list #1",
			handler: handler,
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
			},
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "",
			},
			server: server{
				path: "/api/user/orders",
			},
		},
		{
			name:    "Check order list #2",
			handler: regHandler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/register",
				body:   "{\n    \"login\": \"login\",\n    \"password\": \"password123\"\n} ",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path:   "/api/user/register",
				userID: 1,
			},
		},
		{
			name:    "Check order list #3",
			handler: handler,
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
			},
			want: want{
				code:        http.StatusNoContent,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 1,
			},
		},
		{
			name:    "Check order list #4",
			handler: orderHandler,
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
				path:   "/api/user/orders",
				userID: 1,
			},
		},
		{
			name:    "Check order list #5",
			handler: handler,
			request: request{
				method: http.MethodGet,
				target: "/api/user/orders",
				body:   "",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json; charset=utf-8",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 1,
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

			request := httptest.NewRequest(tt.request.method, tt.request.target, r)

			// Create new recorder
			w := httptest.NewRecorder()
			// Init handler
			rtr := mux.NewRouter()
			rtr.Handle(tt.server.path, tt.handler)

			h := conveyor.Conveyor(
				rtr,
				mocks2.NewMock(lgr, storage, tt.server.userID).CheckAuth,
			)

			// Create server
			h.ServeHTTP(w, request)
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
