package config

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"able-rest-api/internal/infra/security"
)

type DBVendor string

const (
	DBVendorPostgres DBVendor = "postgres"
	DBVendorMySQL    DBVendor = "mysql"
	DBVendorOracle   DBVendor = "oracle"
	DBVendorHSQLDB   DBVendor = "hsqldb"
)

type Config struct {
	App       AppConfig       `yaml:"app"`
	Swagger   SwaggerConfig   `yaml:"swagger"`
	DB        DBConfig        `yaml:"db"`
	SMTP      SMTPConfig      `yaml:"smtp"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Security  SecurityConfig  `yaml:"security"`
}

type AppConfig struct {
	Name string `yaml:"name"`
	Port int    `yaml:"port"`
	Env  string `yaml:"env"`
	Host string `yaml:"host"`
}

type SwaggerConfig struct {
	Enabled bool `yaml:"enabled"`
}

type DBConfig struct {
	Vendor          DBVendor      `yaml:"vendor"`
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	Name            string        `yaml:"name"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type SMTPConfig struct {
	Enabled     bool          `yaml:"enabled"`
	Host        string        `yaml:"host"`
	Port        int           `yaml:"port"`
	Username    string        `yaml:"username"`
	Password    string        `yaml:"password"`
	FromAddress string        `yaml:"from_address"`
	FromName    string        `yaml:"from_name"`
	AuthType    string        `yaml:"auth_type"`
	StartTLS    bool          `yaml:"starttls"`
	TLS         bool          `yaml:"tls"`
	Timeout     time.Duration `yaml:"timeout"`
}

type SchedulerConfig struct {
	Enabled         bool          `yaml:"enabled"`
	PollInterval    time.Duration `yaml:"poll_interval"`
	Timezone        string        `yaml:"timezone"`
	RunnerID        string        `yaml:"runner_id"`
	LockProvider    string        `yaml:"lock_provider"`
	MaxParallelJobs int           `yaml:"max_parallel_jobs"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type SecurityConfig struct {
	SecretProvider      string `yaml:"secret_provider"`
	MasterKeyEnv        string `yaml:"master_key_env"`
	APIKeyEnv           string `yaml:"api_key_env"`
	MaxRequestBodyBytes int64  `yaml:"max_request_body_bytes"`
}

func (c SecurityConfig) GetSecretProvider() string {
	return c.SecretProvider
}

func (c SecurityConfig) GetMasterKeyEnv() string {
	return c.MasterKeyEnv
}

func ResolvePath(args []string) string {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return args[0]
	}
	return "configs/app.yaml"
}

func Load(_ context.Context, path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("설정 파일 읽기 실패: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("설정 YAML 파싱 실패: %w", err)
	}

	applyEnvOverride(cfg)
	if isNonLocalEnv(cfg.App.Env) && runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("설정 파일 확인 실패: %w", err)
		}
		if info.Mode().Perm()&0077 != 0 {
			return nil, errors.New("local 이외 환경에서 설정 파일 권한은 0600 이하여야 합니다")
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("설정 검증 실패: %w", err)
	}

	return cfg, nil
}

func (c *Config) ResolveSecrets(ctx context.Context, provider security.SecretProvider) error {
	dbPassword, err := provider.Resolve(ctx, c.DB.Password)
	if err != nil {
		return fmt.Errorf("db.password 복호화 실패: %w", err)
	}
	c.DB.Password = dbPassword

	smtpPassword, err := provider.Resolve(ctx, c.SMTP.Password)
	if err != nil {
		return fmt.Errorf("smtp.password 복호화 실패: %w", err)
	}
	c.SMTP.Password = smtpPassword

	return nil
}

func (c *Config) Validate() error {
	if c.App.Name == "" {
		return errors.New("app.name는 필수입니다")
	}
	if c.App.Port <= 0 {
		return errors.New("app.port는 1 이상이어야 합니다")
	}
	if c.App.Env == "" {
		return errors.New("app.env는 필수입니다")
	}
	if c.App.Host == "" {
		if isNonLocalEnv(c.App.Env) {
			c.App.Host = "0.0.0.0"
		} else {
			c.App.Host = "127.0.0.1"
		}
	}
	if c.DB.Host == "" {
		return errors.New("db.host는 필수입니다")
	}
	if c.DB.Name == "" {
		return errors.New("db.name는 필수입니다")
	}
	if c.DB.User == "" {
		return errors.New("db.user는 필수입니다")
	}
	if c.DB.Port <= 0 {
		return errors.New("db.port는 1 이상이어야 합니다")
	}

	switch c.DB.Vendor {
	case DBVendorPostgres, DBVendorMySQL, DBVendorOracle, DBVendorHSQLDB:
	default:
		return fmt.Errorf("지원하지 않는 db.vendor: %s", c.DB.Vendor)
	}

	if c.DB.MaxOpenConns <= 0 {
		c.DB.MaxOpenConns = 20
	}
	if c.DB.MaxIdleConns <= 0 {
		c.DB.MaxIdleConns = 10
	}
	if c.DB.ConnMaxLifetime <= 0 {
		c.DB.ConnMaxLifetime = 30 * time.Minute
	}
	if c.Security.SecretProvider == "" {
		c.Security.SecretProvider = "plaintext"
	}
	if c.Security.APIKeyEnv == "" {
		c.Security.APIKeyEnv = "APP_API_KEY"
	}
	if c.Security.MaxRequestBodyBytes == 0 {
		c.Security.MaxRequestBodyBytes = 30 << 20
	}
	if c.Security.MaxRequestBodyBytes < 1 || c.Security.MaxRequestBodyBytes > 100<<20 {
		return errors.New("security.max_request_body_bytes는 1 이상 100 MiB 이하여야 합니다")
	}

	if c.SMTP.Enabled {
		if c.SMTP.Host == "" {
			return errors.New("smtp.host는 필수입니다")
		}
		if c.SMTP.Port <= 0 {
			return errors.New("smtp.port는 1 이상이어야 합니다")
		}
		if c.SMTP.FromAddress == "" {
			return errors.New("smtp.from_address는 필수입니다")
		}
		address, err := mail.ParseAddress(c.SMTP.FromAddress)
		if err != nil || address.Address != c.SMTP.FromAddress || strings.ContainsAny(c.SMTP.FromName, "\r\n") {
			return errors.New("smtp 발신자 설정이 올바르지 않습니다")
		}
		if c.SMTP.Timeout <= 0 {
			c.SMTP.Timeout = 10 * time.Second
		}

		authType := strings.ToLower(strings.TrimSpace(c.SMTP.AuthType))
		if authType == "" {
			c.SMTP.AuthType = "plain"
			authType = "plain"
		}

		switch authType {
		case "plain":
			if c.SMTP.Username == "" {
				return errors.New("smtp.auth_type=plain 일 때 smtp.username은 필수입니다")
			}
			if strings.TrimSpace(c.SMTP.Password) == "" {
				return errors.New("smtp.auth_type=plain 일 때 smtp.password는 필수입니다")
			}
		case "none":
		default:
			return fmt.Errorf("지원하지 않는 smtp.auth_type: %s", c.SMTP.AuthType)
		}

		if c.SMTP.TLS && c.SMTP.StartTLS {
			return errors.New("smtp.tls와 smtp.starttls는 동시에 true일 수 없습니다")
		}
	}

	if c.Scheduler.Enabled {
		if c.Scheduler.PollInterval <= 0 {
			c.Scheduler.PollInterval = 10 * time.Second
		}
		if c.Scheduler.Timezone == "" {
			c.Scheduler.Timezone = "Asia/Seoul"
		}
		if c.Scheduler.RunnerID == "" {
			c.Scheduler.RunnerID = "scheduler-1"
		}
		if c.Scheduler.LockProvider == "" {
			if c.DB.Vendor == DBVendorPostgres {
				c.Scheduler.LockProvider = "postgres"
			} else {
				c.Scheduler.LockProvider = "none"
			}
		}
		if c.Scheduler.MaxParallelJobs <= 0 {
			c.Scheduler.MaxParallelJobs = 1
		}
		if c.Scheduler.ShutdownTimeout <= 0 {
			c.Scheduler.ShutdownTimeout = 30 * time.Second
		}
		switch strings.ToLower(strings.TrimSpace(c.Scheduler.LockProvider)) {
		case "postgres", "none":
		default:
			return fmt.Errorf("지원하지 않는 scheduler.lock_provider: %s", c.Scheduler.LockProvider)
		}
		if strings.EqualFold(c.Scheduler.LockProvider, "postgres") && c.DB.Vendor != DBVendorPostgres {
			return errors.New("scheduler.lock_provider=postgres는 postgres DB vendor에서만 사용할 수 있습니다")
		}
	}

	if isNonLocalEnv(c.App.Env) {
		if len(os.Getenv(c.Security.APIKeyEnv)) < 32 {
			return errors.New("local 이외 환경에서는 32자 이상의 API key 환경 변수가 필요합니다")
		}
		if c.SMTP.Enabled && !c.SMTP.TLS && !c.SMTP.StartTLS {
			return errors.New("local 이외 환경에서 SMTP TLS가 필요합니다")
		}
		if c.DB.Vendor == DBVendorPostgres && c.DB.SSLMode != "verify-full" {
			return errors.New("local 이외 환경에서 PostgreSQL sslmode=verify-full이 필요합니다")
		}
		if c.DB.Vendor == DBVendorMySQL && c.DB.SSLMode != "true" {
			return errors.New("local 이외 환경에서 MySQL tls=true가 필요합니다")
		}
		if strings.EqualFold(strings.TrimSpace(c.Security.SecretProvider), "plaintext") {
			return errors.New("local 이외 환경에서는 security.secret_provider=plaintext 를 사용할 수 없습니다")
		}
		if !security.IsEncryptedValue(strings.TrimSpace(c.DB.Password)) {
			return errors.New("local 이외 환경에서는 db.password를 ENC(...) 형식으로 설정해야 합니다")
		}
		if c.SMTP.Enabled && !security.IsEncryptedValue(strings.TrimSpace(c.SMTP.Password)) {
			return errors.New("local 이외 환경에서는 smtp.password를 ENC(...) 형식으로 설정해야 합니다")
		}
	}

	return nil
}

func isNonLocalEnv(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "", "local", "dev", "development", "test":
		return false
	default:
		return true
	}
}

func applyEnvOverride(cfg *Config) {
	overrideString(&cfg.App.Name, "APP_NAME")
	overrideString(&cfg.App.Host, "APP_HOST")
	overrideInt(&cfg.App.Port, "APP_PORT")
	overrideString(&cfg.App.Env, "APP_ENV")
	overrideBool(&cfg.Swagger.Enabled, "SWAGGER_ENABLED")

	overrideString((*string)(&cfg.DB.Vendor), "DB_VENDOR")
	overrideString(&cfg.DB.Host, "DB_HOST")
	overrideInt(&cfg.DB.Port, "DB_PORT")
	overrideString(&cfg.DB.Name, "DB_NAME")
	overrideString(&cfg.DB.User, "DB_USER")
	overrideString(&cfg.DB.Password, "DB_PASSWORD")
	overrideString(&cfg.DB.SSLMode, "DB_SSLMODE")
	overrideInt(&cfg.DB.MaxOpenConns, "DB_MAX_OPEN_CONNS")
	overrideInt(&cfg.DB.MaxIdleConns, "DB_MAX_IDLE_CONNS")
	overrideDuration(&cfg.DB.ConnMaxLifetime, "DB_CONN_MAX_LIFETIME")

	overrideBool(&cfg.SMTP.Enabled, "SMTP_ENABLED")
	overrideString(&cfg.SMTP.Host, "SMTP_HOST")
	overrideInt(&cfg.SMTP.Port, "SMTP_PORT")
	overrideString(&cfg.SMTP.Username, "SMTP_USERNAME")
	overrideString(&cfg.SMTP.Password, "SMTP_PASSWORD")
	overrideString(&cfg.SMTP.FromAddress, "SMTP_FROM_ADDRESS")
	overrideString(&cfg.SMTP.FromName, "SMTP_FROM_NAME")
	overrideString(&cfg.SMTP.AuthType, "SMTP_AUTH_TYPE")
	overrideBool(&cfg.SMTP.StartTLS, "SMTP_STARTTLS")
	overrideBool(&cfg.SMTP.TLS, "SMTP_TLS")
	overrideDuration(&cfg.SMTP.Timeout, "SMTP_TIMEOUT")

	overrideBool(&cfg.Scheduler.Enabled, "SCHEDULER_ENABLED")
	overrideDuration(&cfg.Scheduler.PollInterval, "SCHEDULER_POLL_INTERVAL")
	overrideString(&cfg.Scheduler.Timezone, "SCHEDULER_TIMEZONE")
	overrideString(&cfg.Scheduler.RunnerID, "SCHEDULER_RUNNER_ID")
	overrideString(&cfg.Scheduler.LockProvider, "SCHEDULER_LOCK_PROVIDER")
	overrideInt(&cfg.Scheduler.MaxParallelJobs, "SCHEDULER_MAX_PARALLEL_JOBS")
	overrideDuration(&cfg.Scheduler.ShutdownTimeout, "SCHEDULER_SHUTDOWN_TIMEOUT")

	overrideString(&cfg.Security.SecretProvider, "SECURITY_SECRET_PROVIDER")
	overrideString(&cfg.Security.MasterKeyEnv, "SECURITY_MASTER_KEY_ENV")
	overrideString(&cfg.Security.APIKeyEnv, "SECURITY_API_KEY_ENV")
	overrideInt64(&cfg.Security.MaxRequestBodyBytes, "SECURITY_MAX_REQUEST_BODY_BYTES")
}

func overrideInt64(target *int64, key string) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			*target = parsed
		}
	}
}

func overrideString(target *string, key string) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		*target = value
	}
}

func overrideInt(target *int, key string) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		*target = parsed
	}
}

func overrideBool(target *bool, key string) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return
	}
	if parsed, err := strconv.ParseBool(value); err == nil {
		*target = parsed
	}
}

func overrideDuration(target *time.Duration, key string) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		*target = parsed
	}
}
