# 10 Sub2API 轻度兼容与官方 API 迁移

## 1. 边界

Sub2API 是上游 AI 网关，不是 YLVEN 业务后端。保持独立部署、独立数据库和独立升级周期。YLVEN 通过 `Sub2APIAdapter` 使用其能力。

## 2. 统一账号

YLVEN 是身份权威。每个 YLVEN 用户通过 `gateway_user_links` 映射一个 Sub2API 用户。未来 YLVEN 提供 OIDC，Sub2API 使用 YLVEN `sub` 完成单点登录。禁止同步密码，禁止仅用可修改邮箱作为永久绑定键。

### 2.1 现有账号绑定

已有 Sub2API 用户通过明确的绑定流程关联 YLVEN：验证 YLVEN 登录、验证 Sub2API 身份、用户确认、写入唯一映射。保留原余额、API Key 和使用记录。

### 2.2 新用户

首次需要 AI 或 API 时通过幂等 `ensure user` 操作创建 Sub2API 用户。相同 YLVEN subject 重复调用必须返回同一 Sub2API 用户。

## 3. Key 类型

### 3.1 App 内部 Key

- 每个用户一个或按策略轮换；
- 仅 YLVEN 后端保存；
- Android 永远不可读取；
- App 消费计入该用户。

### 3.2 开发者 Key

- 用户可创建多个；
- 由 YLVEN 开发者中心管理；
- 映射到 Sub2API 或 YLVEN Developer Gateway；
- 支持独立预算、限流、模型权限和撤销。

## 4. 钱包与余额

YLVEN 保存完整资金与权益账本；Sub2API 保存实时执行余额或订阅投影。写入方向固定为 YLVEN -> Sub2API，使用记录方向固定为 Sub2API -> YLVEN。

不得做字段级双向同步，也不得定时互相覆盖余额。

额度拆分：

- 充值余额；
- 套餐额度；
- 赠送额度。

显示可汇总，但账本必须区分来源、过期时间和消费顺序。

## 5. 支付中心

App、开发者中心或 Sub2API 用户页点击充值，最终都进入 YLVEN Payment Center。支付成功后：

1. 写订单和支付事件；
2. 创建账本分录；
3. 发放额度/权益；
4. 创建 Sub2API 同步任务；
5. 同步成功后标记 CREDITED；
6. 同步失败保持 PAID + CREDIT_PENDING，后台自动重试，不要求重复付款。

## 6. 使用回流和对账

每条使用记录带：用户、Sub2API 用户、Key、来源、会话/任务、模型、推理档位、Token、成本、价格版本和追踪 ID。每日自动对账 YLVEN 使用事件与 Sub2API 扣费；差异进入人工处理列表，不自动覆盖财务事实。

## 7. 测试与生产通道

```text
当前：TEST + SUBSCRIPTION_PROXY
未来：PRODUCTION + OFFICIAL_API
```

模型路由表保存 channel_environment 和 channel_type。迁移时只调整后台通道和路由，不修改 Android 请求协议、会话数据或钱包。

## 8. 升级策略

- 固定经过测试的 Sub2API 版本，不直接使用 `latest`。
- 升级前在 staging 执行登录、用户映射、Key、文本、流式、文件、图片、扣费、续费和到期回归。
- 保留上一镜像和数据库备份，允许快速回滚。
- YLVEN Adapter 必须有契约测试，防止 Sub2API 接口变化静默破坏业务。
