# 07 后端领域模块详细规格

## 1. Identity 与认证

负责邮箱注册、邮箱验证码登录、密码哈希、安全挑战票据、访问 Token、刷新 Token、设备会话、管理员认证、OIDC Provider 和 Sub2API SSO。不得包含钱包或模型路由逻辑。

## 2. Users 与 Profiles

负责用户基础状态、数字 UID、头像、用户名、偏好、语言和时区。邮箱属于身份模块，资料模块只能读取经过授权的规范化邮箱。

## 3. Conversations

负责会话、分支、消息、消息部件、Run 元数据、摘要、归档、删除和分享。AI Runtime 负责执行 Run，但最终结果通过 Conversation 接口持久化。

## 4. Models 与 Routing

负责供应商、模型目录、别名、能力、推理档位、价格版本、路由策略、通道健康和灰度。所有 App 模型列表均来自该模块。

## 5. Providers

包含 `Sub2APIAdapter`、`OpenAIAdapter`、`AnthropicAdapter`、`XAIAdapter`。统一实现文本流、视觉、文件、工具、图片和取消等能力；不支持的能力返回标准化错误。

## 6. Context Compiler

将 YLVEN 结构化消息转换为目标供应商请求：

- 选择有效历史；
- 处理会话摘要；
- 转换文本、图片、文件和工具结果；
- 计算 Token 预算；
- 去除供应商专用内部字段；
- 不将一个供应商的隐藏推理内容传给另一个供应商。

## 7. Projects 与 Knowledge

项目管理长期上下文。Knowledge 模块负责文件分块、Embedding、向量索引、全文检索、权限过滤和引用。首期使用 PostgreSQL FTS + pgvector。

## 8. Files

负责上传会话、对象 Key、MIME、哈希、版本、扫描、解析、缩略图、权限和删除。所有上传先进入隔离区，完成校验和扫描后才能用于 AI。

## 9. Artifacts

负责图片、PPT、文档等作品及其版本、素材依赖、导出和下载。作品与消息可以互相引用，但作品生命周期不依赖聊天消息是否仍存在。

## 10. Jobs

负责长任务状态机：`QUEUED -> RUNNING -> SUCCEEDED/FAILED/CANCELLED`。任务必须记录进度、尝试、错误码、结果、取消请求和事件流。重试必须幂等。

## 11. Wallet 与 Ledger

负责资金账户、额度桶、不可变交易和余额投影。任何余额变化均由账本交易产生；后台禁止直接执行 `UPDATE balance`。

## 12. Products、Plans、Subscriptions、Entitlements

- Product：可销售产品。
- Plan：套餐周期和价格组合。
- Subscription：用户订阅状态。
- Entitlement：具体功能权限或限额。
- Credit Bucket：套餐、赠送或充值额度及过期时间。

## 13. Payments 与 Orders

负责订单、支付渠道适配、回调验签、退款和权益发放。支付成功与额度同步成功必须分开记录；同步失败进入 `CREDIT_PENDING` 自动重试。

## 14. Developer

负责开发者状态、应用、API Key、策略、IP 白名单、预算、Webhook、调用日志和 SDK 文档版本。Developer Gateway 使用该模块的缓存投影进行快速鉴权。

## 15. CMS 与 Feature Flags

负责首页快捷入口、发现卡片、公告、模板、服务状态、灰度和最低版本。客户端只能渲染预定义组件，不执行服务端脚本。

## 16. Notifications

负责邮件、App 推送预留、系统消息和任务完成通知。通知模板带版本和变量白名单，防止任意模板注入。

## 17. Operations、Audit 与 Incidents

负责健康探测、事故、审计、配置版本、备份记录、发布版本和运维任务。审计日志不能由普通管理员删除或修改。
