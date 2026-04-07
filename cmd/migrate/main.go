package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"my-api/internal/infra/config"
	"my-api/internal/platform/logger"
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
	targetDir := filepath.Join("migrations", string(cfg.DB.Vendor))

	if _, err := os.Stat(targetDir); err != nil {
		log.Error("마이그레이션 디렉터리 확인 실패", "vendor", cfg.DB.Vendor, "error", err)
		os.Exit(1)
	}

	// TODO: 운영 환경에서는 golang-migrate 등 표준 도구를 붙여 실제 실행기로 확장한다.
	log.Info("마이그레이션 대상 디렉터리 확인 완료", "vendor", cfg.DB.Vendor, "dir", targetDir)
	fmt.Printf("migration directory: %s\n", targetDir)
}
