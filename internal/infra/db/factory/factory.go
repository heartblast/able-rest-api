package factory

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/heartblast/able-rest-api/internal/infra/config"
)

// NewSQLDB는 설정 기반으로 *sql.DB를 생성한다.
func NewSQLDB(ctx context.Context, cfg config.DBConfig) (*sql.DB, error) {
	driverName, dsn, err := buildDSN(cfg)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open 실패: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("DB ping 실패: %w", sanitizeDBError(err))
	}

	return db, nil
}

func buildDSN(cfg config.DBConfig) (string, string, error) {
	switch cfg.Vendor {
	case config.DBVendorPostgres:
		query := url.Values{}
		if cfg.SSLMode != "" {
			query.Set("sslmode", cfg.SSLMode)
		}
		u := &url.URL{
			Scheme:   "postgres",
			User:     url.UserPassword(cfg.User, cfg.Password),
			Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
			Path:     cfg.Name,
			RawQuery: query.Encode(),
		}
		return "pgx", u.String(), nil
	case config.DBVendorMySQL:
		query := url.Values{}
		query.Set("parseTime", "true")
		if cfg.SSLMode != "" && cfg.SSLMode != "disable" {
			query.Set("tls", cfg.SSLMode)
		}
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, query.Encode())
		return "mysql", dsn, nil
	case config.DBVendorOracle:
		return "", "", errors.New("oracle 드라이버는 기본 빌드에 포함되지 않습니다. 별도 빌드 태그/드라이버 어댑터를 사용하세요")
	case config.DBVendorHSQLDB:
		return "", "", errors.New("hsqldb는 Go용 database/sql 드라이버 선정 후 어댑터를 연결하세요")
	default:
		return "", "", fmt.Errorf("지원하지 않는 db vendor: %s", cfg.Vendor)
	}
}

func sanitizeDBError(err error) error {
	if err == nil {
		return nil
	}
	return errors.New("database connection unavailable")
}
