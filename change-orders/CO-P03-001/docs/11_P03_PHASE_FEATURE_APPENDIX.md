<!-- CO_P03_001_FEATURE_APPENDIX -->
# P03 变更单 CO-P03-001：多轮上下文、产品体验与正式运营补充

> 本附录是 `P03_CONVERSATIONS_STREAMING_CHAT_AND_MESSAGES.md` 的正式组成部分。导入后，P03 Feature 总数由 35 增加为 55；P03-W06、P03-W07、P03-W08 完成前不得关闭 P03。

## 新增实施顺序

1. `P03-W06`：多轮上下文、延迟创建会话、自动标题与标题锁定。
2. `P03-W07`：首页、会话页、输入区、模型选择和正式运营文案重构。
3. `P03-W08`：共享事件流、无状态 AI Runtime、上下文诊断和多实例回归。

## 新增 Feature 逐项合同

### P03-036 — 建立规范化会话事实源

- **产品表面/页面或菜单**：`Backend` / `Conversation Service`。
- **用户或系统动作**：保存同一 conversation_id 下的完整可见消息与顺序。
- **必须覆盖状态**：`idle|success|conflict|error`。不得只实现成功态。
- **接口或事件**：`POST|GET` `/internal/v1/conversations/{conversationId}/context`；负责服务：`Conversation Service`。
- **数据实体**：`conversations|conversation_branches|messages|message_parts`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`对话与内容/会话详情`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：建立规范化会话事实源必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-036-UNIT|P03-036-API|P03-036-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-037 — 按当前分支加载历史上下文

- **产品表面/页面或菜单**：`Backend` / `Context Compiler`。
- **用户或系统动作**：加载最近完整轮次与当前分支。
- **必须覆盖状态**：`loading|success|empty|error`。不得只实现成功态。
- **接口或事件**：`GET` `/internal/v1/conversations/{conversationId}/context-items`；负责服务：`AI Runtime`。
- **数据实体**：`conversation_branches|messages|message_parts`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/AI运行`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：按当前分支加载历史上下文必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-037-UNIT|P03-037-API|P03-037-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-038 — 按模型能力执行 Token 预算与裁剪

- **产品表面/页面或菜单**：`Backend` / `Context Compiler`。
- **用户或系统动作**：计算可用输入预算并保留完整轮次。
- **必须覆盖状态**：`success|compacted|error`。不得只实现成功态。
- **接口或事件**：`POST` `/internal/v1/context/build`；负责服务：`AI Runtime`。
- **数据实体**：`context_builds|context_build_items|model_capabilities`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`模型与供应商/模型目录`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：按模型能力执行 Token 预算与裁剪必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-038-UNIT|P03-038-API|P03-038-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-039 — 生成可追溯会话摘要

- **产品表面/页面或菜单**：`Backend` / `Compaction Worker`。
- **用户或系统动作**：压缩较早消息但不删除原始记录。
- **必须覆盖状态**：`queued|running|success|error`。不得只实现成功态。
- **接口或事件**：`POST` `/internal/v1/conversations/{conversationId}/compact`；负责服务：`Worker`。
- **数据实体**：`conversation_summaries|context_compactions|jobs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`对话与内容/上下文策略`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `conversation_compaction`。
- **验收标准**：生成可追溯会话摘要必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-039-UNIT|P03-039-API|P03-039-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-040 — 切换模型时重新编译中立上下文

- **产品表面/页面或菜单**：`Backend` / `Provider Adapter`。
- **用户或系统动作**：使用同一会话事实重建目标模型输入。
- **必须覆盖状态**：`success|fallback|error`。不得只实现成功态。
- **接口或事件**：`POST` `/internal/v1/context/recompile`；负责服务：`AI Runtime`。
- **数据实体**：`context_builds|provider_conversation_states`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`模型与供应商/路由`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：切换模型时重新编译中立上下文必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-040-UNIT|P03-040-API|P03-040-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-041 — Provider continuation 失败自动回退

