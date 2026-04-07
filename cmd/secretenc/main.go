package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"

	"my-api/internal/infra/security"
)

func main() {
	var plaintext string
	var keyEnv string

	pflag.StringVar(&plaintext, "value", "", "암호화할 평문 값")
	pflag.StringVar(&keyEnv, "key-env", "APP_MASTER_KEY", "AES-256-GCM 마스터 키를 담은 환경변수명")
	pflag.Parse()

	if plaintext == "" {
		fmt.Fprintln(os.Stderr, "value 인자가 비어 있습니다")
		os.Exit(1)
	}

	provider, err := security.NewLocalEncryptionSecretProviderFromEnv(keyEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "시크릿 프로바이더 초기화 실패: %v\n", err)
		os.Exit(1)
	}

	encrypted, err := provider.Encrypt(plaintext)
	if err != nil {
		fmt.Fprintf(os.Stderr, "암호화 실패: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(encrypted)
}
