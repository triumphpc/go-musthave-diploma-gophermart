package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/models"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {

	rtr := mux.NewRouter()
	// Registration users
	rtr.HandleFunc("/api/orders/{number}", func(w http.ResponseWriter, r *http.Request) {
		params := mux.Vars(r)
		number := params["number"]

		rand.Seed(time.Now().UnixNano())

		odr := models.LoyalOrder{
			Order:   number,
			Status:  "REGISTERED",
			Accrual: rand.Intn(100),
		}

		min := 0
		max := 5
		n := rand.Intn(max-min+1) + min

		// Prepare response
		w.Header().Add("Content-Type", "application/json; charset=utf-8")

		switch n {
		case 0:
			w.Header().Add("Retry-After", "5")
			w.WriteHeader(http.StatusTooManyRequests)
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
		case 2:
			w.WriteHeader(http.StatusOK)
			odr.Status = "REGISTERED"
		case 3:
			w.WriteHeader(http.StatusOK)
			odr.Status = "INVALID"
		case 4:
			w.WriteHeader(http.StatusOK)
			odr.Status = "PROCESSING"
		case 5:
			w.WriteHeader(http.StatusOK)
			odr.Status = "PROCESSED"
		}

		body, err := json.Marshal(odr)
		if err != nil {
			return
		}

		_, err = w.Write(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

	}).Methods(http.MethodGet)

	s := &http.Server{
		Addr:    ":8081",
		Handler: rtr,
	}
	log.Fatal(s.ListenAndServe())

}