- **产品表面/页面或菜单**：`Backend` / `Provider Adapter`。
- **用户或系统动作**：上游续聊状态失效时本地重建。
- **必须覆盖状态**：`success|fallback|error`。不得只实现成功态。
- **接口或事件**：`POST` `/internal/v1/provider-state/fallback`；负责服务：`AI Runtime`。
- **数据实体**：`provider_conversation_states|context_builds`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/AI运行`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：Provider continuation 失败自动回退必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-041-UNIT|P03-041-API|P03-041-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-042 — 会话级并发锁与消息幂等

- **产品表面/页面或菜单**：`Backend` / `Conversation Run`。
- **用户或系统动作**：防止重复消息、重复运行和乱序。
- **必须覆盖状态**：`success|duplicate|conflict|error`。不得只实现成功态。
- **接口或事件**：`POST` `/mobile/v1/conversations/{conversationId}/runs`；负责服务：`AI Runtime`。
- **数据实体**：`message_runs|idempotency_records|conversation_locks`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/AI运行`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：会话级并发锁与消息幂等必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-042-UNIT|P03-042-API|P03-042-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-043 — 延迟创建正式会话

- **产品表面/页面或菜单**：`Android|Backend` / `首页与空白会话`。
- **用户或系统动作**：第一条消息发送前只保留本地草稿。
- **必须覆盖状态**：`draft|submitting|success|error`。不得只实现成功态。
- **接口或事件**：`POST` `/mobile/v1/conversations/from-first-message`；负责服务：`Conversation Service`。
- **数据实体**：`conversations|messages|drafts`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`对话与内容/会话列表`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：延迟创建正式会话必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-043-UNIT|P03-043-API|P03-043-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-044 — 第一条有效消息后自动生成标题

- **产品表面/页面或菜单**：`Android|Backend` / `会话标题`。
- **用户或系统动作**：先本地临时标题，再异步优化。
- **必须覆盖状态**：`temporary|generating|success|fallback`。不得只实现成功态。
- **接口或事件**：`POST` `/internal/v1/conversations/{conversationId}/title`；负责服务：`AI Runtime`。
- **数据实体**：`conversations|jobs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`对话与内容/标题模型`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `conversation_title`。
- **验收标准**：第一条有效消息后自动生成标题必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-044-UNIT|P03-044-API|P03-044-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-045 — 用户重命名后锁定标题

- **产品表面/页面或菜单**：`Android|Backend` / `会话菜单`。
- **用户或系统动作**：手动标题不得被自动任务覆盖。
- **必须覆盖状态**：`default|submitting|success|error`。不得只实现成功态。
- **接口或事件**：`PATCH` `/mobile/v1/conversations/{conversationId}/title`；负责服务：`Conversation Service`。
- **数据实体**：`conversations`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`对话与内容/会话详情`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：用户重命名后锁定标题必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-045-UNIT|P03-045-API|P03-045-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-046 — 后台查看上下文构建诊断

- **产品表面/页面或菜单**：`Admin` / `可观测性/AI运行`。
- **用户或系统动作**：查看本次上下文组成、裁剪与回退。
- **必须覆盖状态**：`loading|populated|not_found|error`。不得只实现成功态。
- **接口或事件**：`GET` `/admin/v1/ai-runs/{runId}/context-debug`；负责服务：`Observability Service`。
- **数据实体**：`context_builds|context_build_items|message_runs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/AI运行`；权限：`admin.ai_runs.read`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：后台查看上下文构建诊断必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-046-UNIT|P03-046-API|P03-046-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-047 — 建立用户可见状态文案映射

- **产品表面/页面或菜单**：`Android` / `会话页`。
- **用户或系统动作**：把内部状态映射为普通用户语言。
- **必须覆盖状态**：`thinking|streaming|tool_running|reconnecting|error`。不得只实现成功态。
- **接口或事件**：`GET` `config://consumer-copy-catalog`；负责服务：`Mobile BFF`。
- **数据实体**：`system_configs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`内容运营/客户端文案`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：建立用户可见状态文案映射必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-047-UNIT|P03-047-API|P03-047-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-048 — 将首页重构为对话优先入口

- **产品表面/页面或菜单**：`Android` / `首页`。
- **用户或系统动作**：直接输入并开始对话。
- **必须覆盖状态**：`loading|populated|empty|offline|error`。不得只实现成功态。
- **接口或事件**：`GET` `/mobile/v1/home`；负责服务：`Core API`。
- **数据实体**：`conversations|home_config`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`内容运营/首页配置`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：将首页重构为对话优先入口必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-048-UNIT|P03-048-API|P03-048-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-049 — 把文件图片PPT入口收敛到输入区工具面板

- **产品表面/页面或菜单**：`Android` / `会话输入区`。
- **用户或系统动作**：从加号工具面板选择能力。
- **必须覆盖状态**：`default|open|selected|disabled`。不得只实现成功态。
- **接口或事件**：`GET` `config://composer-tool-tray`；负责服务：`Mobile BFF`。
- **数据实体**：`feature_flags|home_config`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`内容运营/工具入口`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：把文件图片PPT入口收敛到输入区工具面板必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-049-UNIT|P03-049-API|P03-049-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-050 — 模型和回答方式改为可选高级入口

