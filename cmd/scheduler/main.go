package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"able-rest-api/internal/infra/config"
	"able-rest-api/internal/infra/db/factory"
	mailinfra "able-rest-api/internal/infra/mail"
	schedulerinfra "able-rest-api/internal/infra/scheduler"
	"able-rest-api/internal/infra/security"
	mailmodule "able-rest-api/internal/modules/mail"
	"able-rest-api/internal/modules/scheduler"
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

	var mailSender mailmodule.MailSender
	if cfg.SMTP.Enabled {
		mailSender = mailinfra.NewSMTPSender(cfg.SMTP)
	}
	mailService := mailmodule.NewMailService(cfg.SMTP.Enabled, mailSender)
	mailCfg := cfg.Scheduler.MailDispatch
	attachments := make([]mailmodule.MailAttachment, 0, len(mailCfg.Attachments))
	for _, attachment := range mailCfg.Attachments {
		attachments = append(attachments, mailmodule.MailAttachment{
			Filename: attachment.Filename, ContentType: attachment.ContentType, ContentBase64: attachment.ContentBase64,
		})
	}
	mailCommand := mailmodule.NewScheduledDispatch(mailService, mailmodule.MailMessage{
		To: mailCfg.To, CC: mailCfg.CC, BCC: mailCfg.BCC,
		Subject: mailCfg.Subject, Body: mailCfg.Body, IsHTML: mailCfg.IsHTML, Attachments: attachments,
	})
	jobs := []scheduler.JobRunner{scheduler.NewMailDispatchJob(mailCommand, mailCfg.Enabled, mailCfg.Interval)}
	if cfg.Scheduler.ScheduledMail.Enabled {
		store, storeErr := schedulerinfra.NewScheduledMailRepository(cfg.DB.Vendor, db)
		if storeErr != nil {
			log.Error("scheduled mail repository init failed", "error", storeErr)
			os.Exit(1)
		}
		mailSchedule := cfg.Scheduler.ScheduledMail
		jobs = append(jobs, scheduler.NewScheduledMailJob(store, mailmodule.NewMessageDispatch(mailService), true,
			mailSchedule.Interval, mailSchedule.LeaseDuration, mailSchedule.MaxAttempts, mailSchedule.RetryDelay, log))
	}
	jobService := scheduler.NewJobService(
		schedulerinfra.NewExecutionRepository(cfg.DB.Vendor, db),
		jobs...,
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
