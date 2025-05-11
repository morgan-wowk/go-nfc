package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ebfe/scard"
	"github.com/morgan-wowk/nfc/logger"
	"github.com/morgan-wowk/nfc/targets/httptarget"
)

func main() {
	ctx, err := logger.RegisterLoggerInContext(context.Background())
	cctx, cancelCtx := context.WithCancel(ctx)

	scardCtx, err := scard.EstablishContext()
	if err != nil {
		logger.FromContext(ctx).Errorf("Error establishing connection to system: %s", err.Error())
		return
	}
	logger.FromContext(ctx).Info("Established scard context")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.FromContext(ctx).Warnf("Received kill signal: %v", sig)
		logger.FromContext(ctx).Info("Shutting down gracefully within 5 seconds...")
		cancelCtx()
	}()

	defer cleanup(ctx, scardCtx, err)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func(pctx context.Context) {
		httpTarget := httptarget.NewHttpTarget("https://google.ca", nil)
		cardSvc := newCardService(scardCtx, httpTarget)

		c := newController(scardCtx, cardSvc)
		c.init(pctx)
		wg.Done()
	}(cctx)

	wg.Wait()
}

func cleanup(ctx context.Context, scardCtx *scard.Context, err error) {
	if scardCtx != nil {
		logger.FromContext(ctx).Info("Releasing scard context...")
		if err := scardCtx.Release(); err != nil {
			logger.FromContext(ctx).Errorf("Error closing card context: %s", err.Error())
		}
		logger.FromContext(ctx).Info("SCard context released")
	}
	if err := logger.ReleaseLogger(ctx); err != nil {
		logger.FromContext(ctx).Warnf("Error releasing logger: %s", err.Error())
	}
	if r := recover(); r != nil {
		if err != nil {
			err = fmt.Errorf("panic occurred after previous error: %+v, error: %s", r, err.Error())
		} else {
			err = fmt.Errorf("panic: %+v", r)
		}
	}
	if err != nil {
		fmt.Printf("Program exiting with error: %s\n", err.Error())
		os.Exit(1)
	}
}
