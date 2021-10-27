package auth

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/handlers/registration"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pg/mocks"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/logger"
	"go.uber.org/zap"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"reflect"
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
	}

	lgr, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	strg := &mocks.MockStorage{}
	hdlr := Handler{lgr, strg}

	regHndlr := registration.New(lgr, strg)

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name:    "Check auth #1",
			handler: hdlr,
			request: request{
				method: http.MethodPost,
				target: "/api/user/login",
				body:   "{\n    \"login\": \"login\",\n    \"password\": \"password123\"\n} ",
			},
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "",
			},
			server: server{
				path: "/api/user/login",
			},
		},
		{
			name:    "Check auth #2",
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
				path: "/api/user/register",
			},
		},
		{
			name:    "Check auth #3",
			handler: hdlr,
			request: request{
				method: http.MethodPost,
				target: "/api/user/login",
				body:   "{\n    \"login\": \"login\",\n    \"password\": \"password123\"\n} ",
			},
			want: want{
				code:        http.StatusOK,
				contentType: "",
			},
			server: server{
				path: "/api/user/login",
			},
		},
		{
			name:    "Check auth #4",
			handler: hdlr,
			request: request{
				method: http.MethodPost,
				target: "/api/user/login",
				body:   "{\n    \"loin\": \"login\",\n    \"password\": \"passwrd123\"\n} ",
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
			},
			server: server{
				path: "/api/user/login",
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

			// Create server
			rtr.ServeHTTP(w, request)
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

func TestNew(t *testing.T) {
	type args struct {
		l *zap.Logger
		s storage.Storage
	}

	lgr, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	stg := &mocks.MockStorage{}
	flds := args{lgr, stg}

	tests := []struct {
		name string
		args args
		want *Handler
	}{
		{
			name: "New check",
			args: flds,
			want: &Handler{lgr, stg},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := New(tt.args.l, tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("New() = %v, want %v", got, tt.want)
			}
		})
	}
}
