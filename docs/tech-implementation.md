# supermarket 技术实现文档

本文说明从现在到「开发完成」之间，每个服务在技术上怎么实现。开发完成的定义是 README 阶段 1 的终点：从空库开始，本地能走通注册、浏览、加购物车、下单、支付、收到通知、后台看到这笔订单。运维上线、数据库调优、压测、秒杀、AI 客服在它之后，不在本文范围。

商品服务的业务规则已经写在 `docs/product-design.md`。本文只补它的技术细节，规则部分不重复。

**阅读约定**

标「（新增）」的名称现在还不存在，是要写的东西。没标的来自现有代码。文中的数值和取舍都已拍板，决策汇总见第 16 章。

---

## 1. 整体架构

所有外部请求只进 gateway。gateway 按路由决定要不要验令牌，然后用 gRPC 转给后面的服务。服务之间也用 gRPC 同步调用。第 11 步接入 Kafka 后，一部分「做完了通知一声」的调用改成发事件。

```
浏览器 web(5174)  管理台 admin-web(5173)
        │ HTTP /v1/...（开发时由 Vite 代理）
        ▼
   gateway :8080  ── 验令牌 · 传身份 · 转发
        │ gRPC
  ┌─────┼────────┬───────────┬──────────┬──────────┬──────────┐
  ▼     ▼        ▼           ▼          ▼          ▼          ▼
 user  product  inventory   order     payment   notification admin
 9000  9001     9002        9003      9004      9005（新增）  9006（新增）
  │     │        │           │          │          │          │
  └─────┴────────┴───── MySQL（按服务分库）─────┴──────────┴──────────┘
        │                    │
      Redis               Kafka（第 11 步）
```

| 服务 | 目录 | HTTP | gRPC | 数据库 |
|---|---|---|---|---|
| gateway | `app/gateway` | 8080 | — | 无 |
| user | `app/user` | 8000 | 9000 | `user`（现为 `dev`，第 1 步改名） |
| product | `app/product` | 8001 | 9001 | `product` |
| inventory | `app/inventory` | 8002 | 9002 | `inventory` |
| order | `app/order` | 8003 | 9003 | `order` |
| payment | `app/payment` | 8004 | 9004 | `payment` |
| notification | `app/notification`（新增） | 8005 | 9005 | `notification` |
| admin | `app/admin`（新增） | 8006 | 9006 | `admin` |

各服务自己的 HTTP 端口只给本地调试用。浏览器永远只打 gateway。

---

## 2. 通用技术约定

这一章的规则对所有服务都成立，后面各服务章节不再重复。

### 2.1 分层与目录

每个服务照 `AGENTS.md` 的三层走：service 做 proto 和领域对象的互转并校验入参；biz 持有规则和仓库接口；data 实现仓库、持有数据库和缓存连接。每个资源在三层里各占一个目录，比如 `internal/biz/order/`。只有 `cmd` 里的 Wire 把三层接起来。

四个新服务现在只有 `cmd`、`configs`、`internal/conf`、`internal/server`，biz、service、data 目录里只有一行包声明。它们的 `wire.go` 目前只装配了 server，加第一个资源时要把 data、biz、service 的 `ProviderSet` 接进去，否则 Wire 会报「未使用的 provider」。

### 2.2 契约与生成

每个服务的契约放在 `api/<服务>/v1/`，一个资源一个 proto 文件，错误码统一放在同目录的 `error_reason.proto`。写法照现有的 `api/user/v1/user.proto`：HTTP 路由用 `google.api.http` 注解，必填字段用 `field_behavior` 标 `REQUIRED`，proto 里不写注释。

改完 proto 执行 `make api`，它按 `buf.gen.yaml` 生成 Go 桩和 HTTP 桩，并把合并后的 `api/openapi.yaml` 刷新。生成文件不手改。`buf.yaml` 用 STANDARD 规则，所以每个 RPC 的请求和响应都要是独立的消息，响应名以 `Response` 结尾。

只在服务之间调用的 RPC（比如给 order 用的 `BatchGetSkus`）也写进 proto，但不加 HTTP 注解，这样它不会出现在 HTTP 路由里。

### 2.3 配置

配置是手写结构体，放在各服务的 `internal/conf/conf.go`，不为配置写 proto。超时一律写毫秒数字段 `timeout_ms`，由结构体上的 `Timeout()` 方法换算，照 `app/user/internal/conf/conf.go`。

每个服务有两份配置：`configs/dev.yaml` 给本地容器，`configs/prod.yaml` 给服务器。镜像不打包配置，启动时把其中一份挂到 `/configs/app.yaml`。买家密钥是 `tmjwjx-user-jwt-secret`，运营密钥是 `tmjwjx-admin-jwt-secret`，写在 `dev.yaml`。`prod.yaml` 的地址和密钥等服务器接入后再填。每个服务一份自己的库名。

服务要调别的服务时，在配置里加 `client.<服务>.addr` 和 `client.<服务>.timeout_ms`，写法照 `app/gateway/internal/conf/conf.go` 的 `Client`。开发环境写 Compose 里的服务名，不用注册中心。

### 2.4 标识、金额、时间

- **实体 id** 用 UUID v7，由 data 层在写入时生成，照 `app/user/internal/data/user/user.go` 的 `Create`。v7 带时间前缀，按 id 排序大致就是按创建时间排序，插入也不会打散索引。
- **订单号** 另外生成一个给人看的字符串，和 id 分开。格式是「年月日时分秒＋6 位随机数」，靠唯一索引兜底，撞上就重新生成一次。
- **金额** 一律用整数分，数据库列是 64 位整数，proto 字段是 `int64`，字段名以 `_amount` 或 `_price` 结尾。前端只在显示时换算成元。任何服务都不用浮点数算钱。
- **时间** 在 proto 里用 `google.protobuf.Timestamp`，数据库里用 `DATETIME(3)`。连接串带 `loc=Local`，本地开发时存的是本机时区，和 user 现有做法一致。

### 2.5 错误

错误在 biz 层定义，用 Kratos 的 `errors.NotFound`、`BadRequest`、`Conflict`、`Unauthorized`、`Forbidden` 加上 proto 里的错误码拼出来，照 `app/user/internal/biz/user/user.go` 开头那组 `ErrUser...`。命名是 `Err<资源><原因>`，proto 枚举值是 `<服务>_<原因>`。

