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

| 服务 | 目录 | 职责 | 端口（HTTP / gRPC） |
|---|---|---|---|
| gateway | `app/gateway` | 路由、鉴权 | 8080 / — |
| user | `app/user` | 注册登录、资料、地址 | 8000 / 9000 |
| product | `app/product` | 商品目录 | 8001 / 9001 |
| inventory | `app/inventory` | 库存预占 | 8002 / 9002 |
| order | `app/order` | 订单、购物车 | 8003 / 9003 |
| payment | `app/payment` | 支付（mock）、对账 | 8004 / 9004 |
| notification | `app/notification` | 站内通知 | 8005 / 9005 |
| admin | `app/admin` | 运营账号 | 8006 / 9006 |
| web | `frontend/web` | 用户端 | 5174 |
| admin-web | `frontend/admin-web` | 管理台 | 5173 |

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

编排文件是 `deploy/compose.yaml`。Compose 默认只在当前目录找 `compose.yaml`、`compose.yml`、`docker-compose.yaml`、`docker-compose.yml`。文件放在 `deploy/` 下，所以命令都带 `-f deploy/compose.yaml`。

```bash
make up        # 只起 MySQL、Redis、Kafka 和 Kafka 网页
make stack     # 先起基础设施，等 MySQL 健康并执行 init.sql，再起全部业务容器
make down      # 停掉这份编排拉起的容器，MySQL 和 Redis 的数据留在卷里
make db-init   # 在已启动的 mysql 里执行 deploy/mysql/init.sql，建本项目的七个库
make images    # 构建 8 个 Go 服务和 web、admin-web，不含 mysql、redis、kafka、kafka-ui
make up-service SERVICE=user   # 只用已有镜像更新 user，不重启其余容器
```

`make up` 起一套本机共用的 MySQL、Redis、单节点 Kafka（KRaft），外加 Kafka 的网页。四个依赖都只监听 `127.0.0.1`，不建任何业务库，别的项目也可以连。MySQL 根账号 `root` / `root`，Redis 无密码。端口写死为 `127.0.0.1:3306`、`6379`、`9092`。

`make stack` 先做和 `make up` 相同的事，等 MySQL 健康后执行和 `make db-init` 相同的 `deploy/mysql/init.sql`，再 `up -d` 全部业务容器。库必须先建好，业务容器第一次启动才能连上。`make up` 仍然只起基础设施，不带业务服务。

`make images` 按各服务 Dockerfile 打出与 CI 相同的名字，例如 `ghcr.io/tmjwjx/supermarket-user`。镜像里只有程序，没有配置。`make up-service` 对应 `docker compose -f deploy/compose.yaml up -d --no-deps <服务名>`，服务名是 gateway、user、product、inventory、order、payment、notification、admin、web、admin-web。

本地和线上是同一份 `deploy/compose.yaml`。本地 `deploy/config/*.yaml` 是开发内容，挂到容器内 `/configs/app.yaml`。服务器上把 `deploy/compose.yaml` 和一份 `config/` 放在一起，再执行 `docker compose up -d`。那份 `config/` 用线上内容，空模板在 `deploy/config-prod/`。镜像来自 GHCR，服务器不需要源码。

Kafka 网页：http://127.0.0.1:8082 ，打开就是名为 local 的集群。

浏览器访问：用户端 5174、管理台 5173、gateway 8080，Kafka 网页 8082。业务服务的 HTTP 从 8000 起、gRPC 从 9000 起，同一个服务两个端口差 1000，按 user、product、inventory、order、payment、notification、admin 的顺序各加 1。MySQL 3306、Redis 6379、Kafka 9092 用各自的默认端口。Kafka 控制器 9093 和给界面用的 29092 只在容器网络里，不映射到宿主机。

