package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"able-rest-api/internal/app/service"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/db/factory"
	schedulerinfra "able-rest-api/internal/infra/scheduler"
	"able-rest-api/internal/infra/security"
	mailmodule "able-rest-api/internal/modules/mail"
	"able-rest-api/internal/platform/logger"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/app.yaml", "설정 파일 경로")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load(ctx, configPath)
	if err != nil {
		panic(fmt.Errorf("설정 로드 실패: %w", err))
	}

	log := logger.New(cfg.App.Env)
	log.Info("scheduler process starting", "app", cfg.App.Name, "env", cfg.App.Env)

	secretProvider, err := security.NewSecretProvider(cfg.Security)
	if err != nil {
		log.Error("scheduler secret provider init failed", "error", err)
		os.Exit(1)
	}
	if err := cfg.ResolveSecrets(ctx, secretProvider); err != nil {
		log.Error("scheduler secret resolve failed", "error", err)
		os.Exit(1)
	}

	db, err := factory.NewSQLDB(ctx, cfg.DB)
	if err != nil {
		log.Error("scheduler db connect failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	lock, err := schedulerinfra.NewLock(cfg.Scheduler, cfg.DB.Vendor, db)
	if err != nil {
		log.Error("scheduler lock init failed", "error", err)
		os.Exit(1)
	}

	jobService := service.NewJobService(
		schedulerinfra.NewExecutionRepository(cfg.DB.Vendor, db),
		mailmodule.NewMailDispatchJob(),
	)

	runner := schedulerinfra.NewRunner(cfg.Scheduler, log, jobService, lock)

	runCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Run(runCtx)
	}()

	select {
	case <-runCtx.Done():
		log.Info("scheduler shutdown requested")
	case runErr := <-errCh:
		if runErr != nil {
			log.Error("scheduler stopped with error", "error", runErr)
			os.Exit(1)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Scheduler.ShutdownTimeout)
	defer shutdownCancel()

	select {
	case <-shutdownCtx.Done():
		log.Info("scheduler shutdown completed")
	case runErr := <-errCh:
		if runErr != nil {
			log.Error("scheduler shutdown error", "error", runErr)
			os.Exit(1)
		}
	case <-time.After(50 * time.Millisecond):
		log.Info("scheduler stopped")
	}
}