data 层负责把数据库驱动的错误翻译成 biz 的错误：找不到记录变成 `NotFound`，唯一索引冲突（MySQL 1062）变成对应的 `Conflict`。上层不按驱动判断。

错误经 gRPC 传回 gateway 时，Kratos 会带着原因码和 HTTP 状态一路传出去，gateway 原样返回给浏览器，不改写。前端按 `message` 显示，照 `frontend/web/src/api/client.ts` 的 `send`。

### 2.6 列表、分页、局部更新

资源类的列表照 AIP 风格：请求里有 `filter`、`order_by`、`page_size`、`page_token`，响应里有下一页的 `next_page_token`。解析用现有依赖 `go.einride.tech/aip`。biz 层把解析结果收成 `ListOption`（新增），data 层再翻译成 SQL，AIP 的类型不进 data。

`page_size` 默认 20，最大 50。翻页令牌里编码的是偏移量，客户端只能原样传回，不能自己拼。

局部更新请求带 `update_mask`，service 层用 fieldmask 只取出要改的字段交给 biz。

注册、登录、下单、支付这类动作型 RPC 不套 AIP。

### 2.7 服务间调用

调用方在 `internal/client/`（新增）里为每个下游建一个 gRPC 客户端，写法照 `app/gateway/internal/client/user.go` 的 `NewUserClient`，由 Wire 注入到需要它的 biz 用例里。biz 里声明一个窄接口（比如「按 id 批量取 SKU」），client 包实现它，biz 不直接依赖别的服务的 proto。

每次调用都带超时，默认 1000 毫秒。只读调用失败时不重试；写调用是否重试看它是不是幂等的，见 2.9。下游超时或连不上时，调用方返回 503，前端提示稍后再试。

### 2.8 身份传递

现在 gateway 验完令牌只放行：`app/gateway/internal/auth/jwt.go` 的 `Verifier.Verify` 只返回有没有错，不返回用户是谁；`app/gateway/internal/server/http.go` 的 `requireToken` 也不往下游带任何东西。收藏、购物车、订单都要知道当前用户，所以这里要改。

改法分三步。`Verify` 改成验签成功后返回令牌里的 `sub`。gateway 的上游 gRPC 客户端挂上 Kratos 的 `metadata.Client()` 中间件，要登录的路由在转发前把 `sub` 写进元数据，键名是 `x-md-global-user-id`。下游服务的 gRPC 服务器挂 `metadata.Server()`，service 层从元数据取出用户 id，取不到就返回 401。

gateway 的 HTTP 服务器不挂 `metadata.Server()`，所以浏览器自己带的 `x-md-` 请求头不会被透传，客户端没法冒充别人。下游服务只信任元数据里的用户 id，不看请求体里传的用户 id。

gateway 的路由分四种鉴权模式，每条路由写明用哪一种：

| 模式 | 行为 |
|---|---|
| 公开 | 不看令牌，直接转发 |
| 可选 | 令牌有效就带上用户 id；没有或无效当作未登录，照样转发（用于商品详情记浏览） |
| 登录 | 令牌无效返回 401，不转发 |
| 后台 | 要 admin 服务签发的后台令牌，并按角色检查权限，见 11.2 |

### 2.9 幂等

会被重试或重复点击的写操作都要幂等，靠业务上的唯一键保证，而不是靠调用方小心：

- **下单**：前端每次打开结算页生成一个请求号，随下单请求带上。order 按「用户＋请求号」建唯一索引，同一个请求号第二次进来，直接返回第一次建好的订单。
- **预扣、确认、释放库存**：按订单 id 做唯一键，重复调用返回同样的结果，不会扣两次。
- **支付成功回调**：支付单状态用条件更新，只有「待支付」能变成「已成功」，重复回调什么都不改。
- **通知订单已支付**：order 收到同一个支付单的第二次通知直接返回成功。
- **消费 Kafka 事件**：每个消费者记下处理过的事件 id，重复的跳过。

### 2.10 一致性

单个服务内部，一次业务动作涉及的多张表放在一个数据库事务里。

跨服务的动作不用分布式事务，靠「先占住、成功再确认、失败就释放」加上定时扫描兜底。最典型的是下单扣库存，见 7.3 和 9.3。原则是：任何一步失败，最终结果只能是「订单建成且库存占住」或者「订单没建且库存没少」，不能出现一边有一边没有。短时间的中间状态由扫描任务收拾。

第 11 步接入 Kafka 后，要发事件的服务在同一个事务里把事件写进本库的发件箱表 `outbox_events`（新增），再由后台任务投递到 Kafka，这样业务改了就一定能发出去。见第 12 章。

### 2.11 定时任务

订单超时关闭、库存预占过期释放、支付回调补偿、发件箱投递都是服务进程内的定时任务，随服务启动。每个服务可能起多个进程，所以任务处理每条记录时都用条件更新抢占，抢不到就跳过，多个进程同时跑也不会重复处理。

### 2.12 日志与追踪

日志用 `log/slog`，照各服务 `main.go` 里的写法，带上服务名和版本。每个服务的 HTTP 和 gRPC 服务器都挂 `recovery.Recovery()`，照现有 server 代码。

`main.go` 里已经接了 `tracing.TraceAttrs` 取追踪字段，但还没有挂追踪中间件，调用链串不起来。第 1 步在 gateway 和各服务的服务器、客户端都挂上 Kratos 的追踪中间件，一次请求经过的所有服务在日志里带同一个追踪 id。

### 2.13 建表

本地开发时每个服务启动会自动建表，由配置 `auto_migrate` 控制，照 `app/user/internal/data/data.go` 的 `NewData`。数据库本身不由服务建，由第 3 章的初始化脚本建。

---

## 3. 工程底座

### 3.1 本地依赖

仓库根目录新增 `deploy/compose.yaml`（新增），用 Docker Compose 起三样东西：MySQL 8、Redis 7，以及第 11 步才需要的 Kafka。这套服务是本机共用的，启动时不建业务库。本项目的七个库由 `deploy/mysql/init.sql`（新增）单独执行一次来建。数据放在命名卷里，重启不丢；要清空就删卷。

Makefile 加两个目标：`make up` 拉起依赖，`make down` 停掉。

### 3.2 user 库改名

user 的连接串直接写在 `app/user/configs/dev.yaml`，指向 `user` 库。本地没有要保留的数据，改完重启 user 会在新库里重新建表，旧的注册账号需要重新注册。

