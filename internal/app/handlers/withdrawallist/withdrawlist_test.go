package withdrawallist

import (
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	mod "github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
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
		server  server
	}{
		{
			name: "Check withdraw list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/balance/withdrawals",
				body:   "",
				test:   0,
			},
			want: want{
				code:        http.StatusUnauthorized,
				contentType: "",
			},
			server: server{
				path:     "/api/user/balance/withdrawals",
				withAuth: false,
			},
		},
		{
			name: "Check withdraw list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/balance/withdrawals",
				body:   "",
				test:   0,
			},
			want: want{
				code:        http.StatusOK,
				contentType: "application/json; charset=utf-8",
			},
			server: server{
				path:     "/api/user/balance/withdrawals",
				withAuth: true,
			},
		},
		{
			name: "Check withdraw list",
			request: request{
				method: http.MethodGet,
				target: "/api/user/balance/withdrawals",
				body:   "",
				test:   2,
			},
			want: want{
				code:        http.StatusNoContent,
				contentType: "",
			},
			server: server{
				path:     "/api/user/balance/withdrawals",
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

				var wds []mod.Withdraw
				var wd mod.Withdraw
				wds = append(wds, wd)

				var nullWds []mod.Withdraw

				storage.
					On("UserByToken", mock.Anything, mock.Anything).Return(
					mod.User{
						Login:  "test",
						UserID: 123,
					}, nil).
					On("WithdrawsByUserID", mock.Anything, mock.MatchedBy(func(userID int) bool {
						return tt.request.test == 0
					})).Return(wds, nil).
					On("WithdrawsByUserID", mock.Anything, mock.MatchedBy(func(userID int) bool {
						return tt.request.test == 2
					})).Return(nullWds, nil)
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
