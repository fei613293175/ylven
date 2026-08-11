# 数据库、API、SSE 与自动标题详细规格

## 数据对象

### conversations

新增/确认字段：`id`、`user_id`、`title`、`title_source(AUTO_TEMP/AUTO_FINAL/USER)`、`title_locked`、`active_branch_id`、`summary_through_message_id`、`created_at`、`updated_at`、`archived_at`。

### messages

必须包含：`id`、`conversation_id`、`branch_id`、`sequence`、`parent_message_id`、`role`、`status`、`created_at`、`completed_at`。`sequence` 在同一会话分支内严格单调。

### message_parts

支持 `TEXT`、`IMAGE`、`FILE`、`AUDIO`、`TOOL_CALL`、`TOOL_RESULT`、`CITATION`、`ARTIFACT`。不要把所有内容压成一列 Markdown。

### context_builds / context_build_items

每次模型调用记录实际使用的指令、摘要、历史消息、附件、Token 预算、排除原因、上下文哈希和回退方式。不得记录模型隐藏思维链。

### conversation_summaries

保存来源消息区间、结构化状态、自然语言摘要、模型、版本和校验时间。摘要不会删除原始消息。

### provider_conversation_states

保存上游 continuation ID、有效期和状态，仅用于加速；失效时自动回退本地重建。

## 第一条消息事务

1. 客户端持有本地 `draft_session_id`；
2. 用户发送第一条有效消息；
3. 后端在同一事务中创建 conversation、默认 branch 和第一条 user message；
4. 返回正式 `conversation_id` 与 `run_id`；
5. 立即生成本地可读临时标题；
6. 首轮回答完成后异步生成最终标题；
7. 失败时保留用户输入并允许幂等重试，不产生多个空会话。

## 自动标题

- 临时标题：从第一条有效消息截取核心名词，立即展示；
- 最终标题：6～18 个中文字符，不加引号/句号，不使用“新对话”；
- “你好”“测试”等低信息消息等待下一条有效问题再优化；
- 用户手动重命名后 `title_locked=true`，自动任务不得覆盖。

## SSE 事件

内部事件可使用技术代码，但移动端展示必须经过文案映射。事件至少包含 `event_id`、`run_id`、`sequence`、`created_at`、`type`、`payload`。重连使用 `Last-Event-ID` 或等价游标；任何实例都能从共享流恢复。