- **产品表面/页面或菜单**：`Android` / `会话输入区`。
- **用户或系统动作**：默认自动，用户需要时手动选择。
- **必须覆盖状态**：`auto|selected|disabled|degraded`。不得只实现成功态。
- **接口或事件**：`GET` `/mobile/v1/models`；负责服务：`Model Catalog`。
- **数据实体**：`models|reasoning_profiles`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`模型与供应商/模型目录`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：模型和回答方式改为可选高级入口必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-050-UNIT|P03-050-API|P03-050-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-051 — 普通用户界面禁止技术性调试文案

- **产品表面/页面或菜单**：`Android` / `全局消费者界面`。
- **用户或系统动作**：展示正式运营文案。
- **必须覆盖状态**：`pass|violation`。不得只实现成功态。
- **接口或事件**：`GET` `policy://consumer-copy`；负责服务：`Mobile BFF`。
- **数据实体**：`system_configs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`内容运营/客户端文案`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：普通用户界面禁止技术性调试文案必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-051-UNIT|P03-051-API|P03-051-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-052 — 前后端禁止 Demo 与伪完成实现

- **产品表面/页面或菜单**：`All` / `全局`。
- **用户或系统动作**：使用真实数据库、接口、权限、日志和错误处理。
- **必须覆盖状态**：`pass|violation`。不得只实现成功态。
- **接口或事件**：`GET` `policy://production-readiness`；负责服务：`Core API`。
- **数据实体**：`audit_logs|system_configs`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`系统运营/发布门禁`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：前后端禁止 Demo 与伪完成实现必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-052-UNIT|P03-052-API|P03-052-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-053 — 运行事件进入共享事件流并支持游标恢复

- **产品表面/页面或菜单**：`Backend` / `AI Runtime`。
- **用户或系统动作**：任意实例均可恢复 SSE。
- **必须覆盖状态**：`running|reconnecting|completed|error`。不得只实现成功态。
- **接口或事件**：`GET` `/mobile/v1/runs/{runId}/events`；负责服务：`AI Runtime`。
- **数据实体**：`run_events|redis_streams`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/AI运行`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：运行事件进入共享事件流并支持游标恢复必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-053-UNIT|P03-053-API|P03-053-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-054 — AI Runtime 保持无状态并支持水平扩容

- **产品表面/页面或菜单**：`Backend` / `AI Runtime`。
- **用户或系统动作**：多实例处理同一类请求。
- **必须覆盖状态**：`healthy|degraded|error`。不得只实现成功态。
- **接口或事件**：`GET` `/internal/v1/health/ai-runtime`；负责服务：`AI Runtime`。
- **数据实体**：`service_instances|health_checks`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/服务健康`；权限：`authenticated_user`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：AI Runtime 保持无状态并支持水平扩容必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-054-UNIT|P03-054-API|P03-054-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。

### P03-055 — 执行多实例上下文与流式恢复测试

- **产品表面/页面或菜单**：`CI|Backend` / `GitHub Actions`。
- **用户或系统动作**：验证跨实例连续对话和恢复。
- **必须覆盖状态**：`running|pass|fail`。不得只实现成功态。
- **接口或事件**：`POST` `ci://p03-distributed-chat`；负责服务：`CI`。
- **数据实体**：`test_runs|run_events`。必须具有迁移、索引、所有权、幂等或状态约束。
- **管理后台控制**：`可观测性/测试报告`；权限：`ci`。
- **计费与异步任务**：计费 `none`；任务 `none`。
- **验收标准**：执行多实例上下文与流式恢复测试必须以正式运营标准实现：真实数据持久化、稳定错误码、幂等与权限校验齐全；普通用户界面不得出现 demo、debug、transport、provider 或 raw state 文案；相关自动化测试全部通过。
- **自动测试 ID**：`P03-055-UNIT|P03-055-API|P03-055-E2E`。
- **实施补充**：遵循 CO-P03-001 的上下文、UI、文案、分布式与生产就绪合同；不得静默偏离。
