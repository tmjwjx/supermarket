GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
COMPOSE := docker compose -f deploy/compose.yaml
APP_SERVICES := gateway user product inventory order payment notification admin web admin-web

.PHONY: up
# 只起 mysql redis kafka kafka-ui 编排文件是 deploy/compose.yaml 默认查找名是 compose.yaml compose.yml docker-compose.yaml docker-compose.yml 文件在 deploy 下所以用 -f
up:
	$(COMPOSE) up -d mysql redis kafka kafka-ui

.PHONY: down
# 停掉 deploy/compose.yaml 拉起的容器 MySQL 和 Redis 的卷保留
down:
	$(COMPOSE) down

.PHONY: db-init
# 在已启动的 mysql 里执行 deploy/mysql/init.sql 建本项目的库
db-init:
	docker exec -i mysql mysql -uroot -proot < deploy/mysql/init.sql

.PHONY: stack
# 先起基础设施 等 mysql 健康后执行 init.sql 再启动全部业务服务
stack:
	$(COMPOSE) up -d --wait mysql redis kafka kafka-ui
	$(MAKE) db-init
	$(COMPOSE) up -d $(APP_SERVICES)

.PHONY: images
# 按 Dockerfile 构建业务镜像 编排只引用镜像名 不含 mysql redis kafka kafka-ui
images:
	docker build -t ghcr.io/tmjwjx/supermarket-gateway -f app/gateway/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-user -f app/user/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-product -f app/product/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-inventory -f app/inventory/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-order -f app/order/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-payment -f app/payment/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-notification -f app/notification/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-admin -f app/admin/Dockerfile .
	docker build -t ghcr.io/tmjwjx/supermarket-web -f frontend/web/Dockerfile frontend/web
	docker build -t ghcr.io/tmjwjx/supermarket-admin-web -f frontend/admin-web/Dockerfile frontend/admin-web

.PHONY: up-service
# 只用已有镜像更新一个服务 不重启其余容器 用法 make up-service SERVICE=user
up-service:
	@test -n "$(SERVICE)" || { echo "usage: make up-service SERVICE=user"; exit 1; }
	$(COMPOSE) up -d --no-deps $(SERVICE)

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
# 数据层集成测试 连本机 root:root@tcp(127.0.0.1:3306)/
test-integration:
	go test -tags integration -count=1 -p 1 ./app/.../internal/data/...

.PHONY: seed
# seed catalog and stock through product and inventory gRPC (admin account is seed-admin)
seed:
	go run ./app/product/cmd/seed

.PHONY: seed-admin
# 建超级管理员 口令取 admin 配置 seed.init_password 为空或短于 8 位时由程序退出
seed-admin:
	go run ./app/admin/cmd/seed -conf app/admin/configs/dev.yaml

.PHONY: e2e
# 经 http://127.0.0.1:8080 走下单全链路 需要服务已启动并完成种子
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
