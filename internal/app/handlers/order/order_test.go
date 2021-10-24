package order

import (
	"context"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	mocks3 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/checker/mocks"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg/mocks"
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
	strg := &mocks.MockStorage{}
	ckr := &mocks3.Executor{}

	handler := Handler{
		ctx: context.Background(),
		l:   lgr,
		s:   strg,
		c:   ckr,
	}

	regHndlr := registration.New(lgr, strg)

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name:    "Check order #1",
			handler: regHndlr,
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
			name:    "Check order #2",
			handler: handler,
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
				path:   "/api/user/orders",
				userID: 1,
			},
		},
		{
			name:    "Check order #3",
			handler: handler,
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
			name:    "Check order #4",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 1,
			},
		},
		{
			name:    "Check order #5",
			handler: regHndlr,
			request: request{
				method: http.MethodPost,
				target: "/api/user/register",
				body:   "{\n    \"login\": \"login2\",\n    \"password\": \"password1234\"\n} ",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path:   "/api/user/register",
				userID: 2,
			},
		},
		{
			name:    "Check order #6",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
			},
			want: want{
				code:        http.StatusConflict,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 2,
			},
		},
		{
			name:    "Check order #7",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "12345674",
			},
			want: want{
				code:        http.StatusConflict,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 3,
			},
		},
		{
			name:    "Check order #8",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "79927398713",
			},
			want: want{
				code:        http.StatusAccepted,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 3,
			},
		},
		{
			name:    "Check order #9",
			handler: handler,
			request: request{
				method: http.MethodPost,
				target: "/api/user/orders",
				body:   "79927398713",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path:   "/api/user/orders",
				userID: 3,
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
				mocks2.NewMock(lgr, strg, tt.server.userID).CheckAuth,
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