### 3.3 持续集成

新增 `.github/workflows/ci.yaml`（新增），每次推送和合并请求时跑：

1. `buf lint`，契约不合规就失败。
2. `make api` 后检查工作区有没有变化。有变化说明有人改了 proto 没重新生成。
3. `go build ./...` 和 `go test ./...`。
4. 两个前端各自 `npm ci` 和 `npm run build`。

data 层的测试要连真实 MySQL，CI 里用服务容器起一个 MySQL，见第 14 章。

### 3.4 本地启动顺序

README 的「本地开发」一节补上：先 `make up`，再按 user、admin、product、inventory、order、payment、notification、gateway 的顺序启动，最后启动两个前端。gateway 放在后面是因为它启动时就会去连各个上游。种子数据在 product 和 inventory 启动后执行一次。

---

## 4. user 服务

### 4.1 现状

注册、登录、查资料已经实现。注册时用 bcrypt 存密码哈希，登录时比对哈希，成功后由 `UserUsecase.issueToken` 签一张 HS256 的 JWT，载荷只有 `sub`（用户 id）、`iat`、`exp`，默认 7 天。这部分不再改动。

### 4.2 查资料要核对本人

现在持有任意一张有效令牌，就能按路径里的 id 查任何人的资料。改法：gateway 按 2.8 把用户 id 传给 user；`UserService.GetUser` 从元数据取出用户 id，和路径里的 id 比，不一样就返回 `USER_FORBIDDEN`（403，新增错误码）。只在 service 层判断，biz 的 `GetUser` 保持按 id 查，给内部调用用。

同时新增 `GetMe`（新增），路由 `GET /v1/users/me`，直接按元数据里的用户 id 查。前端的「我的」页改用它，不再依赖本地存的 `user_id`。

### 4.3 停用账号的旧令牌

gateway 验令牌只算签名，不查库，所以账号被停用后，旧令牌在过期前仍然能用，最长 7 天。开发完成前不处理：停用账号要靠后台，而后台在第 10 步才有。以后要堵的话，做法是 user 停用账号时把用户 id 写进 Redis 的停用集合，gateway 验签通过后再查一次这个集合，命中就返回 401。

### 4.4 收货地址

下单需要地址，README 也把地址列在 user 的职责里。入口是 `AddressService`（新增）的 `CreateAddress`、`UpdateAddress`、`DeleteAddress`、`ListAddresses`、`GetAddress`、`SetDefaultAddress`，gateway 路由都用「登录」模式，路径在 `/v1/addresses` 下。

地址存在 `addresses` 表（新增）：用户 id、收货人、手机号、省、市、区、详细地址、是否默认。每个用户最多 20 条，超过返回 400。

一个用户只能有一个默认地址。设默认时，在一个事务里先把这个用户的其它地址都改成非默认，再把这一条改成默认。新建的第一条地址自动成为默认。删掉默认地址后，不自动指定新的默认。

所有操作只认元数据里的用户 id。按 id 操作别人的地址，一律当作找不到，返回 404，不暴露那条地址存在。

order 下单时通过内部 RPC `GetAddress` 按「用户 id＋地址 id」取地址，把内容抄进订单。之后用户改地址，已下的订单不变。

### 4.5 后台账号不在这里

运营账号放在单独的 admin 服务里，user 只管买家。见第 11 章。

---

## 5. gateway

### 5.1 结构

gateway 没有自己的数据，也不设 biz 和 data。每个上游一个转发器，放在 `app/gateway/internal/server/`：现有的 `userProxy`，新增 `productProxy`、`inventoryProxy`、`orderProxy`、`paymentProxy`、`notificationProxy`（新增）。再加一个 `adminProxy`（新增）转发后台登录。每个转发器的写法照 `userProxy.register`：绑定请求、标记操作名、经中间件调上游、原样返回结果。

每个上游在 `app/gateway/internal/client/` 下有一个客户端构造函数，照 `NewUserClient`；地址写在 `app/gateway/configs/dev.yaml` 的 `client` 下。

鉴权模式做成四个包装函数，现有的 `requireToken` 改造成「登录」模式，另加「可选」和「后台」（新增）。

### 5.2 路由总表

| 路由 | 上游 | 模式 | 步骤 |
|---|---|---|---|
| `POST /v1/users/register`、`POST /v1/users/login` | user | 公开 | 已有 |
| `GET /v1/users/me` | user | 登录 | 6 |
| `GET /v1/users/{id}` | user | 登录，只能查自己 | 6 |
| `/v1/addresses` 下的增删改查 | user | 登录 | 8 |
| `GET /v1/categories`、`GET /v1/brands` | product | 公开 | 2 |
| `GET /v1/products`、`GET /v1/products/{id}/detail` | product | 公开 | 2 |
| `GET /v1/products/{id}` | product | 可选 | 2 |
| `GET /v1/products:search`、`GET /v1/recommendations` | product | 公开 | 5 |
| `/v1/favorites`、`/v1/browse-histories` | product | 登录 | 6 |
| `GET /v1/products/{id}/reviews` | product | 公开 | 9 |
| `POST /v1/reviews` | product | 登录 | 9 |
| `GET /v1/stocks?sku_ids=` | inventory | 公开 | 7 |
| `/v1/cart/items` 下的增删改查 | order | 登录 | 8 |
| `POST /v1/orders`、`GET /v1/orders`、`GET /v1/orders/{id}` | order | 登录 | 8 |
| `POST /v1/orders/{id}:cancel`、`POST /v1/orders/{id}:confirm` | order | 登录 | 8 |
| `POST /v1/payments`、`GET /v1/payments/{id}` | payment | 登录 | 9 |
| `POST /v1/payments/{id}:simulate` | payment | 登录，只在本地开启 | 9 |
| `/v1/notifications` 下的列表和标记已读 | notification | 登录 | 11 |
| `POST /v1/admin/login` | admin | 公开 | 10 |
| `/v1/admin/...` 下的其它接口 | 各服务 | 后台 | 10 |

公开的商品列表路由固定加上「只看在售」的条件，客户端传什么都覆盖掉。后台的商品列表走 `/v1/admin/products`，不受这个限制。

### 5.3 不做的事

限流和熔断在 README 阶段 4，开发完成前不做。Kratos 自带 `ratelimit` 和 `circuitbreaker` 中间件，到时候挂上即可。

