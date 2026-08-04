# 12 统一账号、SSO、共享额度与 Sub2API 轻度兼容

## 1. 目标体验

同一 YLVEN 用户在 Android App、YLVEN 开发者中心和 Sub2API 用户端使用同一身份。用户在 YLVEN App 或网页购买套餐、获得套餐额度或充值余额后，App 内 AI 消费与外部 API 消费共享总 AI 额度；外部 API 另有账号级和单 Key 预算，避免 Key 泄漏耗尽全部 App 额度。

## 2. 账号映射

YLVEN 内部 UUID 是永久主身份，公开数字 UID 只用于展示。`gateway_user_links` 保存 YLVEN UUID、网关类型、Sub2API user_id、OIDC subject、绑定状态和最后同步时间。邮箱、手机号、用户名都不能作为永久关联键。已有 Sub2API 用户通过一次显式绑定保留原余额、API Key 和历史；新用户采用幂等 ensure-user 流程。

## 3. OIDC

YLVEN Auth 提供 Authorization Code + PKCE、ID Token 和 UserInfo。OIDC `sub` 使用不可变 YLVEN UUID；Sub2API 只保存外部身份映射，不接收 YLVEN 密码。登录、解绑、账号禁用、Token 撤销和管理员强制下线都必须有审计。

## 4. 资金与权益模型

至少区分现金充值额度、套餐额度、赠送额度和冻结额度。正式余额来自不可变账本和可重建投影，不能仅维护一列可写 balance。默认消费顺序由后台版本化配置，例如先消费最早到期额度，再赠送，再现金。退款、撤销、过期和补偿均通过账本交易，不直接覆盖历史。

## 5. 单向同步

写入方向为 YLVEN → Sub2API：确保用户、投影可用额度、模型组、API 权限、并发和预算。回流方向为 Sub2API → YLVEN：调用记录、Token、实际成本、API Key、来源和执行余额。禁止两个系统定时相互覆盖 balance 字段。同步使用 Outbox、幂等键、重试、死信和每日对账；支付成功但投影失败进入 `CREDIT_PENDING`，不要求用户重复支付。

## 6. App 与开发者 Key

每个用户有一个仅服务端可见的 App 内部调用凭据；Android 永远不能获得它。用户创建的开发者 API Key 独立命名、单次显示明文、可轮换/禁用/过期，并支持 IP 白名单、总额度、周期预算和模型白名单。两类 Key 归属同一用户，但鉴权入口和安全策略分离。

## 7. 支付入口

App、YLVEN Web 和 Sub2API 页面可以展示充值入口，但都跳转或调用同一个 YLVEN Payment Center。支付订单、回调验签、退款和账本只在 YLVEN 处理；Sub2API 不应同时启用一套独立且不可对账的支付账本。

## 8. 迁移官方 API

模型通道必须有 `environment`、`channel_type`、`provider`、`adapter`、`credential_ref` 和 `routing_policy`。从订阅测试线路切换官方 API 时，只改变服务端路由和价格版本，不迁移 Android 用户、会话、文件、钱包或公共 API Key。
