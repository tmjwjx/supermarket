# 商品服务实现文档

商品服务（`app/product`）回答四件事：卖什么、长什么样、多少钱、现在能不能卖。库存数量归 inventory，促销价归以后的营销服务，下单那一刻的价格快照归 order。本文说明这四件事在代码里怎么落地，分 7 期交付。

文中标「（新增）」的名称现在还不存在，是要写的东西；没标的来自现有代码。跨服务的技术约定见 `docs/tech-implementation.md`。

## 整体思路

请求从浏览器或后台进来，都先到 gateway（`app/gateway`，HTTP `8080`）。gateway 按路由分三种处理：公开接口直接转发；买家接口先验 JWT，再把用户 id 带给 product；后台接口要等运营账号做完才开放。转发走 gRPC，到 product 的 `9001`。

product 内部照 user 的三层走。service 把 proto 收成领域对象并做入参校验；biz 持有规则，比如上架前要满足什么、状态能从哪到哪；data 负责 MySQL 和 Redis，把领域对象存成表记录。product 有自己的数据库 `product`，不和 user 共用。

对外只有一件事要跨服务：order 下单时向 product 批量取 SKU 的名称、规格、价格和是否可售，自己存成快照。SKU 创建后要在 inventory 里有一条库存记录，这件事在第 7 期接上。

```
浏览器 / 后台
   │ HTTP
   ▼
gateway :8080 ── 公开：直接转发
   │           ── 买家：验签后带用户 id 转发
   │           ── 后台：待运营账号
   │ gRPC
   ▼
product :9001 ── service ── biz ── data ── MySQL product 库
                                   └──── Redis 详情缓存
   ▲
   │ gRPC（第 7 期）
order ── 取 SKU 快照          product ──► inventory 建库存记录
```

## 需求覆盖

| 编号 | 需求 | 实现块 | 判定标准 |
|---|---|---|---|
| R1 | 品牌管理 | 1 | 重复名称创建返回 409 `BRAND_NAME_EXISTS`；隐藏的品牌不出现在 `GET /v1/brands` |
| R2 | 两级分类 | 2 | 在第二级下再建子分类返回 400；有子分类或挂着商品的分类删除返回 409 `CATEGORY_IN_USE` |
| R3 | 属性模板与属性 | 3 | 分类绑模板后，该分类下编辑商品只能填模板里的规格和参数 |
| R4 | 商品与 SKU 录入 | 4 | 同一商品下两个 SKU 规格组合相同，第二个返回 409 `PRODUCT_SPEC_DUPLICATED` |
| R5 | 价格用整数分 | 4 | 售价 `1999` 在页面显示为 ¥19.99；改价后列表里的最低价同步变化 |
| R6 | 图片与图文详情 | 4 | 第 10 张图片返回 400；列表接口响应里不含详情正文 |
| R7 | 商品状态与上架校验 | 5 | 没有启用 SKU 的商品上架返回 400，并指出缺哪一项 |
| R8 | 审核与操作日志 | 5 | 驳回后审核记录里能看到理由；改价后操作日志里有改前和改后的价格 |
| R9 | 回收站 | 5 | 删除后买家列表和详情都查不到；恢复后回到下架状态 |
| R10 | 买家列表与详情 | 6 | 未登录能打开首页列表和详情；下架商品详情显示「已下架」 |
| R11 | 过滤、排序、分页 | 6 | 选一级分类时返回其下所有二级分类的商品；`page_size=20` 每页不超过 20 条 |
| R12 | 关键词搜索 | 7 | 搜「牛奶」能搜到名称或关键词里含「牛奶」的在售商品 |
| R13 | 首页推荐位 | 8 | 结束时间已过的推荐不再出现；推荐的商品下架后也不出现 |
| R14 | 详情缓存 | 9 | 第二次打开同一详情不查 MySQL；改价后下一次打开看到新价格 |
| R15 | 收藏 | 10 | 同一商品收藏两次，列表里只有一条 |
| R16 | 浏览记录 | 11 | 浏览第 101 个商品后，最早那条消失；重复浏览同一商品只保留最新一次 |
| R17 | 评价 | 12 | 未完成的订单项评价返回 403；同一订单项第二次评价返回 409 |
| R18 | 给 order 的快照接口 | 13 | 一次传 50 个 SKU id，返回 50 条；其中下架的标为不可售 |
| R19 | 与 inventory 建库存 | 14 | 新建 SKU 后 inventory 里有一条数量为 0 的记录 |
| R20 | 种子数据 | 15 | 空库执行一次种子脚本后首页有商品；再执行一次数量不变 |

