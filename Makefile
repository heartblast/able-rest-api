APP_NAME=able-rest-api
CONFIG?=configs/app.yaml

.PHONY: run build test openapi-check secretenc

run:
	go run ./cmd/server $(CONFIG)

build:
	go build ./...

test:
	go test ./...

openapi-check:
	go test ./docs ./internal/delivery/http/router

secretenc:
	go run ./cmd/secretenc --value "$(VALUE)" --key-env "$(KEY_ENV)"
