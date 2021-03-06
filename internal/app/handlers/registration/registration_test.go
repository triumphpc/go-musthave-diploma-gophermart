package registration

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg/mocks"
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
	str := &mocks.MockStorage{}
	handler := Handler{lgr, str}

	tests := []struct {
		name    string
		want    want
		request request
		handler http.Handler
		server  server
	}{
		{
			name:    "Check registration #1",
			handler: handler,
			request: request{
				method: http.MethodGet,
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
			name:    "Check registration #2",
			handler: handler,
			request: request{
				method: http.MethodGet,
				target: "/api/user/register",
				body:   "{\n    \"login\": \"login\",\n    \"password\": \"password123\"\n} ",
			},
			want: want{
				code:        http.StatusConflict,
				contentType: "",
			},
			server: server{
				path: "/api/user/register",
			},
		},
		{
			name:    "Check registration #3",
			handler: handler,
			request: request{
				method: http.MethodGet,
				target: "/api/user/register",
				body:   "}/",
			},
			want: want{
				code:        http.StatusBadRequest,
				contentType: "",
			},
			server: server{
				path: "/api/user/register",
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
