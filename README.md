# supermarket

开一家超市：自研电商微服务。复刻 [mall](https://github.com/macrozheng/mall) 的功能需求边界，架构自研并全面超出蓝本。

个人全栈学习项目，2026-09 启动，共 18 周。终点形态：

- 全量部署 Kubernetes
- 秒杀零超卖（Redis Lua 预扣）
- 压测报告与性能调优
- AI 客服（RAG）

## 架构

> 架构图待补（Excalidraw），当前为规划形态。

调用关系：客户端 → gateway → 各业务服务；服务间同步调用走 gRPC，异步事件走 Kafka；MySQL 按服务分库，Redis 做缓存与预扣库存。

## 服务

| 服务 | 职责 | 状态 |
|---|---|---|
| gateway | API 网关：路由、鉴权、限流 | 未开始 |
| user | 用户：注册登录、资料、地址 | 未开始 |
| product | 商品：SPU/SKU、分类、检索 | 未开始 |
| order | 订单：状态机、延迟取消 | 未开始 |
| inventory | 库存：预扣、防超卖 | 未开始 |
| payment | 支付（mock）：幂等、对账 | 未开始 |
| notification | 通知：Kafka 消费、多渠道下发 | 未开始 |
| admin-web | 管理台（React + TS） | 未开始 |

## 技术栈

- 后端：Go · [kratos](https://github.com/go-kratos/kratos) · MySQL · Redis · Kafka
- 前端：TypeScript · React · Vite
- 基建：Docker · Kubernetes · GitHub Actions

## 路线

| 阶段 | 内容 | 周期 |
|---|---|---|
| 1 · 项目开发 | 仓库/CI/compose，下单全链路本地跑通 | 约 8 周（当前） |
| 2 · 运维与上线 | CI/CD、K8s 集群、线上排查 | 约 4 周 |
| 3 · 数据库深入 | EXPLAIN 实战、索引与慢查询优化 | 约 2 周 |
| 4 · 高可用与性能 | 缓存三大问题、限流熔断、压测 | 约 2 周 |
| 5 · AI 与收尾 | AI 客服（RAG）、架构文档 | 约 2 周 |

## 本地开发

依赖：Go 1.26+、Docker、Node 26+（管理台阶段）。

各服务落地后在此补充：compose 一键起依赖（MySQL/Redis/Kafka）、服务启动顺序、种子数据。
