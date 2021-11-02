package main

import (
	"context"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/goproducer"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/rabbit"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/withdrawal"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/routes"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/logger"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/compressor"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/conveyor"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Entrypoint project
func main() {
	// Init:
	// Environments
	ent := env.New()
	// Logger
	lgr, err := logger.New()
	if err != nil {
		log.Fatal(err)
	}
	// Ctx
	ctx, cancel := context.WithCancel(context.Background())

	// Pg
	stg, err := pg.New(ctx, lgr, ent)
	if err != nil {
		lgr.Fatal("Pg init error", zap.Error(err))
	}

	// Publisher
	pub := initBrokerPublisher(lgr, ent, stg)
	// Subscriber
	sub := broker.NewConsumer(lgr, ent, stg)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := pub.Run(ctx, sub); err != nil {
			if !errors.Is(err, context.Canceled) {
				lgr.Error("Broker returned error", zap.Error(err))
				cancel()
			}
		}
	}()

	// Withdraw handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := withdrawal.Run(ctx, lgr, stg); err != nil {
			if !errors.Is(err, context.Canceled) {
				lgr.Error("Withdraw pool returned error", zap.Error(err))
				cancel()
			}
		}
	}()

	// System signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		killSignal := <-interrupt
		switch killSignal {
		case os.Interrupt:
			lgr.Info("Got SIGINT...")
		case syscall.SIGTERM:
			lgr.Info("Got SIGTERM...")
		}
		cancel()
	}()

	if err := serve(ctx, stg, lgr, pub, ent); err != nil {
		lgr.Error("failed to serve:", zap.Error(err))
	}

	wg.Wait()

	lgr.Info("Done")
}

// serve implementation
func serve(ctx context.Context, stg storage.Storage, lgr *zap.Logger, bkr broker.Publisher, ent *env.Env) (err error) {
	// Routes
	rtr := routes.Router(stg, lgr, bkr)
	http.Handle("/", rtr)
	// Server
	srv := &http.Server{
		Addr: ent.ServerAddress,
		Handler: conveyor.Conveyor(
			rtr,
			compressor.New(lgr).Gzip,
		),
	}
	// Run server
	go func() {
		if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			lgr.Fatal("app error exit", zap.Error(srv.ListenAndServe()))
		}
	}()

	lgr.Info("The service is ready to listen and serve.", zap.String("addr", ent.ServerAddress))

	<-ctx.Done()

	// Close storage connect
	stg.Close()
	lgr.Info("Storage connection stopped")
	lgr.Info("Server stopped")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = srv.Shutdown(ctxShutDown); err != nil {
		lgr.Fatal("server shutdown failed", zap.Error(err))
	}

	lgr.Info("Server exited properly")

	return
}

// initBrokerPublisher broker publisher by config
func initBrokerPublisher(lgr *zap.Logger, ent *env.Env, stg storage.Storage) broker.Publisher {
	if ent.BrokerType == env.BrokerTypeRabbitMQ {
		return rabbit.NewProducer(lgr, ent, stg)
	}

	return goproducer.NewProducer(lgr, ent, stg)
}
