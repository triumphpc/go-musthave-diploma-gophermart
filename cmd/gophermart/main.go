package main

import (
	"context"
	"errors"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/env"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/broker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/pg"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/storage"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/pkg/withdrawal"
	"github.com/triumphpc/go-musthave-diploma-gophermart/internal/app/routes"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/checker"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/logger"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/compressor"
	"github.com/triumphpc/go-musthave-diploma-gophermart/pkg/middlewares/conveyor"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
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
	// Init publisher
	pub := broker.NewPublisher(lgr, ent, stg)

	wg := sync.WaitGroup{}

	// Init subscribers
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := initSubscribers(ctx, pub, lgr, ent, stg); err != nil {
			if !errors.Is(err, context.Canceled) {
				lgr.Error("Broker returned error", zap.Error(err))
				cancel()
			}
		}
	}()

	ckr := checker.New(lgr, ent, stg)
	// Init broker listeners
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := ckr.RunListeners(ctx, pub); err != nil {
			if !errors.Is(err, context.Canceled) {
				lgr.Error("Broker listeners error", zap.Error(err))
				cancel()
			}
		}
	}()

	// Init repeater
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err = ckr.Repeater(ctx, pub); err != nil {
			if !errors.Is(err, context.Canceled) {
				lgr.Error("Withdraw pool returned error", zap.Error(err))
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

	// Init server
	if err := serve(ctx, cancel, lgr, stg, ent, pub, ckr); err != nil {
		lgr.Error("failed to serve:", zap.Error(err))
	}

	wg.Wait()

	lgr.Info("Done")
}

// serve implementation
func serve(
	ctx context.Context,
	cancel context.CancelFunc,
	lgr *zap.Logger,
	stg storage.Storage,
	ent *env.Env,
	pub broker.Publisher,
	ckr checker.Controller,
) (err error) {
	// Routes
	rtr := routes.Router(lgr, stg, pub, ckr)
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

	// System signals
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case killSignal := <-interrupt:
		switch killSignal {
		case os.Interrupt:
			lgr.Info("Got SIGINT...")
		case syscall.SIGTERM:
			lgr.Info("Got SIGTERM...")
		}
		cancel()

	case <-ctx.Done():
		lgr.Info("out...")
	}

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

// initSubscribers create subscribers on publisher
func initSubscribers(ctx context.Context, pub broker.Publisher, lgr *zap.Logger, ent *env.Env, stg storage.Storage) error {
	group, currentCtx := errgroup.WithContext(ctx)

	for i := 0; i < runtime.NumCPU(); i++ {
		workID := i
		f := func() error {
			// create new subscriber
			sub := broker.NewSubscriber(lgr, ent, stg)
			// subscribe in pub channel
			if err := sub.Subscribe(currentCtx, pub.Channel(), workID); err != nil {
				return err
			}

			return currentCtx.Err()
		}
		group.Go(f)
	}

	return group.Wait()
}
