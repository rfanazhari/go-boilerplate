package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/SALT-Indonesia/salt-pkg/logmanager"

	"github.com/rfanazhari/go-boilerplate/infrastructure/config"
	"github.com/rfanazhari/go-boilerplate/infrastructure/runtime"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	logApp := logmanager.NewApplication(
		logmanager.WithAppName("taskdemo"),
		logmanager.WithService("taskdemo"),
		logmanager.WithMaskingConfig([]logmanager.MaskingConfig{
			{FieldPattern: "password", Type: logmanager.FullMask},
			{FieldPattern: "Authorization", Type: logmanager.FullMask},
			{FieldPattern: "Cookie", Type: logmanager.FullMask},
			{FieldPattern: "token", Type: logmanager.FullMask},
			{FieldPattern: "email", Type: logmanager.EmailMask},
		}),
	)
	cfg, err := config.FromEnv()
	if err != nil {
		txn := logApp.Start("startup", "config", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
		return err
	}
	rt, err := runtime.Start(ctx, cfg, logApp)
	if err != nil {
		txn := logApp.Start("startup", "runtime", logmanager.TxnTypeOther)
		txn.NoticeError(err)
		txn.End()
		return err
	}
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), rt.Shutdown)
	defer cancel()
	return rt.Stop(shutdownCtx)
}
