package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/heartblast/able-rest-api/internal/infra/config"
	"github.com/heartblast/able-rest-api/internal/infra/db/factory"
	schedulerinfra "github.com/heartblast/able-rest-api/internal/infra/scheduler"
	"github.com/heartblast/able-rest-api/internal/infra/security"
	"github.com/heartblast/able-rest-api/internal/modules/mail"
	"github.com/heartblast/able-rest-api/internal/modules/scheduler"
)

type scheduleRequest struct {
	ScheduledAt time.Time            `json:"scheduled_at"`
	Mail        mail.SendMailRequest `json:"mail"`
}

func main() {
	var configPath, inputPath string
	flag.StringVar(&configPath, "config", "configs/app.yaml", "설정 파일 경로")
	flag.StringVar(&inputPath, "input", "", "예약 요청 JSON 파일 경로")
	flag.Parse()
	if err := run(context.Background(), configPath, inputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath, inputPath string) error {
	if inputPath == "" {
		return errors.New("-input JSON 파일이 필요합니다")
	}
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("입력 파일 열기 실패: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 30<<20+1))
	decoder.DisallowUnknownFields()
	var request scheduleRequest
	if err := decoder.Decode(&request); err != nil {
		return fmt.Errorf("예약 요청 JSON 오류: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("예약 요청에는 JSON 객체 하나만 허용됩니다")
	}
	if request.ScheduledAt.IsZero() {
		return errors.New("scheduled_at은 필수입니다")
	}
	attachments := make([]mail.MailAttachment, 0, len(request.Mail.Attachments))
	for _, attachment := range request.Mail.Attachments {
		attachments = append(attachments, mail.MailAttachment{Filename: attachment.Filename, ContentType: attachment.ContentType, ContentBase64: attachment.ContentBase64})
	}
	message := mail.MailMessage{To: request.Mail.To, CC: request.Mail.CC, BCC: request.Mail.BCC, Subject: request.Mail.Subject, Body: request.Mail.Body, IsHTML: request.Mail.IsHTML, Attachments: attachments}
	cfg, err := config.Load(ctx, configPath)
	if err != nil {
		return err
	}
	provider, err := security.NewSecretProvider(cfg.Security)
	if err != nil {
		return err
	}
	if err := cfg.ResolveSecrets(ctx, provider); err != nil {
		return err
	}
	db, err := factory.NewSQLDB(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer db.Close()
	store, err := schedulerinfra.NewScheduledMailRepository(cfg.DB.Vendor, db)
	if err != nil {
		return err
	}
	id, err := scheduler.ScheduleMail(ctx, store, request.ScheduledAt, message)
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
}