跨域不用处理：开发时前端经 Vite 代理访问 gateway，浏览器看到的是同源。上线时怎么部署，不在本文范围。

---

## 6. product 服务

业务规则见 `docs/product-design.md`，这里只写技术落点。

### 6.1 契约

`api/product/v1/` 下每个资源一个文件（新增）：`brand.proto`、`category.proto`、`attribute.proto`、`product.proto`、`recommendation.proto`、`favorite.proto`、`browse_history.proto`、`review.proto`，外加 `error_reason.proto`。每个文件一个 service，名字是 `<资源>Service`。

错误码（新增）：`PRODUCT_NOT_FOUND`、`PRODUCT_INVALID_ARGUMENT`、`PRODUCT_STATUS_CONFLICT`、`PRODUCT_SPEC_DUPLICATED`、`PRODUCT_NOT_SELLABLE`、`BRAND_NOT_FOUND`、`BRAND_NAME_EXISTS`、`BRAND_IN_USE`、`CATEGORY_NOT_FOUND`、`CATEGORY_NAME_EXISTS`、`CATEGORY_IN_USE`、`CATEGORY_NOT_LEAF`、`ATTRIBUTE_TEMPLATE_IN_USE`、`REVIEW_NOT_ALLOWED`、`REVIEW_EXISTS`。

### 6.2 两张核心表

`products` 和 `skus` 是读写最频繁的两张表，字段如下。其余 13 张表的字段见 `docs/product-design.md` 的「数据表」一节。

**products**

| 字段 | 含义 |
|---|---|
| `id` | UUID v7 |
| `category_id` | 所属二级分类 |
| `brand_id` | 所属品牌 |
| `name`、`subtitle`、`keywords` | 名称、副标题、搜索关键词 |
| `main_image` | 主图地址，冗余第一张图片，列表不用再查图片表 |
| `unit`、`weight_gram` | 单位、重量（克） |
| `services` | 服务承诺，JSON 数组 |
| `status` | 草稿 1、待审核 2、已驳回 3、已通过 4、在售 5、下架 6 |
| `min_price`、`max_price` | 启用 SKU 的最低和最高售价，冗余 |
| `sales_count` | 销量，冗余 |
| `deleted_at` | 软删除时间，非空表示在回收站 |

索引：「状态＋分类＋创建时间」给按分类列表用，「状态＋最低价」给按价格排序用。

**skus**

| 字段 | 含义 |
|---|---|
| `id` | UUID v7，也是 inventory、购物车、订单项里引用的 SKU id |
| `product_id` | 所属商品 |
| `specs` | 规格值组合，JSON 对象，如 `{"容量":"500ml"}` |
| `spec_hash` | 规格组合的哈希，用来判重 |
| `price`、`market_price` | 售价、划线价，单位分 |
| `image`、`barcode` | 图片、条码 |
| `enabled` | 是否启用 |
| `deleted_at` | 软删除时间 |

唯一索引：「商品＋规格哈希」。规格哈希的算法：把规格按名称排序，拼成「名称=值」一行一个的字符串，取 SHA-256。

### 6.3 缓存

商品详情缓存在 Redis，键是 `product:detail:<商品 id>`，值是详情的 JSON，10 分钟过期。data 层读时先查缓存，没命中查库并回填；任何写商品、SKU、参数、图片的事务提交后删掉这个键。删除失败只记日志。

product 的 `NewData` 除了 MySQL 还要打开 Redis。配置照 user 已有的 `Redis` 结构；user 现在虽然配了 Redis 但没有用，product 是第一个真正用 Redis 的服务。

### 6.4 对外依赖

product 要调三个服务，客户端都放在 `app/product/internal/client/`（新增）：

- **inventory**：建 SKU 后建库存记录，上架前检查每个启用 SKU 都有库存记录。接口见 7.2。
- **order**：写评价前确认订单项属于这个用户且已完成。接口见 9.6。
- **user**：写评价时由 product 调 user 的内部接口取一次评价人的显示名，存进评价表，读评价时不再调 user。显示名是昵称；没有昵称就是「用户」加手机号后四位。匿名评价显示「匿名用户」。

### 6.5 种子数据

新增命令 `app/product/cmd/seed`（新增），经 gRPC 调 product 的内部写接口建品牌、分类、属性模板、商品，然后提交、通过、上架，最后配首页推荐。每一步按名称查重，重复执行不会多出数据。第 7 步有了 inventory 之后，脚本再顺带给每个 SKU 入库 100 件。

---

## 7. inventory 服务

inventory 只回答一件事：每个 SKU 还能卖多少，并保证并发下单时不会卖超。它不认识商品名称，只认 SKU id。

### 7.1 数据

**stocks**（新增）：每个 SKU 一条。

| 字段 | 含义 |
|---|---|
| `sku_id` | 唯一 |
| `on_hand` | 仓库里实际有的数量 |
| `reserved` | 已被未支付订单占住的数量 |
| `version` | 每次更新加一，排查用 |

可卖数量 = `on_hand` − `reserved`。这个值不存，每次算。

**stock_reservations**（新增）：每个订单在每个 SKU 上占了多少。

| 字段 | 含义 |
|---|---|
| `order_id`、`sku_id` | 联合唯一 |
| `quantity` | 占住的数量 |
| `status` | 已占住 1、已确认 2、已释放 3 |
| `expires_at` | 过了这个时间还没确认就自动释放 |

**stock_logs**（新增）：每次数量变化一条，写明原因（入库、出库、预占、确认、释放）、关联的订单 id、变化前后的数量。只增不改。

### 7.2 契约

`api/inventory/v1/stock.proto`（新增），`StockService` 包含：

- `CreateStock`：内部调用。给一个 SKU 建一条数量为 0 的记录。已存在就直接返回成功。
- `GetStocks`：按一组 SKU id 批量查可卖数量。买家公开，页面只显示「有货」「仅剩 N 件」「缺货」，不显示具体大数。
- `AdjustStock`：后台入库或出库，传变化量和原因。出库后可卖数量不能小于 0。
- `ReserveStock`、`ConfirmReservation`、`ReleaseReservation`：内部调用，给 order 用，见 7.3。

错误码（新增）：`STOCK_NOT_FOUND`、`STOCK_INSUFFICIENT`、`STOCK_INVALID_ARGUMENT`、`RESERVATION_NOT_FOUND`、`RESERVATION_STATUS_CONFLICT`。

### 7.3 预占、确认、释放

