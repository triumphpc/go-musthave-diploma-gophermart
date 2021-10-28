package withdraw

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	ord "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/order"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models/user"
	mocks4 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker/mocks"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg/mocks"
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
		path string
		usr  user.User
	}

	lgr, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	storage := &mocks.MockStorage{}
	usr := user.User{
		UserID: 1,
	}

	regHandler := registration.New(lgr, storage)
	handler := New(lgr, storage)

	broker := &mocks4.QueueBroker{}

	broker.On("Push", mock.MatchedBy(func(input order.Order) bool {
		// no implement
		return true
	})).Return(func(input order.Order) error {
		return nil
	}, nil)

	broker.On("Run", mock.MatchedBy(func(ctx context.Context) bool {
		// no implement
		return true
	})).Return(func(ctx context.Context) error {
		return nil
	}, nil)

	orderHandler := ord.New(lgr, storage, broker)

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name:    "Check withdraw #1",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "",
			},
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "",
			},
			server: server{
				path: "/api/user/balance/withdraw",
			},
		},
		{
			name:    "Check withdraw #2",
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
				path: "/api/user/register",
				usr:  usr,
			},
		},
		{
			name:    "Check withdraw #3",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "{\"order\": \"3\",\"sum\": 6\n}",
			},
			want: want{
				code:        http.StatusUnprocessableEntity,
				contentType: "",
			},
			server: server{
				path: "/api/user/balance/withdraw",
				usr:  usr,
			},
		},
		{
			name:    "Check withdraw #4",
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
				path: "/api/user/orders",
				usr:  usr,
			},
		},
		{
			name:    "Check withdraw #5",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "{\"order\": \"3\",\"sum\": 6\n}",
			},
			want: want{
				code:        http.StatusUnprocessableEntity,
				contentType: "",
			},
			server: server{
				path: "/api/user/balance/withdraw",
				usr:  usr,
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
				mocks2.NewMock(lgr, storage, tt.server.usr).CheckAuth,
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