## 数据表

全部在 `product` 库，每张表都有创建和更新时间。商品和 SKU 用软删除，回收站就是软删除掉的那些。

| 表（新增） | 业务含义 | 关键约束 |
|---|---|---|
| `brands` | 品牌 | 名称唯一 |
| `categories` | 分类，最多两级 | 同一父级下名称唯一 |
| `attribute_templates` | 属性模板，如「饮料」 | 名称唯一 |
| `attributes` | 模板下的规格或参数 | 同一模板下名称唯一 |
| `products` | 商品（SPU） | 按「状态＋分类」「状态＋最低价」建索引 |
| `skus` | 规格 | 「商品＋规格哈希」唯一 |
| `product_params` | 商品填的参数值 | 「商品＋属性」唯一 |
| `product_images` | 商品图片 | 每个商品最多 9 张 |
| `product_details` | 图文详情，电脑版和手机版各一份 | 一个商品一条 |
| `product_audits` | 审核记录 | 只增不改 |
| `product_logs` | 操作日志 | 只增不改 |
| `recommendations` | 首页推荐位 | 「位置＋商品」唯一 |
| `favorites` | 收藏 | 「用户＋商品」唯一 |
| `browse_histories` | 浏览记录 | 「用户＋商品」唯一 |
| `reviews` | 评价 | 「订单项」唯一 |

## 实现逻辑

### 0. 服务底座

入口是现有的 `app/product/cmd/product/main.go`。现在它只起 HTTP `8001` 和 gRPC `9001`，没有任何资源。

第一期先补三样东西。配置里加数据库和 Redis，写法照 `app/user/internal/conf/conf.go` 的 `Data`。连接串直接写在 `app/product/configs/dev.yaml`，指向 `mysql:3306/product`，不和 user 共用一个库。`app/product/internal/data/data.go` 照 user 的 `NewData` 打开 MySQL 和 Redis，本地开发时自动建表。`app/product/cmd/product/wire.go` 把 data、biz、service 三层接进来。

本地第一次运行前，要手动在 MySQL 里建 `product` 库。服务不负责建库，只负责建表。

每个资源在三层里各占一个目录，比如 `internal/biz/brand/`、`internal/data/brand/`、`internal/service/brand/`（新增）。契约放在 `api/product/v1/`，一个资源一个 proto 文件，错误码统一写在 `api/product/v1/error_reason.proto`（新增）。

### 1. 品牌

后台创建、修改、删除品牌，买家读品牌列表。入口是 `BrandService`（新增）的 `CreateBrand`、`UpdateBrand`、`DeleteBrand`、`ListBrands`。

品牌存在 `brands` 表：名称、首字母、logo 地址、介绍、是否显示、排序值。创建时名称重复，data 层把 MySQL 的唯一冲突翻译成 `BRAND_NAME_EXISTS`（409）。修改走局部更新，用 `fieldmask` 只改传进来的字段。

删除前 biz 先问商品仓库「还有没有商品挂在这个品牌下」。有就拒绝，返回 `BRAND_IN_USE`（409）。

买家读列表时只返回「是否显示」为真的品牌，按排序值从小到大，再按首字母排。后台读列表返回全部。两者是同一个 RPC，区别在 gateway：公开路由固定加上「只看显示的」条件，客户端改不掉。

### 2. 分类

入口是 `CategoryService`（新增）：创建、修改、删除，以及买家用的分类树 `ListCategories`。

分类存在 `categories` 表：名称、父分类 id、图标、排序值、是否显示、绑定的属性模板 id。一级分类的父 id 为空。

