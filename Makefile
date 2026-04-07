APP_NAME=able-rest-api
CONFIG?=configs/app.yaml

.PHONY: run build test swag secretenc

run:
	go run ./cmd/server $(CONFIG)

build:
	go build ./...

test:
	go test ./...

swag:
	swag init -g cmd/server/main.go -o docs

secretenc:
	go run ./cmd/secretenc --value "$(VALUE)" --key-env "$(KEY_ENV)"
