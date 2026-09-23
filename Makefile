GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

.PHONY: up
# start local mysql redis and kafka
up:
	docker compose -f deploy/compose.yaml up -d

.PHONY: down
# stop local mysql redis and kafka
down:
	docker compose -f deploy/compose.yaml down

.PHONY: init
# init env
init:
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

.PHONY: api
# generate api proto
api:
	buf generate --template buf.gen.yaml

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: test
# run unit tests
test:
	go test ./...

.PHONY: test-integration
# run data layer tests against local mysql (TEST_MYSQL_DSN)
test-integration:
	go test -tags integration -count=1 -p 1 ./app/.../internal/data/...

.PHONY: seed
# seed catalog and stock through product and inventory gRPC (admin account is seed-admin)
seed:
	go run ./app/product/cmd/seed

.PHONY: seed-admin
# create the super admin for admin-web, requires ADMIN_INIT_PASSWORD with at least 8 characters
seed-admin:
	@test -n "$$ADMIN_INIT_PASSWORD" || { echo "ADMIN_INIT_PASSWORD is required (at least 8 characters)"; exit 1; }
	go run ./app/admin/cmd/seed -conf app/admin/configs

.PHONY: e2e
# run checkout flow through gateway (E2E_GATEWAY) with all services up and seeded
e2e:
	go test -tags e2e -count=1 -v ./app/gateway/e2e/...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api
	make generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
