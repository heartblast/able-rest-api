package security

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const encryptedPrefix = "ENC("

// SecretProvider는 시크릿 해석 전략을 정의한다.
type SecretProvider interface {
	Name() string
	Resolve(ctx context.Context, value string) (string, error)
}

// SecurityConfigProvider는 필요한 보안 설정만 노출하는 최소 계약이다.
type SecurityConfigProvider interface {
	GetSecretProvider() string
	GetMasterKeyEnv() string
}

// PlaintextSecretProvider는 평문 시크릿만 허용한다.
type PlaintextSecretProvider struct{}

// Name은 프로바이더 이름을 반환한다.
func (p PlaintextSecretProvider) Name() string { return "plaintext" }

// Resolve는 평문을 그대로 반환한다.
func (p PlaintextSecretProvider) Resolve(_ context.Context, value string) (string, error) {
	if IsEncryptedValue(value) {
		return "", errors.New("암호화된 값이 감지되었지만 plaintext provider가 설정되어 있습니다")
	}
	return value, nil
}

// LocalEncryptionSecretProvider는 로컬 환경변수 마스터 키로 AES-256-GCM 복호화를 수행한다.
type LocalEncryptionSecretProvider struct {
	masterKey []byte
}

// Name은 프로바이더 이름을 반환한다.
func (p *LocalEncryptionSecretProvider) Name() string { return "local_encryption" }

// Resolve는 ENC(...) 형식이면 복호화하고 아니면 평문을 그대로 반환한다.
func (p *LocalEncryptionSecretProvider) Resolve(_ context.Context, value string) (string, error) {
	if !IsEncryptedValue(value) {
		return value, nil
	}
	return p.decrypt(value)
}

// Encrypt는 평문을 ENC(...) 형식으로 암호화한다.
func (p *LocalEncryptionSecretProvider) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return "", fmt.Errorf("AES cipher 생성 실패: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("AES-GCM 생성 실패: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce 생성 실패: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.StdEncoding.EncodeToString(ciphertext) + ")", nil
}

func (p *LocalEncryptionSecretProvider) decrypt(value string) (string, error) {
	raw := strings.TrimSuffix(strings.TrimPrefix(value, encryptedPrefix), ")")
	payload, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", fmt.Errorf("암호문 base64 디코딩 실패: %w", err)
	}

	block, err := aes.NewCipher(p.masterKey)
	if err != nil {
		return "", fmt.Errorf("AES cipher 생성 실패: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("AES-GCM 생성 실패: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", errors.New("암호문 길이가 올바르지 않습니다")
	}

	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("복호화 실패: %w", err)
	}

	return string(plaintext), nil
}

// PlaceholderSecretProvider는 향후 Vault/KMS/HSM 확장 포인트다.
type PlaceholderSecretProvider struct {
	name string
}

// Name은 프로바이더 이름을 반환한다.
func (p PlaceholderSecretProvider) Name() string { return p.name }

// Resolve는 미구현 확장 포인트임을 알린다.
func (p PlaceholderSecretProvider) Resolve(_ context.Context, _ string) (string, error) {
	return "", fmt.Errorf("%s provider는 아직 구현되지 않았습니다", p.name)
}

// NewSecretProvider는 설정 기반으로 시크릿 프로바이더를 생성한다.
func NewSecretProvider(cfg SecurityConfigProvider) (SecretProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.GetSecretProvider())) {
	case "", "plaintext":
		return PlaintextSecretProvider{}, nil
	case "local_encryption":
		return NewLocalEncryptionSecretProviderFromEnv(cfg.GetMasterKeyEnv())
	case "vault", "kms", "hsm":
		return PlaceholderSecretProvider{name: cfg.GetSecretProvider()}, nil
	default:
		return nil, fmt.Errorf("지원하지 않는 secret provider: %s", cfg.GetSecretProvider())
	}
}

// NewLocalEncryptionSecretProviderFromEnv는 환경변수에서 마스터 키를 읽어 provider를 생성한다.
func NewLocalEncryptionSecretProviderFromEnv(keyEnv string) (*LocalEncryptionSecretProvider, error) {
	keyEnv = strings.TrimSpace(keyEnv)
	if keyEnv == "" {
		return nil, errors.New("master key env 이름이 비어 있습니다")
	}

	raw := strings.TrimSpace(os.Getenv(keyEnv))
	if raw == "" {
		return nil, fmt.Errorf("마스터 키 환경변수 %s 가 비어 있습니다", keyEnv)
	}

	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("마스터 키 base64 디코딩 실패: %w", err)
	}
	if len(key) != 32 {
		return nil, errors.New("AES-256-GCM 마스터 키는 base64 디코딩 후 32바이트여야 합니다")
	}

	return &LocalEncryptionSecretProvider{masterKey: key}, nil
}

// IsEncryptedValue는 ENC(...) 형식을 판별한다.
func IsEncryptedValue(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix) && strings.HasSuffix(value, ")")
}
