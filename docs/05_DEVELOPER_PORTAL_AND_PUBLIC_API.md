# 05 开发者中心与公共 API 规格

## 1. 产品定位

YLVEN 开发者中心是面向注册用户的独立 Web 产品。用户使用与 Android App 相同的 YLVEN 账号登录，查看统一套餐、余额和使用量，并创建外部 API Key。外部开发者只能接触 YLVEN 域名、YLVEN Key、YLVEN 模型别名和 YLVEN 错误码，不能直接依赖 Sub2API 地址或内部字段。

## 2. Web 菜单

- 概览；
- API Key；
- API Playground；
- 模型与能力；
- 模型价格；
- 调用日志；
- 用量统计；
- 预算与告警；
- 充值与账单；
- 套餐与权益；
- IP 白名单；
- Webhook；
- 文档与 SDK；
- 服务状态；
- 账号与安全。

## 3. API Key

- 前缀采用 `ylv_live_` 和 `ylv_test_`。
- 完整 Key 只在创建时展示一次；数据库只保存不可逆哈希、前缀和末尾提示。
- 支持名称、状态、过期时间、IP 白名单、模型白名单、请求来源、总预算、周期预算、并发、RPM 和 TPM。
- Android App 内部使用的 Sub2API Key 与开发者 Key 完全分离，用户不可查看内部 Key。
- Key 泄漏时可以单独撤销或轮换，不影响用户 App 登录。

## 4. 预算和余额

App 对话与外部 API 可以共享总体 AI 额度，但开发者 API 必须设置独立上限：

1. 账户 API 总预算；
2. 单个 API Key 预算；
3. 单个供应商或模型预算；
4. 50%、80%、100% 告警；
5. 达到 100% 自动拒绝外部 API，请求不能耗尽用户为 App 保留的全部额度。

## 5. API 层次

### 5.1 OpenAI 兼容层

首期兼容：

- `GET /v1/models`
- `POST /v1/responses`
- `POST /v1/chat/completions`
- `POST /v1/images/generations`

兼容层只保证文档声明的字段。未支持字段必须返回明确错误，不能静默忽略导致结果不确定。

### 5.2 YLVEN 原生 API

- `/api/public/v1/files`
- `/api/public/v1/images/jobs`
- `/api/public/v1/presentations`
- `/api/public/v1/jobs`
- `/api/public/v1/usage`
- `/api/public/v1/webhooks`

原生 API 用于异步图片、PPT、文件和作品能力。

## 6. 模型命名

同时提供：

1. 稳定别名，例如 `ylven-fast`、`ylven-reasoning`、`ylven-code`、`ylven-vision`、`ylven-image`；
2. 明确的原始模型 ID，供高级用户选择。

稳定别名允许后台调整实际路由；原始模型 ID 可能下线，必须提供弃用期和迁移说明。

## 7. 公共 API 网关

独立部署 `ylven-developer-gateway`，职责包括：

- API Key 鉴权；
- IP 和模型权限；
- 并发、RPM、TPM 和预算；
- 请求规范化；
- 幂等；
- 流式转发；
- 计量、价格快照和扣费；
- 追踪 ID；
- 风险控制；
- 版本兼容。

公共 API 不直接访问 YLVEN 管理数据库的敏感表，应通过服务接口或受限只读模型获取必要信息。

## 8. 商业开放前置条件

在向外部用户收费前必须满足：

- 生产路由已迁移到官方 API 或明确授权通道；
- 服务条款、隐私政策和退款规则完成；
- 计费对账、异常消费、Key 泄漏和预算保护经过压测；
- 模型价格有版本和生效时间；
- 公开状态页、错误码、文档和兼容策略完成；
- 当前订阅代理通道不作为商业 SLA 的生产来源。