**预占**。入口 `ReserveStock`，order 下单时调用，传订单 id、每个 SKU 要多少、过期时间。

inventory 在一个事务里处理整张订单。先把 SKU id 排好序，按顺序逐个更新，这样两张订单同时争抢同一批 SKU 时不会互相锁死。每个 SKU 用一条条件更新：只有「实际数量减已占数量不小于这次要的数量」时，才把已占数量加上这次的数量。更新影响 0 行，说明不够，整个事务回退，返回 `STOCK_INSUFFICIENT`，错误信息里列出哪些 SKU 不够。全部成功就为每个 SKU 写一条「已占住」的预占记录和一条日志。

同一个订单 id 再次预占时，发现预占记录已经存在，直接返回成功，不重复加。这保证了 order 重试下单是安全的。

**确认**。入口 `ConfirmReservation`，订单支付成功后由 order 调用，传订单 id。在一个事务里，把这个订单所有「已占住」的预占记录改成「已确认」，同时每个 SKU 的实际数量和已占数量各减去这次的数量。记录已经是「已确认」就直接返回成功；已经是「已释放」返回 `RESERVATION_STATUS_CONFLICT`，这种情况见 10.4。

**释放**。入口 `ReleaseReservation`，订单取消时调用。把「已占住」的记录改成「已释放」，已占数量减回去，实际数量不变。已经释放过的直接返回成功；已经确认的返回冲突。

**过期释放**。新增定时任务，每分钟一次，找出过期时间已过、仍是「已占住」的预占记录，按订单逐个释放。它兜底两种情况：order 在预占成功后、建订单前崩了，留下一份没人认领的占用；order 关单时调释放失败了。过期时间由 order 在预占时传入，比订单的支付超时多 5 分钟，保证正常情况下总是 order 先关单、自己释放。

### 7.4 前端展示

商品详情页在拿到商品后，再调一次 `GET /v1/stocks?sku_ids=` 拿到每个 SKU 的可卖数量。选中的 SKU 缺货时，数量选择和购买按钮灰掉。这是展示用的，真正能不能买以下单时的预占为准。

---

## 8. 购物车

购物车放在 order 服务里，是 order 下的一个资源，代码在 `internal/biz/cart/`、`internal/data/cart/`、`internal/service/cart/`（新增）。下单时直接读本库的购物车，不跨服务。

### 8.1 数据

**cart_items**（新增）：用户 id、SKU id、数量、是否勾选、加入时间。「用户＋SKU」唯一。每行数量最多 99，每个用户最多 100 行，超过返回 400。

购物车里不存价格和商品名，每次读的时候现取，所以用户看到的永远是当前价。

### 8.2 流程

入口是 `CartService`（新增）的 `AddCartItem`、`UpdateCartItem`、`RemoveCartItems`、`ListCartItems`，路由在 `/v1/cart/items` 下，都要登录。

**加入**：先调 product 的 `CheckSellable` 确认 SKU 可卖，不可卖返回 `PRODUCT_NOT_SELLABLE`。已经在购物车里就把数量加上去，超过 99 按 99 算；不在就新建一行，默认勾选。

**修改**：可以改数量和勾选状态，数量改成 0 等于删除。

**查看**：取出这个用户的全部行，一次调 product 的 `BatchGetSkus` 拿名称、规格、价格、图片、可卖状态，再一次调 inventory 的 `GetStocks` 拿可卖数量，拼好返回。下架或停用的 SKU 标为「已失效」，放在列表最后，不能勾选。数量超过可卖数量的标为「库存不足」。响应里带上勾选商品的合计金额。

下单成功后，order 删掉这次下单用到的购物车行。删除失败不影响订单，用户下次看到还在，自己删掉即可。

---

## 9. order 服务

### 9.1 数据

**orders**（新增）

| 字段 | 含义 |
|---|---|
| `id` | UUID v7 |
| `order_no` | 给人看的订单号，唯一 |
| `user_id` | 下单用户 |
| `request_id` | 下单请求号，「用户＋请求号」唯一，用于幂等 |
| `status` | 待支付 1、已支付 2、已发货 3、已完成 4、已取消 5 |
| `items_amount` | 商品金额合计 |
| `freight_amount` | 运费，开发完成前固定为 0 |
| `pay_amount` | 应付金额 = 商品金额 + 运费 |
| 收货人、手机号、省、市、区、详细地址 | 下单时从地址抄来的快照 |
| `remark` | 买家留言 |
| `expires_at` | 支付截止时间 |
| `stock_released` | 取消后库存是否已释放，给扫描任务补偿用 |
| `stock_confirmed` | 支付后库存是否已确认扣减，给扫描任务补偿用 |
| `paid_at`、`shipped_at`、`completed_at`、`cancelled_at` | 各状态的时间 |
| `cancel_reason` | 买家取消、超时未付 |
| `payment_id` | 成功支付的支付单 |

索引：「用户＋创建时间」给我的订单列表用，「状态＋支付截止时间」给超时扫描用。

**order_items**（新增）：订单 id、SKU id、商品 id、商品名、规格、图片、单价、数量、小计、是否已评价。这些都是下单那一刻的快照，之后商品改价改名都不影响。

### 9.2 状态

| 动作 | 触发 | 允许的当前状态 | 变成 |
|---|---|---|---|
| 支付成功 | payment 通知 | 待支付 | 已支付 |
| 买家取消 | 买家 | 待支付 | 已取消 |
| 超时关闭 | 扫描任务 | 待支付且已过截止时间 | 已取消 |
| 发货 | 后台 | 已支付 | 已发货 |
| 确认收货 | 买家 | 已发货 | 已完成 |
| 自动收货 | 扫描任务 | 已发货且超过 N 天 | 已完成 |

每个状态变化都是一条条件更新，带上「当前状态仍是预期状态」，抢不到返回 `ORDER_STATUS_CONFLICT`（409）。支付超时是 30 分钟；发货后 7 天自动收货。已支付的订单开发完成前不支持取消和退款。

### 9.3 下单

入口 `OrderService.CreateOrder`（新增），`POST /v1/orders`，要登录。请求里带：要买的 SKU 和数量（从购物车勾选的行来，或者详情页的立即购买）、地址 id、留言、请求号。

处理步骤如下，前面任何一步失败都直接返回，不会留下任何数据：

