# supermarket

开一家超市：自研电商微服务。复刻 [mall](https://github.com/macrozheng/mall) 的功能需求边界，架构自研并全面超出蓝本。

个人全栈学习项目，2026-09 启动，共 18 周。终点形态：

- 全量部署 Kubernetes
- 秒杀零超卖（Redis Lua 预扣）
- 压测报告与性能调优
- AI 客服（RAG）

## 架构

> 架构图待补（Excalidraw），当前为规划形态。

调用关系：客户端（web H5 / admin-web）→ gateway → 各业务服务；服务间同步调用走 gRPC，异步事件走 Kafka；MySQL 按服务分库，Redis 做缓存与预扣库存。

## 服务

| 服务 | 目录 | 职责 | 端口（HTTP / gRPC） | 状态 |
|---|---|---|---|---|
| gateway | `app/gateway` | API 网关：路由、鉴权、限流 | 8080 / — | 转发注册登录，查资料验签 |
| user | `app/user` | 用户：注册登录、资料、地址 | 8000 / 9000 | 注册、登录、查资料 |
| product | `app/product` | 商品：SPU/SKU、分类、检索 | 8001 / 9001 | 目录已建 |
| inventory | `app/inventory` | 库存：预扣、防超卖 | 8002 / 9002 | 目录已建 |
| order | `app/order` | 订单：状态机、延迟取消 | 8003 / 9003 | 目录已建 |
| payment | `app/payment` | 支付（mock）：幂等、对账 | 8004 / 9004 | 目录已建 |
| notification | — | 通知：Kafka 消费、多渠道下发 | — | 未开始 |
| web | `frontend/web` | 用户端 H5：商品浏览、购物车、下单 | 5174 | 注册、登录、我的 |
| admin-web | `frontend/admin-web` | 管理台（React + TS） | 5173 | 空壳，待运营账号 |

## 技术栈

- 后端：Go · [kratos](https://github.com/go-kratos/kratos) · MySQL · Redis · Kafka
- 前端：TypeScript · React · Vite（admin-web 与 web 同栈）
- 基建：Docker · Kubernetes · GitHub Actions

## 路线

| 阶段 | 内容 | 周期 |
|---|---|---|
| 1 · 项目开发 | 仓库/CI/compose，下单全链路本地跑通（web 简版：浏览+下单，随 admin-web） | 约 8 周（当前） |
| 2 · 运维与上线 | CI/CD、K8s 集群、线上排查 | 约 4 周 |
| 3 · 数据库深入 | EXPLAIN 实战、索引与慢查询优化 | 约 2 周 |
| 4 · 高可用与性能 | 缓存三大问题、限流熔断、压测 | 约 2 周 |
| 5 · AI 与收尾 | AI 客服（RAG）、架构文档 | 约 2 周 |

## 本地开发

依赖：Go 1.26+、Docker、Node 26+。

```bash
make up
```

`make up` 起 MySQL、Redis 和单节点 Kafka（KRaft）。第一次会建好 `user`、`product`、`inventory`、`order`、`payment`、`notification`、`admin` 七个库。MySQL 根账号 `root` / `root`。`make down` 停掉，MySQL 数据留在卷里。

宿主端口可以用环境变量改，默认值如下：

| 变量 | 默认 | 用途 |
|---|---|---|
| `MYSQL_PORT` | 3306 | MySQL |
| `REDIS_PORT` | 6379 | Redis |
| `KAFKA_PORT` | 9092 | Kafka（对外地址 `127.0.0.1:$KAFKA_PORT`） |

改了端口，服务那边也要跟着改：库连接串用 `USER_DATABASE_SOURCE` 这类变量（见各服务 `configs/config.yaml`），Redis 用 `REDIS_ADDR`，Kafka 用 `KAFKA_BROKERS`。

本机已经有 MySQL、Redis 容器时，可以只起 Kafka：`docker compose -f deploy/compose.yaml up -d kafka`，再用 `docker exec -i <mysql 容器> mysql -uroot -proot < deploy/mysql/init.sql` 建库。

服务按这个顺序启动，都在各自的 `cmd` 目录执行 `go run . -conf ../../configs`：user、admin、product、inventory、order、payment、notification、gateway。gateway 最后起，因为它要连上游。

配置文件里的 `${VAR:默认值}` 直接读同名环境变量（不带前缀），没设就用默认值。

密钥：买家令牌用 `AUTH_JWT_SECRET`，运营令牌用 `ADMIN_JWT_SECRET`，所有服务要用同一组值。两把都不能为空，也不能相同，否则服务启动即退出；本地缺省值以 `change-me-` 开头，只能本机用，启动时会打警告，部署前必须换掉。例如：

```bash
export AUTH_JWT_SECRET=$(openssl rand -hex 32) ADMIN_JWT_SECRET=$(openssl rand -hex 32)
```

各服务自己的 HTTP 端口不信任请求里带的 `x-md-global-*` 身份头，买家接口自己验买家令牌，`/v1/admin/` 接口自己验运营令牌（`aud=admin`）和角色；gRPC 端口仍按元数据取身份，只给内网调用。

初始数据：

```bash
make seed                                   # 经 product、inventory 的 gRPC 建品牌、分类、属性、商品并上架，每个启用 SKU 入库 100，按名称幂等
ADMIN_INIT_PASSWORD='至少8位的口令' make seed-admin   # 建超级管理员 admin，登录 admin-web 用
```

`make seed` 不建运营账号。超级管理员由 `make seed-admin`（即 `go run ./app/admin/cmd/seed -conf app/admin/configs`）建，必须设 `ADMIN_INIT_PASSWORD`，至少 8 位，没设或太短直接报错退出，没有内置默认口令；账号已存在时跳过，不会改口令。

测试：

```bash
make test              # 单元测试，不连数据库
make test-integration  # 数据层集成测试，连 TEST_MYSQL_DSN（缺省 root:root@tcp(127.0.0.1:3306)/），自建 *_it 库
make e2e               # 全部服务启动并跑过种子后，经 gateway（E2E_GATEWAY，缺省 http://127.0.0.1:8080）走注册到收通知的全链路
```

前端：`frontend/web` 是 `npm run dev`（5174），`frontend/admin-web` 是 `npm run dev`（5173）。请求经 Vite 转到 gateway 的 8080。
