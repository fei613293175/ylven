# 22 API 合同、事件、版本与兼容策略

## 1. 三类入口

- Mobile/Admin API：面向 Android 和管理后台，使用 YLVEN 用户或管理员 Token。
- Developer API：面向外部 API Key，独立限流、预算、审计和错误文档。
- 内部事件/Worker 合同：通过 NATS JetStream 或任务表传递，必须有 Schema 版本和幂等键。

Android 不能调用 Admin 或 Sub2API 管理接口，外部开发者不能依赖内部实体字段。

## 2. OpenAPI

`contracts/openapi-skeleton.yaml` 是接口库存骨架，不是可以长期保留通用 200 响应的最终合同。每阶段实现时应补充请求/响应 Schema、认证、分页、幂等头、错误码、示例和安全限制，并保留 `x-feature-ids`。共享端点可关联多个 Feature，但不能让后生成内容覆盖前一功能归属。

## 3. 版本兼容

公共 API 使用明确版本路径或版本头，模型别名与 API 版本分开管理。破坏性变更必须提供新版本、弃用日期、迁移说明、遥测和后台通知。Android Mobile API 可通过最低支持版本和 feature flag 演进，但服务端不能在旧 App 仍受支持时删除必需字段。新增响应字段应向后兼容；枚举新增需提供 unknown fallback。

## 4. 事件合同

每个事件包含 `event_id`、`event_type`、`schema_version`、`occurred_at`、`producer`、`aggregate_id`、`correlation_id`、`causation_id` 和 payload。消费者必须幂等，不能假定只投递一次；Schema 演进遵守新增可选字段优先，破坏性变化创建新版本。

## 5. 分页、排序和搜索

高增长表优先使用稳定游标分页；后台复杂报表可使用受限 offset。排序字段必须白名单，搜索参数限制长度和复杂度。所有列表按用户/组织/权限过滤后再分页，不能先分页再过滤导致跨账号泄漏。

## 6. 兼容 Sub2API 与官方 API

Provider Adapter 将 YLVEN 统一 Run/Message/Attachment/Tool 格式转换为 Sub2API 或官方协议。公共 YLVEN API 不透传上游不可稳定保证的字段；原始响应只在受控日志或 `provider_metadata` 中按隐私策略保存。