1. **查重**。按「用户＋请求号」查订单，已经有就直接返回那一张。用户连点两次或者网络重试，只会有一张订单。
2. **取商品**。调 product 的 `BatchGetSkus`。有任何一个不可卖，返回 `ORDER_ITEM_NOT_SELLABLE`，列出是哪几件。
3. **算钱**。单价以 product 返回的为准，不信任前端传的价格。算出每行小计、商品合计、应付金额。
4. **取地址**。调 user 的 `GetAddress`，按「当前用户＋地址 id」取，取不到返回 400。
5. **生成 id 和截止时间**。订单 id 在这一步就生成，因为下一步要用它占库存。
6. **占库存**。调 inventory 的 `ReserveStock`，传订单 id、每个 SKU 的数量、过期时间（截止时间再加 5 分钟）。库存不足返回 `ORDER_STOCK_INSUFFICIENT`，列出哪几件不够。
7. **写订单**。在一个事务里写 `orders` 和 `order_items`，状态为待支付。
8. **清购物车**。删掉这次用到的购物车行，失败只记日志。

第 7 步失败时（比如数据库断了），order 立刻调 `ReleaseReservation` 把刚占的库存放回去；如果这次释放也失败，就交给 7.3 的过期释放兜底。order 进程在第 6 步和第 7 步之间崩了，同样由过期释放兜底。这两种情况下，用户看到的是下单失败，库存会在几分钟内回来。

第 6 步调用超时时，order 不知道 inventory 到底占没占上。这时直接返回失败，并调一次释放；inventory 那边的预占记录如果存在，会被这次释放或者过期释放清掉。用户重新提交时换了一个新的订单 id，不会和上一次冲突。前端重新提交用的是同一个请求号，但第 1 步查不到订单（因为订单没写成），所以会正常走一遍。

### 9.4 取消与超时关闭

**买家取消**：入口 `CancelOrder`（新增），`POST /v1/orders/{id}:cancel`。**超时关闭**：新增定时任务，每分钟一次，按支付截止时间从早到晚，每批取 200 张已过截止时间的待支付订单，逐张关闭。

两者走同一套关闭逻辑：先条件更新把订单改成已取消，记下原因和时间，`stock_released` 为假；再调 inventory 释放库存，成功后把 `stock_released` 改为真。释放失败不影响取消本身，扫描任务每次也会捞出「已取消但库存未释放」的订单再调一次释放。

### 9.5 支付成功

入口 `MarkOrderPaid`（新增），内部调用，由 payment 在支付成功后调用，传订单 id 和支付单 id。

order 条件更新：只有待支付的订单能变成已支付，同时记下支付单 id 和支付时间。更新成功后调 inventory 的 `ConfirmReservation`，成功就把 `stock_confirmed` 改为真。确认失败时订单仍是已支付，扫描任务每分钟捞出「已支付但库存未确认」的订单重试。

订单已经是已支付，且支付单 id 相同，说明是重复通知，直接返回成功。订单已经被取消（用户付款那一刻刚好超时），返回 `ORDER_STATUS_CONFLICT`，由 payment 按 10.4 退款。

### 9.6 查询与其它

- `ListOrders`：我的订单，按状态过滤，按创建时间倒序，AIP 分页。只返回当前用户的。
- `GetOrder`：订单详情，带订单项。不是当前用户的订单当作找不到，返回 404。
- `ConfirmOrder`：确认收货。
- `ShipOrder`：后台发货，第 10 步开放。
- `GetOrderItem`：内部调用，给 product 写评价前核对：订单项属于这个用户、订单已完成、还没评价过。评价写成功后 product 再调 `MarkOrderItemReviewed`（新增）把「是否已评价」改为真。
- 订单完成时通知 product 给对应商品加销量。开发完成前先同步调 product 的 `IncreaseSales`（新增），失败只记日志；第 11 步改成事件。

错误码（新增）：`ORDER_NOT_FOUND`、`ORDER_INVALID_ARGUMENT`、`ORDER_STATUS_CONFLICT`、`ORDER_ITEM_NOT_SELLABLE`、`ORDER_STOCK_INSUFFICIENT`、`ORDER_ADDRESS_NOT_FOUND`。

---

## 10. payment 服务（模拟）

开发完成前不接真实支付渠道。payment 自己扮演渠道：生成支付单，由一个本地页面模拟「支付成功」或「支付失败」，然后走和真实回调一样的处理流程。以后接真实渠道时，只换「谁来触发回调」这一层。

### 10.1 数据

**payments**（新增）

| 字段 | 含义 |
|---|---|
| `id` | UUID v7 |
| `order_id` | 订单，唯一，一张订单只有一张支付单 |
| `user_id` | 付款用户 |
| `amount` | 金额，单位分，建单时从订单取 |
| `status` | 待支付 1、已成功 2、已失败 3、已关闭 4、已退款 5 |
| `channel` | 固定为 `mock` |
| `channel_trade_no` | 渠道流水号，模拟时自己生成 |
| `notified` | 是否已成功通知 order |
| `paid_at`、`refunded_at` | 时间 |

### 10.2 建支付单

入口 `PaymentService.CreatePayment`（新增），`POST /v1/payments`，要登录，传订单 id。

payment 调 order 的 `GetOrder` 取订单，核对三件事：订单属于当前用户、订单是待支付、订单没过截止时间。任一不满足返回 400 或 409。这张订单已经有待支付的支付单，就直接返回它；有已失败的，把它重置成待支付再返回，让用户重试。金额以订单的应付金额为准。

响应里带支付单 id。前端跳到模拟支付页 `/pay/:id`（新增）。

### 10.3 模拟回调

入口 `SimulatePayment`（新增），`POST /v1/payments/{id}:simulate`，传「成功」或「失败」。这个接口只在本地开启，由配置开关控制，开关关掉时返回 404。

处理和真实回调完全一样：条件更新支付单，只有待支付能变成已成功或已失败。重复回调什么都不改，直接返回当前状态。

变成已成功后，调 order 的 `MarkOrderPaid`。成功就把 `notified` 改为真。失败（超时、连不上）就留着，新增定时任务每分钟捞出「已成功但未通知」的支付单重试，直到成功。order 那边的处理是幂等的，重试多少次都只会改一次订单。

### 10.4 付款时订单已取消

用户在截止时间前一刻打开支付页，点支付时订单刚好被超时关闭，会出现钱付了但订单已取消。`MarkOrderPaid` 返回冲突时，payment 把支付单改成已退款，模拟原路退回，`notified` 改为真，不再重试。前端支付结果页显示「订单已超时取消，款项已退回」。

