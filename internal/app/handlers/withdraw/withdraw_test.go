package withdraw

import (
	"errors"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	mocks2 "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage/mocks"
	ht "github.com/triumphpc/go-musthave-diploma-gophermart/pkg/http"
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
	}

	type server struct {
		path     string
		withAuth bool
	}

	tests := []struct {
		name    string
		want    want
		request request
		server  server
	}{
		{
			name: "Check withdraw",
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
				path:     "/api/user/balance/withdraw",
				withAuth: false,
			},
		},
		{
			name: "Check withdraw",
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "{\"order\": \"fasdf\",\"sum\": 6\n}",
			},
			want: want{
				code:        http.StatusUnprocessableEntity,
				contentType: "",
			},
			server: server{
				path:     "/api/user/balance/withdraw",
				withAuth: true,
			},
		},
		{
			name: "Check withdraw",
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "{\"order\": \"fd555\",\"sum\": 6\n}",
			},
			want: want{
				code:        http.StatusUnprocessableEntity,
				contentType: "",
			},
			server: server{
				path:     "/api/user/balance/withdraw",
				withAuth: true,
			},
		},
		{
			name: "Check withdraw",
			request: request{
				method: http.MethodPost,
				target: "/api/user/balance/withdraw",
				body:   "{\"order\": \"3\",\"sum\": 6\n}",
			},
			want: want{
				code:        http.StatusPaymentRequired,
				contentType: "",
			},
			server: server{
				path:     "/api/user/balance/withdraw",
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
						Login:  "test",
						UserID: 123,
					}, nil).
					On("OrderByCode", mock.Anything, mock.Anything).Return(models.Order{}, errors.New("test"))
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
