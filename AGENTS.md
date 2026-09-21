# 仓库规范

本文件是 supermarket 的分层约定。改代码时按这里的依赖方向走，不要为了省事反着引。

## 目录

```
api/<domain>/<version>/     Proto 源文件和生成桩。对外契约。
app/<svc>/cmd/<app>/        入口、Wire、main.go。
app/<svc>/configs/          运行时配置（config.yaml）。禁止提交真实密钥。
app/<svc>/internal/conf/    手写配置结构体（conf.go）。不要再为配置写 proto。
app/<svc>/internal/server/  HTTP / gRPC 服务器装配。
app/<svc>/internal/service/ 传输适配；一个资源一个文件。
app/<svc>/internal/biz/     领域模型、用例、仓库接口、错误。
app/<svc>/internal/data/    仓库实现和存储客户端。
```

## 分层与依赖

三种模型走三层：`biz` 拥有 DO，`data` 拥有 PO；`service` 只在边界上做转换。

```
   客户端 ──► DTO ──► service ──► DO ──► biz ──► DO ──► data ──► PO ──► 存储
                                    ▲                ▲
                                    │ 声明            │ 实现
                                    └─── 仓库接口 ────┘

   DTO  数据传输对象 — proto 请求 / 响应。
   DO   领域对象     — 纯业务模型，不带 proto，不带存储标签。
   PO   持久化对象   — 存储形态，归 data 所有。
```

| 层       | 拥有 | 边界上说的话 | 禁止说的话           |
|----------|------|--------------|----------------------|
| service  | —    | DTO ↔ DO    | PO、存储客户端       |
| biz      | DO   | DO           | DTO、PO、存储客户端  |
| data     | PO   | DO ↔ PO     | DTO                  |

- `service` 引用 `api/...`（DTO）和 `biz`（DO）。禁止引用 `data`。
- `biz` 只有错误码枚举可以引用 `api/...`。禁止引用 `service`、`data`。仓库接口在这里声明，这是依赖倒置的缝。
- `data` 引用 `biz` 以实现仓库接口。禁止引用 `service`，禁止引用 DTO。
- 只有 `cmd` 通过 Wire 把各层接起来。

箭头反了就是分层错误；改设计，不要硬加 import。

### 各层职责

**service（DTO ↔ DO）**

- `convert<Resource>` 把进入的 proto 收成 DO。回程在返回处当场拼；返回类型以 proto 为准——通常是资源本身（`return &v1.<Resource>{...}, nil`），有时是列表包装（`*v1.<Resources>Set`），删除则是 `&emptypb.Empty{}`。当场拼是为了每个 handler 自洽。
- 嵌入 `Unimplemented<Resource>ServiceServer`。
- 资源若是 AIP 风格的列表 / 部分更新：用 `filtering` / `ordering` / `pagination` 解析列表请求，用 `fieldmask.Update` 做局部更新。注册、登录这类 RPC 不必硬套。
- 在 service 边界校验入参，再交给用例。
- 返回 `biz` 的错误。不写业务规则，不碰存储，不碰 PO。

**biz（只认 DO）**

- 拥有 DO（`type <Resource> struct` — 无 proto、无存储标签）、用例、仓库接口（`type <Resource>Repo interface`）。
- 拥有用 `errors.NotFound` / `errors.BadRequest` 加上 API 错误码枚举拼出来的类型错误。
- 需要列表时再提供 `ListOption`（`ListFilter`、`ListOrderBy`、`ListOffset`、`ListLimit`），调用方组合查询，不把存储原语漏上去。

**data（DO ↔ PO）**

- *仓库形态*：实现 `biz.<Resource>Repo`。构造函数返回接口，不返回具体类型：
  `func New<Resource>Repo(d *Data) biz.<Resource>Repo`。
- *PO 与转换*：存储形态和 DO 不一致时再定义 PO。PO 留在 `data` 里。用自由函数 `new<Resource>`（DO → PO，写）和 `toBiz`（PO → DO，读）。驱动专用的 builder 不得离开 `data`。
- *共享客户端*：`*Data`（声明在 `internal/data/data.go`）持有长寿命存储客户端。仓库收 `*Data`，自己不建连接。
- *查询*：在仓库内部把 `ListOptions.Filter` / `OrderBy` 译成存储驱动的查询语言。
- *错误*：把驱动错误映射成 `biz` 的类型错误，上层不要按驱动分支。

**server**

- 构造 HTTP / gRPC 服务器，挂中间件，注册服务。不做翻译，不写业务。

### 加一个资源

1. **DTO**：在 `api/<domain>/<version>/` 写清 RPC（CRUD 资源用 `Create` / `Get` / `List` / `Update` / `Delete`；身份类用 `Register` / `Login` / `GetUser` 这类语义名），然后 `make api`。
2. **DO + 仓库接口**：两者都在 `biz` 声明；用例建在接口上。
3. **仓库实现**：在 `data` 里实现并返回 `biz.<Resource>Repo`；形态不一致再加 PO 和转换函数。
4. **装配**：仓库构造进 `data.ProviderSet`，用例进 `biz.ProviderSet`，服务进 `service.ProviderSet`；在 `internal/server` 注册 HTTP / gRPC。
5. **再生成**：`make all` 刷新 Wire 和 `go.mod`。

### 测试缝

测试和被测代码放一起（`*_test.go`）。分层测：service 假用例，biz 假仓库，data 在存储边界测仓库实现。

## 生成文件

用 `make api` 或 `make all` 重新生成；不要手改 `*.pb.go`、`*_grpc.pb.go`、`*_http.pb.go`、`wire_gen.go`。

## 命名与错误码

- 资源名：`<Resource>`（如 `User`）；集合 RPC：`List<Resources>`。
- 类型：仓库 `<Resource>Repo`，用例 `<Resource>Usecase`，服务 `<Resource>Service`。PO 放在 `internal/data/`；名称跟存储驱动走，用 `new<Resource>(do)` / `toBiz(po)` 转换。
- 错误码：写在 `api/<domain>/<version>/error_reason.proto`，在 `biz` 里暴露成 `Err<Resource><Cause>`。

## 注释

- `.proto` 不写注释；名字和字段把契约说清楚。
- 手写实现代码的注释尽量用中文。只补名字读不出来的东西：约束、不变量、为什么这样。名字已经说清的不要再写一遍。不要写阶段、排期、以后再加。
- 实现函数把功能说清楚；复杂逻辑把逻辑说清楚。
- 标识符（包名、类型、RPC、字段、错误码）保持英文。
- 生成文件不要手改；插件产出的英文注释保持原样。

## 提交与安全

- Conventional Commits：`feat:`、`fix:`、`refactor:`、`chore(deps):`、`docs:`、`test:`。生成文件和它的源放在同一次提交。
- 不要把真实密钥写进 `configs/config.yaml`。