### 10.5 对账

新增定时任务，每天凌晨 2 点执行一次，比对前一天「已成功」的支付单和 order 里「已支付」的订单，两边对不上的写进日志，并在管理台的对账页列出来。开发完成前只做发现，不自动修复。

错误码（新增）：`PAYMENT_NOT_FOUND`、`PAYMENT_INVALID_ARGUMENT`、`PAYMENT_STATUS_CONFLICT`、`PAYMENT_ORDER_NOT_PAYABLE`。

---

## 11. 后台

第 10 步之前，后台接口一律不经 gateway 开放，写数据只能在服务内部调用，靠种子脚本。操作人传固定值 `system`。

### 11.1 admin 服务与运营账号

运营账号放在单独的 `app/admin`（新增），和买家的 user 服务完全分开。骨架照其它服务建，HTTP `8006`、gRPC `9006`，数据库 `admin`。

账号存在 `admin_users` 表（新增）：用户名（唯一）、密码哈希、显示名、角色、状态（正常、停用）。密码用 bcrypt，做法和 user 的 `Create`、`VerifyPassword` 一样。

角色固定三种，写在 biz 代码里，不存表：

| 角色 | 能管的范围 |
|---|---|
| 超级管理员 | 全部，包括运营账号本身 |
| 商品运营 | 品牌、分类、属性、商品、审核、推荐位、库存 |
| 订单客服 | 订单、发货、支付对账 |

入口是 `AdminUserService`（新增）：`Login`（用户名加密码）、`GetMe`，以及只给超级管理员的 `CreateAdminUser`、`ListAdminUsers`、`UpdateAdminUser`（改角色、停用）、`ResetAdminPassword`。登录的校验顺序和买家一样：用户名不存在和密码错误返回同一个错误，停用的账号拒绝登录。

第一个超级管理员由新增命令 `app/admin/cmd/seed` 建，用户名 `admin`，密码取配置 `seed.init_password`。这一项先留空，为空或短于 8 位时命令失败退出。已经存在就跳过。

### 11.2 后台令牌与 gateway 鉴权

admin 登录成功后签一张 HS256 的 JWT，密钥是 admin 配置里的 `auth.jwt_secret`，和买家令牌用的那把 `auth.jwt_secret` 不是同一把。载荷有 `sub`（运营账号 id）、`aud` 固定为 `admin`、`role`（角色）、`iat`、`exp`，有效期 12 小时。

两把密钥分开，两种令牌就天然不能混用：买家令牌拿到后台路由，用后台密钥验签不过；后台令牌拿到买家路由，用买家密钥也验签不过。

gateway 新增一个后台验签器 `NewAdminVerifier`（新增），和现有的 `NewVerifier` 并列，读配置 `auth.admin_jwt_secret`。「后台」模式的处理顺序是：验签，检查 `aud` 是 `admin`，取出角色，再按路由表判断这个角色能不能访问这条路径，不能就返回 403 `GATEWAY_FORBIDDEN`（新增）。通过后把运营账号 id 和角色写进元数据，键名 `x-md-global-admin-id` 和 `x-md-global-admin-role`，再转发。

路由和角色的对应写在 gateway 代码里，按路径前缀：`/v1/admin/brands`、`categories`、`attributes`、`products`、`recommendations`、`stocks` 给商品运营；`/v1/admin/orders`、`payments` 给订单客服；`/v1/admin/users` 只给超级管理员；超级管理员能访问全部。

运营账号被停用后，旧令牌最长还能用 12 小时。开发完成前接受这个窗口。

### 11.3 各服务的后台接口

后台路由统一放在 `/v1/admin/` 下，和买家路由分开。每个服务在 proto 里为后台单独写 RPC，比如 `AdminListProducts`、`AdminShipOrder`（新增），这样买家接口的「只看在售」「只看自己的」限制不会被后台需求削弱。

这些 RPC 从元数据取运营账号 id，写进审核记录、操作日志、发货记录的操作人。取不到就返回 401。它们不再检查角色，角色已经由 gateway 挡过了。

### 11.4 页面

`frontend/admin-web` 现在只有一个空的首页。结构照 `frontend/web`：`src/api/` 放请求，`src/session/`（新增）放后台令牌，`src/pages/` 放页面，`src/routes.tsx` 放路由。后台令牌存储的键名和用户端不同，同一个浏览器同时登录两边不会互相覆盖。

页面清单（新增）：登录、品牌、分类、属性模板、商品列表、商品编辑（四步：基本信息、规格与价格、参数、图片与详情）、审核、回收站、操作日志、推荐位、库存（查看和入库出库）、订单列表与发货、支付对账。

---

## 12. 通知与事件（Kafka）

### 12.1 发事件

要发事件的服务（product、order、payment）各自新增一张 `outbox_events` 表：事件 id、主题、业务键、内容、是否已投递、创建时间。业务改动和写发件箱在同一个事务里，所以业务改了就一定有事件。

每个服务新增一个投递任务，每秒一次，按创建时间每批取 100 条未投递的事件发到 Kafka，发成功就标为已投递。发了但没来得及标记就崩了，下次会再发一遍，所以消费方必须能处理重复。Kafka 客户端用 `github.com/twmb/franz-go`。

### 12.2 主题

| 主题 | 发送方 | 业务键 | 消费方 |
|---|---|---|---|
| `product.sku.created` | product | SKU id | inventory 建库存记录 |
| `order.created` | order | 订单 id | notification |
| `order.paid` | order | 订单 id | notification |
| `order.cancelled` | order | 订单 id | notification |
| `order.shipped` | order | 订单 id | notification |
| `order.completed` | order | 订单 id | product 加销量、notification |
| `payment.refunded` | payment | 支付单 id | notification |

业务键同时作为 Kafka 的消息键，保证同一张订单的事件按顺序到达同一个分区。

### 12.3 哪些调用改成事件

改成事件的只有「做完了告诉对方一声、对方失败也不影响我」的调用：建 SKU 后建库存、订单完成后加销量，以及所有通知。

下单时的预占、支付后的确认、取消时的释放、支付成功通知订单，仍然是同步调用。它们需要立刻知道结果，不能等。

建库存改成事件后，上架前检查库存记录那一步保留，当作兜底。

### 12.4 消费

