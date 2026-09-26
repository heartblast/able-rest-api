package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func secureConfig() *Config {
	return &Config{
		App:      AppConfig{Name: "api", Port: 8080, Env: "production"},
		DB:       DBConfig{Vendor: DBVendorPostgres, Host: "db.example.com", Port: 5432, Name: "app", User: "app", Password: "ENC(ciphertext)", SSLMode: "verify-full"},
		Security: SecurityConfig{SecretProvider: "local_encryption", APIKeyEnv: "CONFIG_TEST_API_KEY"},
	}
}

func TestProductionSecurityDefaultsAndFailures(t *testing.T) {
	t.Setenv("CONFIG_TEST_API_KEY", strings.Repeat("k", 40))
	cfg := secureConfig()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if cfg.Security.MaxRequestBodyBytes != 30<<20 {
		t.Fatalf("body limit = %d", cfg.Security.MaxRequestBodyBytes)
	}

	t.Run("missing API key", func(t *testing.T) {
		t.Setenv("CONFIG_TEST_API_KEY", "")
		if err := secureConfig().Validate(); err == nil {
			t.Fatal("missing key accepted")
		}
	})
	t.Run("insecure DB", func(t *testing.T) {
		cfg := secureConfig()
		cfg.DB.SSLMode = "disable"
		if err := cfg.Validate(); err == nil {
			t.Fatal("insecure DB accepted")
		}
	})
	t.Run("insecure SMTP", func(t *testing.T) {
		cfg := secureConfig()
		cfg.SMTP.Enabled = true
		cfg.SMTP.Host = "smtp.example.com"
		cfg.SMTP.Port = 25
		cfg.SMTP.FromAddress = "a@example.com"
		cfg.SMTP.AuthType = "none"
		if err := cfg.Validate(); err == nil {
			t.Fatal("plaintext SMTP accepted")
		}
	})
	t.Run("header injection in sender", func(t *testing.T) {
		cfg := secureConfig()
		cfg.SMTP.Enabled = true
		cfg.SMTP.Host = "smtp.example.com"
		cfg.SMTP.Port = 587
		cfg.SMTP.FromAddress = "a@example.com\r\nBcc: b@example.com"
		cfg.SMTP.AuthType = "none"
		cfg.SMTP.StartTLS = true
		if err := cfg.Validate(); err == nil {
			t.Fatal("injected sender accepted")
		}
	})
}

func TestProductionConfigFilePermissions(t *testing.T) {
	if os.PathSeparator == '\\' {
		t.Skip("Windows permissions differ")
	}
	t.Setenv("CONFIG_TEST_API_KEY", strings.Repeat("k", 40))
	path := filepath.Join(t.TempDir(), "app.yaml")
	data := []byte("app: {name: api, port: 8080, env: production}\ndb: {vendor: postgres, host: db.example.com, port: 5432, name: app, user: app, password: 'ENC(ciphertext)', sslmode: verify-full}\nsecurity: {secret_provider: local_encryption, api_key_env: CONFIG_TEST_API_KEY}\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(context.Background(), path); err == nil {
		t.Fatal("world-readable config accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(context.Background(), path); err != nil {
		t.Fatalf("private config rejected: %v", err)
	}
}
