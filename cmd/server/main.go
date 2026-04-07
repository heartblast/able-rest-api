package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "able-rest-api/docs"
	"able-rest-api/internal/app/service"
	"able-rest-api/internal/delivery/http/router"
	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/db/factory"
	mailinfra "able-rest-api/internal/infra/mail"
	"able-rest-api/internal/infra/persistence"
	"able-rest-api/internal/infra/security"
	"able-rest-api/internal/platform/logger"
)

// @title My API
// @version 1.0
// @description 멀티 DB와 암호화된 설정을 지원하는 REST API 샘플입니다.
// @BasePath /
func main() {
	ctx := context.Background()

	cfgPath := config.ResolvePath(os.Args[1:])
	cfg, err := config.Load(ctx, cfgPath)
	if err != nil {
		panic(fmt.Errorf("설정 로드 실패: %w", err))
	}

	log := logger.New(cfg.App.Env)
	log.Info("애플리케이션 시작", "app", cfg.App.Name, "env", cfg.App.Env, "vendor", cfg.DB.Vendor)

	secretProvider, err := security.NewSecretProvider(cfg.Security)
	if err != nil {
		log.Error("시크릿 프로바이더 초기화 실패", "error", err)
		os.Exit(1)
	}

	if err := cfg.ResolveSecrets(ctx, secretProvider); err != nil {
		log.Error("설정 시크릿 복호화 실패", "error", err)
		os.Exit(1)
	}

	db, err := factory.NewSQLDB(ctx, cfg.DB)
	if err != nil {
		log.Error("DB 연결 실패", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	repos, err := persistence.NewRepositories(cfg.DB.Vendor, db)
	if err != nil {
		log.Error("리포지토리 초기화 실패", "error", err)
		os.Exit(1)
	}

	var mailSender *mailinfra.SMTPSender
	if cfg.SMTP.Enabled {
		mailSender = mailinfra.NewSMTPSender(cfg.SMTP)
	}
	mailService := service.NewMailService(cfg.SMTP.Enabled, mailSender)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.App.Port),
		Handler:           router.New(cfg, log, db, repos, mailService),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("HTTP 서버 리슨", "addr", srv.Addr)
		if serveErr := srv.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stopCh:
		log.Info("종료 시그널 수신", "signal", sig.String())
	case serveErr := <-errCh:
		log.Error("HTTP 서버 비정상 종료", "error", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("HTTP 서버 graceful shutdown 실패", "error", err)
		os.Exit(1)
	}

	log.Info("애플리케이션 종료 완료")
}