创建时 biz 检查层级：父分类必须存在且本身是一级，否则返回 `CATEGORY_NOT_FOUND` 或 400。这样分类最多两级。同一父级下名称不能重复。

删除有两个前提：没有子分类，也没有商品挂在它下面。任一条件不满足，返回 `CATEGORY_IN_USE`。

买家读分类树时，data 一次查出所有「显示」的分类，biz 在内存里拼成两层，每层按排序值排。一级分类被隐藏时，它下面的二级分类也不返回。

### 3. 属性模板与属性

属性决定商品有哪些规格可选、要填哪些参数。入口是 `AttributeService`（新增），管理模板和模板下的属性。

模板存在 `attribute_templates`，属性存在 `attributes`。每个属性有：所属模板、名称、类型（规格或参数）、可选值列表、是否允许手填、排序值。可选值列表存成 JSON 数组。

分类通过「绑定的属性模板 id」挂上模板。编辑商品时，后台先按分类取模板，再按模板出表单：规格类属性用来组合出 SKU，参数类属性用来填展示信息。

保存商品时，biz 校验两件事。每个 SKU 用到的规格名都必须在模板里，规格值要在可选值里（允许手填的除外）。每个参数也必须是模板里的参数类属性。不符合就返回 `PRODUCT_INVALID_ARGUMENT`，并说明是哪个属性。

模板已被分类绑定时不能删，返回 `ATTRIBUTE_TEMPLATE_IN_USE`。属性改名或删除不影响已有商品，已存的规格值和参数值原样保留，只是之后编辑时表单里不再出现。

### 4. 商品、SKU、图片、详情

这一块是商品录入。入口是 `ProductService`（新增）的 `CreateProduct`、`UpdateProduct`、`GetProduct`、`ListProducts`，另有 `GetProductDetail` 单独取图文详情。

一个商品拆在五张表：`products` 存本体，`skus` 存规格，`product_params` 存参数值，`product_images` 存图片，`product_details` 存图文详情。创建和整体修改时，五张表在一个事务里一起写，要么全成功，要么全回退。

`products` 的关键字段：分类 id（必须是二级分类）、品牌 id、名称、副标题、关键词、单位、重量、服务承诺、状态、最低售价、最高售价、销量。

`skus` 的关键字段：规格值组合、售价、划线价、图片、条码、是否启用。价格一律存整数分，页面显示时再除以 100。同一商品下规格组合不能重复。判重时，biz 把规格按名称排序后拼成一串，data 算出哈希存进「规格哈希」字段，靠「商品＋规格哈希」唯一索引兜底。重复时返回 `PRODUCT_SPEC_DUPLICATED`。

最低和最高售价是冗余字段，只算启用的 SKU。每次写 SKU，同一个事务里重新算一遍写回 `products`。列表按价格过滤和排序都用它，不用每次去 SKU 表聚合。

图片最多 9 张，按顺序存，第一张就是主图。第 10 张返回 400。这一期只存图片地址，上传另做。

列表查询不带图文详情，只取卡片要的字段：id、名称、主图、最低价、划线价、销量。详情接口 `GetProduct` 返回本体、全部启用的 SKU、参数、图片，以及从 SKU 里汇总出的「每个规格有哪些值」，页面用它画规格选择器。图文详情单独走 `GetProductDetail`，因为它可能很大。

### 5. 商品状态、审核、日志、回收站

商品从录入到卖出，状态按下面这张表走。入口是 `ProductService` 上的 `SubmitProduct`、`ApproveProduct`、`RejectProduct`、`PublishProduct`、`UnpublishProduct`、`DeleteProduct`、`RestoreProduct`、`PurgeProduct`（新增），以及批量上下架 `BatchPublishProducts`、`BatchUnpublishProducts`（新增）。

| 动作 | 允许的当前状态 | 变成 |
|---|---|---|
| 提交审核 | 草稿、已驳回 | 待审核 |
| 审核通过 | 待审核 | 已通过 |
| 驳回 | 待审核 | 已驳回 |
| 上架 | 已通过、下架 | 在售 |
| 下架 | 在售 | 下架 |
| 删除 | 任意 | 进回收站（软删除） |
| 恢复 | 回收站里 | 下架 |
| 彻底删除 | 回收站里 | 物理删除 |