每个服务两份配置：`configs/dev.yaml` 和 `configs/prod.yaml`。开发环境买家密钥是 `tmjwjx-user-jwt-secret`，运营密钥是 `tmjwjx-admin-jwt-secret`，所有服务要写成同一组。两把都不能为空，也不能相同，否则服务启动即退出。以 `change-me-` 开头时启动会打警告。`prod.yaml` 的数据库地址、Redis 地址、下游地址、Kafka 地址和密钥先空着，等服务器接入后再填。product、inventory、order、payment、notification 的 `dev.yaml` 里 `kafka.brokers` 是 `kafka:29092`，给容器网络用。宿主机上的程序若要连已映射的 Kafka，用 `127.0.0.1:9092`。

不要把 `-conf` 指到整个 `configs` 目录，否则 `dev.yaml` 和 `prod.yaml` 会合并。默认是 `../../configs/dev.yaml`。这份文件里的主机名 `mysql`、`redis` 只在 Compose 网络里能解析，宿主机上直接 `go run` 连不上数据库。

各服务自己的 HTTP 端口不信任请求里带的 `x-md-global-*` 身份头，买家接口自己验买家令牌，`/v1/admin/` 接口自己验运营令牌（`aud=admin`）和角色；gRPC 端口仍按元数据取身份，只给内网调用。

初始数据：

```bash
make seed        # 经 product、inventory 的 gRPC 建品牌、分类、属性、商品并上架，每个启用 SKU 入库 100，按名称幂等
make seed-admin  # 建超级管理员 admin，登录 admin-web 用
```

`make seed` 不建运营账号。它在宿主机上跑，默认连已映射的 `127.0.0.1:9001` 和 `127.0.0.1:9002`，需要改地址时用 `-product` 和 `-inventory`。超级管理员由 `make seed-admin`（即 `go run ./app/admin/cmd/seed -conf app/admin/configs/dev.yaml`）建，口令读配置字段 `seed.init_password`。dev.yaml 和 prod.yaml 都还没写这个字段，读出来是空，没填或短于 8 位直接报错退出，没有内置默认口令；账号已存在时跳过，不会改口令。这条命令在宿主机上执行，读到的数据库主机名是 `mysql`，需要能解析这个名字。

测试：

```bash
make test              # 单元测试，不连数据库
make test-integration  # 数据层集成测试，连 root:root@tcp(127.0.0.1:3306)/，自建 *_it 库
make e2e               # 全部服务启动并跑过种子后，经 http://127.0.0.1:8080 走注册到收通知的全链路
```

前端：`frontend/web` 是 `npm run dev`（5174），`frontend/admin-web` 是 `npm run dev`（5173）。开发时请求经 Vite 转到 gateway 的 8080。构建产物用相对路径 `/v1`，容器里的 Nginx 把 `/v1` 反代到 `http://gateway:8080`，浏览器访问 `127.0.0.1:5174` 和 `127.0.0.1:5173` 即可。

## CI

推送和 Pull Request 都会跑 Go 检查、MySQL 集成测试，以及 web、admin-web 的测试和构建，并构建上面 10 个业务镜像。Pull Request 只构建，不推送。CI 只构建镜像，不把 yaml 打进镜像。本地和线上用同一份 `deploy/compose.yaml`，配置在旁边的 `config/`。

推到默认分支后，用 `GITHUB_TOKEN` 推到 GHCR，标签为提交 SHA 和 `latest`：

- `ghcr.io/tmjwjx/supermarket-gateway`
- `ghcr.io/tmjwjx/supermarket-user`
- `ghcr.io/tmjwjx/supermarket-product`
- `ghcr.io/tmjwjx/supermarket-inventory`
- `ghcr.io/tmjwjx/supermarket-order`
- `ghcr.io/tmjwjx/supermarket-payment`
- `ghcr.io/tmjwjx/supermarket-notification`
- `ghcr.io/tmjwjx/supermarket-admin`
- `ghcr.io/tmjwjx/supermarket-web`
- `ghcr.io/tmjwjx/supermarket-admin-web`
