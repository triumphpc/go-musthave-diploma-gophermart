package main

import (
	"context"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/withdrawal"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/routes"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/logger"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/authchecker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/compressor"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/conveyor"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	// Broker
	bkr := broker.Init(lgr, stg, ent)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := bkr.Run(ctx); err != nil {
			lgr.Error("Worker pool returned error", zap.Error(err))
			cancel()
		}
	}()

	// Run withdraw handler
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := withdrawal.Run(ctx, lgr, stg); err != nil {
			lgr.Error("Worker pool returned error", zap.Error(err))
			cancel()
		}
	}()

	// Routes
	rtr := routes.Router(stg, lgr, bkr)
	http.Handle("/", rtr)
	// Server
	srv := &http.Server{
		Addr: ent.ServerAddress,
		Handler: conveyor.Conveyor(
			rtr,
			compressor.New(lgr).Gzip,
			authchecker.New(lgr, stg).CheckAuth,
		),
	}
	// Run server
	go func() {
		lgr.Info("app error exit", zap.Error(srv.ListenAndServe()))
	}()
	lgr.Info("The service is ready to listen and serve.", zap.String("addr", ent.ServerAddress))
	// Context with cancel func
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
	// Add context for Graceful shutdown
	select {
	case killSignal := <-interrupt:
		switch killSignal {
		case os.Interrupt:
			lgr.Info("Got SIGINT...")
		case syscall.SIGTERM:
			lgr.Info("Got SIGTERM...")
		}
	case <-ctx.Done():
	}

	lgr.Info("The service is shutting down...")
	cancel()
	stg.Close()
	// Server shutdown
	if err := srv.Shutdown(ctx); err != nil {
		lgr.Info("app error exit", zap.Error(err))
	}

	wg.Wait()

	lgr.Info("Done")
}