状态变化靠条件更新保证并发安全：更新语句带上「当前状态仍是预期状态」。两个人同时操作同一个商品，只有先到的那个成功，后到的看到状态已经变了，返回 `PRODUCT_STATUS_CONFLICT`（409），刷新后重来。

上架前 biz 逐项检查：分类是二级且在显示；品牌在显示；至少一张图；至少一个启用的 SKU；所有启用 SKU 的售价大于 0。有一项不满足就拒绝，错误信息里写明缺哪一项。批量上架逐个检查，能上的上，不能上的在响应里列出 id 和原因，不会因为一个失败而全部回退。

在售的商品不能改规格结构，也就是不能增删 SKU、不能改规格值，要先下架。改价和改文字信息可以在售时做。

每次提交、通过、驳回，都在同一事务里往 `product_audits` 写一条：动作、操作人、理由。改价、上下架、改规格、删除和恢复，往 `product_logs` 写一条：操作人、动作、改前和改后。两张表只增不改。

操作人 id 由调用方传入。运营账号做好之前，种子脚本和内部调用传固定值 `system`。

回收站里的商品买家完全看不到。彻底删除会连带删掉它的 SKU、参数、图片和详情。已经下过的订单不受影响，因为 order 存的是快照。

### 6. 买家列表与详情

买家不登录也能逛。入口是 gateway 的三条公开路由：`GET /v1/products`、`GET /v1/products/{id}`、`GET /v1/products/{id}/detail`（新增），转到 `ProductService` 的 `ListProducts`、`GetProduct`、`GetProductDetail`。

列表照 AGENTS 的约定用 AIP 风格：`filter`、`order_by`、`page_size`、`page_token`，解析用现有依赖 `go.einride.tech/aip`。允许过滤的字段是分类、品牌、价格区间；允许排序的是综合、最新、价格、销量。`page_size` 不传时默认 20，最大 50。下一页用上一页返回的 `page_token`，不用页码。

gateway 的公开列表路由固定加上「状态为在售」的条件，客户端传什么都覆盖掉。后台以后走另一条路由，能看全部状态。

按一级分类过滤时，biz 先查出它下面所有二级分类的 id，再用这组 id 去查商品。

详情页对下架商品照样返回，状态字段写着下架，页面据此显示「已下架」、灰掉购买按钮。在回收站里的商品返回 `PRODUCT_NOT_FOUND`（404）。

### 7. 关键词搜索

入口是 `ProductService.SearchProducts`（新增），gateway 公开路由 `GET /v1/products:search`（新增）。

这一期用 MySQL 在名称和关键词两个字段上做模糊匹配，只查在售商品，排序和分页规则同列表。README 后面计划接 Elasticsearch，到时候只换 data 层的实现，接口和页面都不用动。

### 8. 首页推荐位

首页有三个推荐位：新品、人气、专题。入口是 `RecommendationService`（新增）的后台增删改，以及买家读取 `ListRecommendations`，gateway 公开路由 `GET /v1/recommendations?slot=`（新增）。

每条推荐存在 `recommendations`：位置、商品 id、排序值、开始时间、结束时间。买家读取时，data 只取当前时间落在起止时间之间的，biz 再去掉已经不在售的商品，按排序值排。过期的推荐不用人去删，自然就不出现了。

### 9. 详情缓存

买家打开详情最频繁，所以只缓存 `GetProduct` 的结果。入口仍是 `GetProduct`，缓存在 data 层做，biz 感知不到。

读的时候先查 Redis，键是商品 id。命中就直接返回；没命中查 MySQL，写回 Redis 再返回。缓存 10 分钟过期。

任何改动商品、SKU、参数、图片的操作，事务提交后删掉这个商品的缓存，下一次读自然会重新填。删缓存失败只记日志，不让写操作失败，最坏情况是买家在过期前看到旧数据。列表不缓存。

### 10. 收藏

