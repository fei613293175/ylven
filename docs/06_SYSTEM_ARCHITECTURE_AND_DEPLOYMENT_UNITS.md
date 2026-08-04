# 06 系统总体架构与部署单元

## 1. 架构总览

```text
Android App / Admin Web / Developer Web / External API Clients
                         |
                  Cloudflare WAF/CDN
                         |
                 NGINX Edge Gateway
                         |
       +-----------------+------------------+
       |                 |                  |
  Mobile/Core API    Developer Gateway   Admin API
       |                 |                  |
       +-----------------+------------------+
                         |
                YLVEN Control Plane
                         |
                 YLVEN AI Runtime
                         |
          +--------------+---------------+
          |                              |
    Sub2API Adapter                Official API Adapters
          |                              |
       Sub2API                OpenAI / Anthropic / xAI

PostgreSQL | PgBouncer | Redis | NATS JetStream | Cloudflare R2
Workers: file parsing | image | PPT | notifications | billing sync
Observability: OpenTelemetry | Prometheus | Loki | Tempo | Grafana
```

## 2. 首期独立部署单元

### 2.1 `ylven-core-api`

模块化单体，负责用户、认证、设备、会话元数据、项目、文件元数据、作品、套餐、钱包、订单、配置、后台接口和移动 BFF。保持无状态，允许水平扩容。

### 2.2 `ylven-ai-runtime`

负责聊天 Run、流式 SSE、上下文编译、模型路由、推理参数映射、工具编排、取消、断线恢复、用量事件和上游熔断。该服务与普通后台查询分离，避免复杂管理查询影响长连接。

### 2.3 `ylven-developer-gateway`

负责公共 API 鉴权、预算、限流、OpenAI 兼容、原生 API、计量和开发者日志。不得复用 Android JWT 作为外部 API 凭据。

### 2.4 `ylven-worker`

以不同队列和 Worker Pool 处理：文件解析、病毒扫描、知识索引、图片生成/编辑、PPT 生成/渲染、导出、通知、余额同步和对账。支付/余额任务与图片/PPT 任务分开队列和优先级。

### 2.5 `ylven-scheduler`

执行额度重置、模型能力探测、过期清理、账本聚合、同步重试、备份检查和告警评估。使用分布式锁，确保同一计划任务只有一个有效执行者。

### 2.6 Web 前端

- `ylven-admin-web`
- `ylven-developer-web`
- `ylven-download-web`
- `ylven-docs-web`

可以位于 pnpm monorepo，共用组件与 API 类型，但独立构建和部署。

## 3. 核心技术基线

- Backend：Go 1.26.5、Gin、pgx/v5、sqlc、Goose 数据库迁移。
- Database：PostgreSQL 18.4，开启必要扩展 `citext`、`pgcrypto`、`vector`。
- Connection Pool：PgBouncer，事务池模式；需要会话特性的查询必须明确处理。
- Cache/Rate Limit：Redis 8 系列。
- Event Bus：NATS Server 2.14.x + JetStream。
- Storage：Cloudflare R2，S3 API。
- Reverse Proxy：NGINX，SSE 路由关闭代理缓冲。
- Observability：OpenTelemetry、Prometheus、Loki、Tempo、Grafana、Alertmanager。
- Admin/Developer：Vue 3 + TypeScript + Vite + Ant Design Vue。
- Deployment：Docker Compose 首期，后期可迁移 Kubernetes，不在首期引入 Kubernetes 复杂度。

## 4. 模块化单体目录建议

```text
backend/
  cmd/
    core-api/
    ai-runtime/
    developer-gateway/
    worker/
    scheduler/
  internal/
    identity/
    users/
    devices/
    conversations/
    models/
    routing/
    providers/
    projects/
    files/
    knowledge/
    artifacts/
    jobs/
    billing/
    wallet/
    subscriptions/
    developer/
    cms/
    notifications/
    operations/
    audit/
  db/
    migrations/
    queries/
    generated/
  pkg/
```

模块之间通过公开接口和领域事件协作，禁止跨模块直接修改对方表。数据库可以共用一个集群，但 schema/查询归属明确。

## 5. 控制平面与数据平面

### 控制平面

用户、套餐、价格、模型目录、路由规则、Feature Flag、后台配置和审计。

### 数据平面

聊天流式请求、公共 API、图片/PPT 任务、文件处理和实时计量。

数据平面必须优先保证低延迟和可用性；控制平面统计查询不得使用数据平面关键连接池。

## 6. 事件与一致性

- 使用 PostgreSQL 事务 Outbox，业务事务与待发布事件同事务提交。
- Outbox Publisher 将事件发布到 NATS JetStream。
- 消费者按事件 ID 幂等处理。
- 支付、账本、余额和权益使用强一致事务；搜索索引、统计和通知允许最终一致。
- 不通过多个服务共同修改同一余额字段实现分布式事务。

## 7. 扩容策略

- Core API、AI Runtime、Developer Gateway 均无状态，多实例部署。
- AI Runtime 按并发流式连接和上游供应商连接池扩容。
- Worker 按队列积压和任务类型独立扩容。
- PostgreSQL 首期主库 + 备份；达到指标后增加只读副本用于统计。
- Redis 首期持久化单实例或主从；生产商业化前至少完成高可用方案。
- R2 保存大对象，应用实例不使用本地磁盘作为永久存储。