每个消费方新增一张 `consumed_events` 表，记处理过的事件 id。收到事件先查这张表，处理过就跳过；没处理过就在一个事务里做业务处理并记下事件 id。处理失败不提交偏移量，Kafka 会重投。同一条事件连续失败 5 次就记日志并跳过，避免卡住后面的消息。

### 12.5 notification 服务

新增服务 `app/notification`（新增），照其它四个服务的骨架建，HTTP `8005`、gRPC `9005`。

它消费上面的订单和支付事件，按事件类型套一段文案，写进 `notifications` 表：用户 id、标题、正文、关联的订单 id、是否已读、创建时间。这就是站内信。短信和邮件开发完成前只打日志，不真发。

买家接口 `ListNotifications`、`MarkNotificationsRead`（新增），路由在 `/v1/notifications` 下，要登录。`web` 顶部显示未读数，点进去是消息列表。

---

## 13. 前端

### 13.1 用户端 `frontend/web`

结构保持现在的分法：`src/api/` 每个上游一个文件（现有 `user.ts`，新增 `product.ts`、`inventory.ts`、`cart.ts`、`order.ts`、`payment.ts`、`notification.ts`），都经 `client.ts` 的 `send` 发请求；`src/session/` 管令牌；`src/pages/` 一个页面一个文件；`src/routes.tsx` 管路由。

有四处公共改动：

- **带令牌**：现在每个接口自己拼 `Authorization`，比如 `getUser` 要调用方传 token。改成 `send` 统一从 `loadToken` 取，有就带上。
- **401 处理**：`send` 收到 401 时调 `clearSession`，跳到登录页，登录后回到原页面。
- **要登录的页面**：路由上加一层守卫（新增），没有令牌直接跳登录页。
- **金额**：新增 `formatPrice`（新增），把分换成「¥19.99」。页面不自己除 100。

页面按步骤增加（新增）：首页、商品详情（第 2 步）；规格选择（第 3 步）；分类页、搜索页、首页推荐（第 5 步）；我的收藏、浏览记录（第 6 步）；详情页库存提示（第 7 步）；购物车、地址管理、结算页、订单列表、订单详情（第 8 步）；模拟支付页、支付结果页、评价（第 9 步）；消息列表（第 11 步）。

结算页在打开时生成下单请求号，存在页面状态里。用户重复点「提交订单」用的是同一个请求号。

### 13.2 管理台 `frontend/admin-web`

见 11.4。

---

## 14. 测试

- **biz**：用假仓库和假的下游客户端测规则，照 `app/user/internal/biz/user/user_test.go`。重点测状态流转、上架校验、下单每一步失败时的结果。
- **service**：用假用例测入参校验和 proto 互转，照 `app/user/internal/service/user/user_test.go`。
- **data**：连真实 MySQL 测仓库，每个测试用独立的库或事务回滚。重点测唯一冲突的翻译、条件更新在并发下只成功一次、预占库存在并发下不超卖（开 50 个协程同时抢 10 件，最终只有 10 个成功，`reserved` 等于 10）。本地用 `make up` 起的 MySQL，CI 用服务容器。
- **gateway**：用假的上游测四种鉴权模式：没有令牌、错误令牌、过期令牌、有效令牌在每种模式下的结果，以及客户端自带的 `x-md-` 请求头不会被转发。
- **前端**：两个前端 `npm run build` 通过。
- **全链路验收**：第 12 步写一个脚本（新增），从空库开始经 gateway 依次调用注册、登录、浏览、加购物车、下单、模拟支付、查通知，每一步断言结果。

---

## 15. 交付步骤

和之前列的 12 个步骤一一对应。

| 步骤 | 内容 | 本文章节 |
|---|---|---|
| 1 | 工程底座：Compose、建库脚本、CI、user 库改名、启动顺序 | 3 |
| 2 | 商品目录与只读、种子数据、gateway 商品公开路由、首页和详情 | 6，商品设计第 1 期 |
| 3 | 属性与规格 | 商品设计第 2 期 |
| 4 | 状态与审核 | 商品设计第 3 期 |
| 5 | 搜索、推荐、详情缓存 | 6.3，商品设计第 4 期 |
| 6 | gateway 传身份、查资料只能查自己、收藏和浏览记录 | 2.8、4.2、商品设计第 5 期 |
| 7 | inventory 全部、商品接上建库存和上架检查 | 7 |
| 8 | 地址、购物车、下单、取消、超时关闭 | 4.4、8、9 |
| 9 | 支付、对账、销量回写、评价 | 10、9.6、商品设计第 7 期 |
| 10 | admin 服务与运营账号、后台令牌与路由、各服务后台接口、管理台页面、发货 | 11 |
| 11 | Kafka、发件箱、notification、站内信 | 12 |
| 12 | 全链路验收脚本 | 14 |

每一步完成的标准：新增代码三层单测通过，接口经 gateway 调得通，对应页面能用，CI 是绿的。

---

## 16. 决策汇总

| 事项 | 结论 | 章节 |
|---|---|---|
| 运营账号放哪 | 单开 admin 服务 | 11 |
| 购物车放哪 | 放在 order 里 | 8 |
| user 的库 | 从 `dev` 改为 `user`，每个服务在自己的 yaml 里写连接串 | 2.3、3.2 |
| 传用户 id 的键名 | `x-md-global-user-id`；后台是 `x-md-global-admin-id` 和 `x-md-global-admin-role` | 2.8、11.2 |
| 评价前核对订单项 | order 提供 `GetOrderItem` 和 `MarkOrderItemReviewed` | 9.6 |
| 建库存 | inventory 提供 `CreateStock`，重复调用只建一条 | 7.2 |
| 销量回写 | 先同步调 `IncreaseSales`，接 Kafka 后改事件 | 9.6、12.3 |
| 停用账号的旧令牌 | 开发完成前不拦 | 4.3 |
| 追踪 | 第 1 步接上 | 2.12 |

---

## 17. 不做什么

- 秒杀、Redis Lua 预扣、限流、熔断：README 阶段 4。
- Elasticsearch 搜索：先用 MySQL 模糊匹配。
- 真实支付渠道、退款申请、售后。
- 优惠券、满减、会员价、运费模板。
- 图片上传服务：只存地址。
- 真实短信和邮件：只打日志。
- 注册中心、配置中心：地址写在配置里。
- Kubernetes 部署、线上监控：README 阶段 2。