收藏需要知道是谁。这一块依赖 gateway 先改造：现在 `Verifier.Verify` 只判断令牌对不对，验完不告诉调用方用户是谁；`requireToken` 也只是放行。要改成验签后取出 `sub`，放进发给 product 的 gRPC 元数据里。元数据的键名是 `x-md-global-user-id`。gateway 只用验签得到的 id 设置这个键，客户端自己带的同名请求头一律不转发，防止冒充别人。

入口是 `FavoriteService`（新增）的 `AddFavorite`、`RemoveFavorite`、`ListFavorites`，gateway 路由都要登录。product 的 service 层从元数据里取用户 id，取不到返回 401。

收藏存在 `favorites`：用户 id、商品 id、收藏时间。重复收藏靠「用户＋商品」唯一索引兜底，已经收藏过就当成功返回，不报错。取消没收藏过的商品也当成功。列表按收藏时间倒序，每条带商品卡片；商品下架或进了回收站的，卡片标成「已失效」而不是消失。

### 11. 浏览记录

记浏览不需要单独的接口。买家打开详情时，如果带了有效令牌，gateway 照样把用户 id 放进元数据；`GetProduct` 发现有用户 id，就顺手记一条。令牌无效时 gateway 当作未登录处理，详情照样能看，不返回 401。

记录存在 `browse_histories`：用户 id、商品 id、最后浏览时间。同一商品再次浏览只更新时间。写完后删掉这个用户超出 100 条的最旧记录。记录失败只写日志，不影响详情返回。

查看和清空走 `BrowseHistoryService`（新增）的 `ListBrowseHistories`、`ClearBrowseHistories`，要登录，按最后浏览时间倒序。

### 12. 评价

评价要先有订单，所以放在 order 之后做。入口是 `ReviewService`（新增）的 `CreateReview`（要登录）和 `ListProductReviews`（公开，路由 `GET /v1/products/{id}/reviews`）。

写评价时，product 先调 order 的内部接口 `GetOrderItem`（新增）确认四件事：这个订单项属于当前用户、订单已完成、商品就是这个商品、还没评价过。任一条件不满足返回 `REVIEW_NOT_ALLOWED`（403）。评价写成功后，product 再调 order 的 `MarkOrderItemReviewed`（新增）把订单项标为已评价。

评价存在 `reviews`：订单项 id、用户 id、商品 id、SKU 规格、星级 1 到 5、文字、图片、是否匿名。一个订单项只能评一次，重复返回 `REVIEW_EXISTS`（409）。文字最多 500 字。

写评价时顺带调 user 取一次评价人的显示名存进表里：有昵称用昵称，没有就用「用户」加手机号后四位。读评价时不再调 user。读评价按时间倒序分页，匿名评价显示「匿名用户」。

### 13. 给 order 的快照接口

下单时 order 要一次拿到多个 SKU 的当前信息。入口是 `ProductService.BatchGetSkus`（新增），只在服务之间调用，gateway 不开放。

order 传一组 SKU id，product 返回每个 SKU 的商品名、规格、售价、主图，以及「现在能不能卖」。能卖的条件是：SKU 启用、商品在售、商品不在回收站。一次最多 50 个，超过返回 400。不存在的 id 在结果里标成不可售，而不是整个请求失败，这样 order 能准确告诉用户是哪一件出了问题。

结算前的检查 `CheckSellable`（新增）用同一套判断，只返回能不能卖，不返回价格。

### 14. 与 inventory 建库存记录

每个 SKU 在 inventory 里都应该有一条库存记录，否则永远卖不出去。这一块等 inventory 有接口后接上。

创建 SKU 的事务提交后，product 同步调用 inventory 的 `CreateStock`（新增）建一条数量为 0 的记录。调用失败不回滚 SKU，因为 SKU 已经存好了；补救放在上架检查里：上架时多一项检查，问 inventory 每个启用的 SKU 是否都有库存记录，缺的当场补建，补建失败就拒绝上架。建库存记录在 inventory 那边要做成可重复调用，同一个 SKU 建两次只会有一条。

