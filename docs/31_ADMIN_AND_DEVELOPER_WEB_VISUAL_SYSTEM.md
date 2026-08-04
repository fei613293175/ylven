# 31 — 管理后台与开发者中心视觉系统

## 1. 管理后台

基准画布 1440×1000px；最小支持宽度 1280px。侧栏 248px，可收起至 72px；顶栏 64px；内容边距 24px；卡片间距 16px。表头 44px、数据行 48px、控件 40px、紧凑控件 32px、抽屉 520px。

所有 195 个后台控制菜单均被分配 `YL-M-*` Page ID。即使某控制由后端 Feature 首先引入，也必须有可查询、可配置、可审计、可诊断的后台页面，不能只留接口。

列表页必须覆盖 Loading、Populated、Empty、Filter Active、Bulk Selected、Permission Denied、Network Error 和 Server Error。配置页必须额外覆盖 Dirty、Validation Error、Saving、Success、Version Conflict 与 Rollback Confirm。

## 2. 开发者中心

基准画布 1440×1000px；侧栏 240px；顶栏 64px；内容边距 32px。文档正文宽 760px，右侧目录 220px。API Key 完整值只在 Created Once 状态展示；所有示例请求必须使用 YLVEN 域名和测试 Key 前缀，不能出现真实 Secret。

## 3. 响应式和移动访问

专业后台首期以桌面为主。小于 1280px 时允许侧栏折叠，但数据表不能挤成不可读卡片；关键财务和审计页面不提供手机端高风险写操作。开发者中心在 1024px 以上完整支持，手机端可只读概览、Key 禁用和告警查看。

## 4. 图表与数据真实性

图表空数据、部分数据和采集延迟均有独立状态。不得以随机数或静态图表冒充真实指标。金额、Token、时间范围必须标注统计截止时间、币种和计价版本。
