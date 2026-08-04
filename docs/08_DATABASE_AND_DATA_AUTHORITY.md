# 08 数据库与数据权威设计

## 1. 数据库原则

- PostgreSQL 18.4 是 YLVEN 核心关系数据库。
- 所有主键使用 UUIDv7 或等价时间有序 UUID；对外数字 UID 单独生成。
- 时间统一保存为 UTC `timestamptz`，前端按用户时区显示。
- 金额和额度使用整数最小单位或 `numeric`，禁止浮点数记账。
- 邮箱使用 `citext` 唯一索引；不对 Gmail 点号和加号做未经用户授权的特殊归一化。
- 大表从设计时保留分区策略，但不为低数据量提前创建复杂分片。

## 2. 身份与用户

核心表：

- `users`
- `user_profiles`
- `auth_identities`
- `email_verification_transactions`
- `email_otp_challenges`
- `security_challenge_tickets`
- `user_devices`
- `auth_sessions`
- `refresh_tokens`
- `roles`
- `permissions`
- `role_bindings`
- `gateway_user_links`

`gateway_user_links` 以 YLVEN UUID 为主，保存 Sub2API user_id、OIDC subject、状态和最后同步时间。邮箱不是永久绑定键。

## 3. 会话与消息

- `conversations`
- `conversation_branches`
- `messages`
- `message_parts`
- `message_runs`
- `run_events`
- `tool_calls`
- `citations`
- `conversation_summaries`
- `message_feedback`

`message_parts.part_type` 至少支持：`TEXT`、`IMAGE`、`FILE`、`AUDIO`、`TOOL_CALL`、`TOOL_RESULT`、`CITATION`、`ARTIFACT`、`ERROR`。

Run 保存：请求模型、实际模型、供应商、推理档位、路由通道、状态、首字延迟、总耗时、Token、成本、价格版本、上游 request_id 和追踪 ID。

## 4. 模型和路由

- `providers`
- `provider_channels`
- `models`
- `model_aliases`
- `model_capabilities`
- `reasoning_profiles`
- `routing_policies`
- `route_targets`
- `model_health_probes`
- `model_price_versions`

配置能力与验证能力分开保存。App 只展示 `verified=true` 且未过期的能力。

## 5. 项目、文件和知识库

- `projects`
- `project_members`
- `project_instructions`
- `attachments`
- `file_versions`
- `file_extractions`
- `project_files`
- `knowledge_documents`
- `knowledge_chunks`
- `knowledge_indexes`

对象内容存 R2，数据库保存对象 Key、大小、MIME、哈希、扫描状态、权限、生命周期和引用计数。

## 6. 作品和任务

- `artifacts`
- `artifact_versions`
- `artifact_assets`
- `jobs`
- `job_events`
- `job_attempts`
- `job_results`
- `dead_letter_jobs`
- `templates`
- `template_versions`

任务表不保存大型结果二进制，只保存 R2 对象引用和结构化元数据。

## 7. 商业化和账本

- `products`
- `plans`
- `prices`
- `subscriptions`
- `entitlement_grants`
- `orders`
- `payments`
- `refunds`
- `wallet_accounts`
- `ledger_transactions`
- `ledger_entries`
- `balance_projections`
- `credit_buckets`
- `usage_events`
- `usage_price_snapshots`
- `usage_aggregates`
- `quota_counters`
- `gateway_sync_jobs`
- `reconciliation_runs`
- `reconciliation_items`

账本交易和分录不可修改；纠错通过反向分录。`balance_projections` 是可重建投影，不是财务事实。

## 8. 开发者平台

- `developer_profiles`
- `developer_apps`
- `api_keys`
- `api_key_policies`
- `api_request_logs`
- `ip_allowlists`
- `webhooks`
- `webhook_deliveries`

API 调用日志达到规模后按月分区；详细日志按保留策略归档至对象存储或分析库。

## 9. 运营和安全

- `feature_flags`
- `discover_cards`
- `announcements`
- `notifications`
- `app_versions`
- `audit_logs`
- `security_events`
- `incidents`
- `system_configs`
- `outbox_events`
- `inbox_deduplication`

## 10. 索引和查询规则

- 所有外键建立与查询模式匹配的索引。
- 会话列表使用 `(user_id, updated_at desc, id)` 游标分页。
- 消息使用 `(conversation_id, created_at, id)`。
- 任务使用 `(status, priority, available_at)`。
- 使用量使用 `(user_id, occurred_at)` 并按月分区。
- 审计使用 `(actor_id, occurred_at)`、`(resource_type, resource_id)`。
- 禁止在高频请求中使用无界 `OFFSET` 深分页。

## 11. 备份和恢复

- PostgreSQL 每日全量/增量策略并保留 WAL，支持时间点恢复。
- R2 关键桶启用版本/生命周期策略或等价备份。
- 至少每季度执行一次恢复演练；“备份成功”不能仅凭任务退出码判断，必须验证可恢复性。