以后有了 Kafka，改成 product 发「SKU 已创建」事件，inventory 订阅。上架检查那一项保留，当作兜底。

销量回写同样依赖 order：订单完成时，order 同步调 product 的 `IncreaseSales`（新增）给对应商品加销量，失败只记日志。接入 Kafka 后改成 product 订阅「订单已完成」事件。

### 15. 种子数据

后台还没有，商品要先靠脚本放进库。入口是新增的命令 `app/product/cmd/seed`，手动执行一次，通过 gRPC 调 product 的内部写接口。

脚本按「品牌 → 分类 → 属性模板 → 商品」的顺序写，然后把商品提交、通过、上架，最后配几条首页推荐。每一步先按名称查是否已存在，存在就跳过，所以重复执行不会多出数据。数据是 3 个一级分类，每个下 2 个二级分类，共 20 个商品。

### 16. gateway 路由

gateway 新增一个上游客户端 `NewProductClient`（新增），写法照现有 `app/gateway/internal/client/user.go` 的 `NewUserClient`，地址配置在 `client.product.addr`，默认 `127.0.0.1:9001`。路由照现有 `app/gateway/internal/server/http.go` 的 `userProxy`，新增 `productProxy`（新增）。

| 路由 | 登录 | 期数 |
|---|---|---|
| `GET /v1/categories`、`GET /v1/brands` | 否 | 1 |
| `GET /v1/products`、`GET /v1/products/{id}`、`GET /v1/products/{id}/detail` | 否（带令牌时记浏览） | 1 |
| `GET /v1/products:search`、`GET /v1/recommendations` | 否 | 4 |
| 收藏、浏览记录 | 是 | 5 |
| `GET /v1/products/{id}/reviews` | 否 | 7 |
| 写评价 | 是 | 7 |
| 品牌、分类、属性、商品、推荐位的写接口 | 后台账号 | 6 |

后台账号放在单独的 admin 服务里，用自己的密钥签后台令牌，gateway 按角色放行，写法见 `docs/tech-implementation.md` 第 11 章。后台路由统一在 `/v1/admin/` 下。在那之前，写接口只能在服务内部调用。

### 17. 前端

**用户端 `frontend/web`**

- 第 1 期：首页商品列表、商品详情。
- 第 2 期：详情页的规格选择，选中后价格和图片跟着换。
- 第 4 期：分类页、搜索结果页（筛选和排序）、首页三个推荐位。
- 第 5 期：我的收藏、浏览记录。
- 第 7 期：评价列表和写评价。

页面只请求 gateway。价格统一在一个地方从分换算成元，页面不自己除 100。

**管理台 `frontend/admin-web`**，第 6 期：品牌、分类、属性模板、商品编辑（分四步：基本信息、规格与价格、参数、图片与详情）、审核、回收站、操作日志、推荐位。

## 分期

| 期 | 内容 | 对应实现块 |
|---|---|---|
| 1 | 底座、品牌、分类、商品与 SKU、买家列表与详情、种子数据、gateway 公开路由、web 首页和详情 | 0、1、2、4、6、15、16 |
| 2 | 属性模板与属性、详情页规格选择 | 3 |
| 3 | 状态机、上架校验、审核、日志、回收站 | 5 |
| 4 | 过滤排序完善、搜索、推荐位、详情缓存、分类页和搜索页 | 6、7、8、9 |
| 5 | gateway 传用户身份、收藏、浏览记录 | 10、11 |
| 6 | 后台账号、admin-web 全部页面、后台路由 | 16、17 |
| 7 | 快照接口、建库存、销量回写、评价 | 12、13、14 |

第 1 期的商品没有状态机。种子数据直接写成在售，第 3 期再把状态流转补上。第 2 期之前，SKU 的规格值不校验模板。

每期完成的标准：三层单测通过，接口经 gateway 调得通，对应页面能用。

## 不做什么

- 不存库存数量，不做库存扣减。
- 不做阶梯价、满减、会员价、优惠券。
- 不做图片上传，只存地址。
- 不接 Elasticsearch，搜索先用 MySQL。
- 不做 Kafka 事件，先同步调用。
