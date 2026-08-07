# 32. 前端页面与状态目录 V1.4

本版共 **332** 个页面、弹层、组件板和设计板，绑定 **2,671** 个独立状态效果图。完整 State ID、PNG 路径、SHA-256 与审批状态以逐页 YAML 和 `contracts/mockup-manifest.csv` 为准。

| Page ID | 名称 | 平台 | 视觉类型 | 父页面 | 状态数 | Work Packet |
|---|---|---|---|---|---:|---|
| `YL-A-001` | App Shell | ANDROID | `COMPONENT_BOARD` | — | 1 | P00-W01 |
| `YL-A-002` | Design System | ANDROID | `DESIGN_BOARD` | — | 1 | P00-W01 |
| `YL-A-003` | 首页/工作/发现/我的 | ANDROID | `COMPONENT_BOARD` | YL-A-001 | 1 | P00-W01 |
| `YL-A-004` | 启动页 | ANDROID | `PAGE` | — | 6 | P02-W01 |
| `YL-A-005` | 登录页 | ANDROID | `PAGE` | — | 7 | P02-W01 |
| `YL-A-006` | 登录页/安全验证 | ANDROID | `OVERLAY` | YL-A-005 | 8 | P02-W01 |
| `YL-A-007` | 安全验证 WebView | ANDROID | `PAGE` | YL-A-005 | 7 | P02-W01, P02-W05 |
| `YL-A-008` | 验证码页 | ANDROID | `PAGE` | YL-A-005 | 11 | P02-W02, P02-W05 |
| `YL-A-009` | 登录完成 | ANDROID | `PAGE` | YL-A-008 | 3 | P02-W02 |
| `YL-A-010` | 注册页 | ANDROID | `PAGE` | — | 7 | P02-W02, P02-W03 |
| `YL-A-011` | 注册页/安全验证 | ANDROID | `OVERLAY` | YL-A-010 | 8 | P02-W03 |
| `YL-A-012` | 注册验证码页 | ANDROID | `PAGE` | YL-A-010 | 11 | P02-W03 |
| `YL-A-013` | 注册完成 | ANDROID | `PAGE` | YL-A-012 | 3 | P02-W03 |
| `YL-A-014` | 全局会话 | ANDROID | `SYSTEM_OVERLAY` | YL-A-001 | 5 | P02-W04 |
| `YL-A-015` | 我的/退出登录 | ANDROID | `OVERLAY` | YL-A-086 | 4 | P02-W04 |
| `YL-A-016` | 我的/登录设备 | ANDROID | `PAGE` | YL-A-086 | 7 | P02-W04 |
| `YL-A-017` | 登录与注册 | ANDROID | `SYSTEM_STATE_BOARD` | — | 5 | P02-W05 |
| `YL-A-018` | 首页 | ANDROID | `PAGE` | — | 7 | P03-W01 |
| `YL-A-019` | 首页/新对话 | ANDROID | `PAGE` | YL-A-018 | 9 | P03-W01 |
| `YL-A-020` | 会话历史 | ANDROID | `PAGE` | YL-A-018（导航父页面） | 9 | P03-W01 |
| `YL-A-021` | 会话搜索 | ANDROID | `PAGE` | YL-A-020 | 9 | P03-W01 |
| `YL-A-022` | 会话菜单 | ANDROID | `OVERLAY` | YL-A-023 | 5 | P03-W01, P03-W05, P06-W02 |
| `YL-A-023` | 会话页 | ANDROID | `PAGE` | — | 13 | P03-W02, P03-W03, P03-W04 |
| `YL-A-024` | 会话输入区 | ANDROID | `COMPONENT_BOARD` | YL-A-023 | 5 | P03-W02 |
| `YL-A-025` | 离线缓存 | ANDROID | `SYSTEM_STATE_BOARD` | YL-A-023 | 3 | P03-W03 |
| `YL-A-026` | AI 回复 | ANDROID | `COMPONENT_BOARD` | YL-A-023 | 7 | P03-W03, P03-W04 |
| `YL-A-027` | AI 回复/代码块 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 3 | P03-W03 |
| `YL-A-028` | AI 回复/表格 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 3 | P03-W03 |
| `YL-A-029` | 通用来源引用组件 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 4 | P03-W03 |
| `YL-A-030` | 消息操作栏 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 3 | P03-W04, P03-W05, P04-W02 |
| `YL-A-031` | 新建对话 | ANDROID | `PAGE` | YL-A-018 | 8 | P03-W04 |
| `YL-A-032` | 输入框 | ANDROID | `COMPONENT_BOARD` | YL-A-024 | 5 | P03-W05 |
| `YL-A-033` | 模型选择器 | ANDROID | `OVERLAY` | YL-A-023 | 6 | P04-W01 |
| `YL-A-034` | 推理选择器 | ANDROID | `OVERLAY` | YL-A-023 | 6 | P04-W01 |
| `YL-A-035` | 我的/AI设置 | ANDROID | `PAGE` | YL-A-086 | 8 | P04-W02, P09-W04 |
| `YL-A-036` | 会话设置 | ANDROID | `PAGE` | YL-A-023 | 8 | P04-W02 |
| `YL-A-037` | 输入区模型状态条 | ANDROID | `COMPONENT_BOARD` | YL-A-024 | 3 | P04-W02 |
| `YL-A-038` | AI 回复来源标签 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 3 | P04-W02 |
| `YL-A-039` | 会话分支切换 | ANDROID | `PAGE` | YL-A-023 | 9 | P04-W03 |
| `YL-A-040` | 多模型对比 | ANDROID | `PAGE` | YL-A-023 | 8 | P04-W03 |
| `YL-A-041` | 对比结果 | ANDROID | `PAGE` | YL-A-040 | 9 | P04-W03 |
| `YL-A-042` | 模型错误弹层 | ANDROID | `OVERLAY` | YL-A-023 | 5 | P04-W03 |
| `YL-A-043` | 我的/服务状态 | ANDROID | `PAGE` | YL-A-086 | 9 | P04-W05, P09-W05 |
| `YL-A-044` | 附件选择器 | ANDROID | `OVERLAY` | YL-A-024 | 6 | P05-W01 |
| `YL-A-045` | 上传进度 | ANDROID | `OVERLAY` | YL-A-023 | 9 | P05-W01 |
| `YL-A-046` | 会话/图片附件 | ANDROID | `COMPONENT_BOARD` | YL-A-023 | 4 | P05-W01 |
| `YL-A-047` | 附件卡片 | ANDROID | `COMPONENT_BOARD` | YL-A-023 | 4 | P05-W02 |
| `YL-A-048` | 资料库选择器 | ANDROID | `OVERLAY` | YL-A-044 | 6 | P05-W03 |
| `YL-A-049` | 工作/文件 | ANDROID | `PAGE` | YL-A-077 | 9 | P05-W03 |
| `YL-A-050` | 文件搜索 | ANDROID | `PAGE` | YL-A-049 | 9 | P05-W03 |
| `YL-A-051` | 文件菜单 | ANDROID | `OVERLAY` | YL-A-052 | 5 | P05-W03 |
| `YL-A-052` | 文件详情 | ANDROID | `PAGE` | YL-A-049 | 9 | P05-W03 |
| `YL-A-053` | 工作/项目 | ANDROID | `PAGE` | YL-A-077 | 9 | P06-W01 |
| `YL-A-054` | 新建项目 | ANDROID | `PAGE` | YL-A-053 | 8 | P06-W01 |
| `YL-A-055` | 项目详情 | ANDROID | `PAGE` | YL-A-053 | 9 | P06-W01 |
| `YL-A-056` | 项目设置 | ANDROID | `PAGE` | YL-A-055 | 8 | P06-W01 |
| `YL-A-057` | 项目设置/指令 | ANDROID | `PAGE` | YL-A-055 | 8 | P06-W02 |
| `YL-A-058` | 项目/资料 | ANDROID | `PAGE` | YL-A-055 | 9 | P06-W02, P06-W03, P06-W04 |
| `YL-A-059` | 作品菜单 | ANDROID | `OVERLAY` | YL-A-079 | 5 | P06-W02, P09-W01 |
| `YL-A-060` | 项目知识引用组件 | ANDROID | `COMPONENT_BOARD` | YL-A-026 | 4 | P06-W03 |
| `YL-A-061` | 工作/搜索 | ANDROID | `PAGE` | YL-A-077 | 9 | P06-W04 |
| `YL-A-062` | 工作/图片 | ANDROID | `PAGE` | YL-A-077 | 9 | P07-W01 |
| `YL-A-063` | 图片工作台 | ANDROID | `PAGE` | YL-A-062 | 8 | P07-W01, P07-W02 |
| `YL-A-064` | 任务详情 | ANDROID | `PAGE` | YL-A-063 | 9 | P07-W02 |
| `YL-A-065` | 图片生成结果 | ANDROID | `PAGE` | YL-A-064 | 7 | P07-W03 |
| `YL-A-066` | 图片作品详情 | ANDROID | `PAGE` | YL-A-065 | 7 | P07-W03 |
| `YL-A-067` | 工作/PPT向导 | ANDROID | `PAGE` | YL-A-077 | 8 | P08-W01 |
| `YL-A-068` | PPT向导/基本信息 | ANDROID | `PAGE` | YL-A-067 | 8 | P08-W01 |
| `YL-A-069` | PPT向导/资料 | ANDROID | `PAGE` | YL-A-067 | 8 | P08-W01 |
| `YL-A-070` | PPT向导/模板 | ANDROID | `PAGE` | YL-A-067 | 8 | P08-W01 |
| `YL-A-071` | PPT向导/大纲 | ANDROID | `PAGE` | YL-A-067 | 8 | P08-W01 |
| `YL-A-072` | PPT大纲编辑 | ANDROID | `PAGE` | YL-A-071 | 8 | P08-W01 |
| `YL-A-073` | PPT任务详情 | ANDROID | `PAGE` | YL-A-072 | 9 | P08-W02 |
| `YL-A-074` | PPT 预览编辑 | ANDROID | `PAGE` | YL-A-073 | 7 | P08-W03 |
| `YL-A-075` | PPT单页编辑 | ANDROID | `PAGE` | YL-A-074 | 8 | P08-W03 |
| `YL-A-076` | PPT 作品详情 | ANDROID | `PAGE` | YL-A-074 | 7 | P08-W04 |
| `YL-A-077` | 工作首页 | ANDROID | `PAGE` | — | 7 | P09-W01 |
| `YL-A-078` | AI 工具目录 | ANDROID | `PAGE` | YL-A-077 | 7 | P09-W01 |
| `YL-A-079` | 工作/作品 | ANDROID | `PAGE` | YL-A-077 | 9 | P09-W01 |
| `YL-A-080` | 工作/任务 | ANDROID | `PAGE` | YL-A-077 | 9 | P09-W01 |
| `YL-A-081` | 发现 | ANDROID | `PAGE` | — | 7 | P09-W02 |
| `YL-A-082` | 发现/分类 | ANDROID | `PAGE` | YL-A-081 | 9 | P09-W02 |
| `YL-A-083` | 发现/Prompt模板 | ANDROID | `PAGE` | YL-A-081 | 9 | P09-W02 |
| `YL-A-084` | 发现/模型实验室 | ANDROID | `PAGE` | YL-A-081 | 9 | P09-W02 |
| `YL-A-085` | 发现/公告 | ANDROID | `PAGE` | YL-A-081 | 9 | P09-W02 |
| `YL-A-086` | 我的 | ANDROID | `PAGE` | — | 7 | P09-W03 |
| `YL-A-087` | 我的/个人资料 | ANDROID | `PAGE` | YL-A-086 | 8 | P09-W03 |
| `YL-A-088` | 我的/收藏 | ANDROID | `PAGE` | YL-A-086 | 9 | P09-W03 |
| `YL-A-089` | 我的/自定义指令 | ANDROID | `PAGE` | YL-A-086 | 9 | P09-W04 |
| `YL-A-090` | 我的/个人记忆 | ANDROID | `PAGE` | YL-A-086 | 8 | P09-W04 |
| `YL-A-091` | 我的/通知设置 | ANDROID | `PAGE` | YL-A-086 | 8 | P09-W04 |
| `YL-A-092` | 我的/外观与语言 | ANDROID | `PAGE` | YL-A-086 | 8 | P09-W04 |
| `YL-A-093` | 我的/数据与隐私 | ANDROID | `PAGE` | YL-A-086 | 8 | P09-W05 |
| `YL-A-094` | 我的/检查更新 | ANDROID | `PAGE` | YL-A-086 | 9 | P09-W05 |
| `YL-A-095` | 我的/帮助与反馈 | ANDROID | `PAGE` | YL-A-086 | 9 | P09-W05 |
| `YL-A-096` | 我的/套餐与会员 | ANDROID | `PAGE` | YL-A-086 | 10 | P10-W01 |
| `YL-A-097` | 套餐确认 | ANDROID | `PAGE` | YL-A-096 | 10 | P10-W01 |
| `YL-A-098` | 我的/钱包 | ANDROID | `PAGE` | YL-A-086 | 10 | P10-W02 |
| `YL-A-099` | 我的/账单 | ANDROID | `PAGE` | YL-A-098 | 10 | P10-W02 |
| `YL-A-100` | 我的/用量中心 | ANDROID | `PAGE` | YL-A-086 | 10 | P10-W03 |
| `YL-D-001` | 开发者中心 | DEVELOPER | `PAGE` | — | 7 | P00-W05 |
| `YL-D-002` | 绑定 Sub2API | DEVELOPER | `PAGE` | — | 10 | P10-W03 |
| `YL-D-003` | 开发者概览 | DEVELOPER | `PAGE` | — | 7 | P11-W01 |
| `YL-D-004` | 开发者开通 | DEVELOPER | `PAGE` | — | 10 | P11-W01 |
| `YL-D-005` | Sub2API登录 | DEVELOPER | `PAGE` | — | 10 | P11-W01 |
| `YL-D-006` | API Key | DEVELOPER | `PAGE` | — | 10 | P11-W01 |
| `YL-D-007` | API Key创建结果 | DEVELOPER | `PAGE` | — | 10 | P11-W01 |
| `YL-D-008` | API Key详情 | DEVELOPER | `PAGE` | — | 10 | P11-W01 |
| `YL-D-009` | API Key预算 | DEVELOPER | `PAGE` | — | 10 | P11-W02 |
| `YL-D-010` | 预算与告警 | DEVELOPER | `PAGE` | — | 10 | P11-W02, P11-W05 |
| `YL-D-011` | API Key安全 | DEVELOPER | `PAGE` | — | 10 | P11-W02 |
| `YL-D-012` | 用量统计 | DEVELOPER | `PAGE` | — | 10 | P11-W04 |
| `YL-D-013` | 调用日志 | DEVELOPER | `PAGE` | — | 7 | P11-W04 |
| `YL-D-014` | API Playground | DEVELOPER | `PAGE` | — | 10 | P11-W04 |
| `YL-D-015` | 模型与价格 | DEVELOPER | `PAGE` | — | 10 | P11-W04 |
| `YL-D-016` | API文档 | DEVELOPER | `PAGE` | — | 6 | P11-W04 |
| `YL-D-017` | Webhook | DEVELOPER | `PAGE` | — | 7 | P11-W05 |
| `YL-DS-001` | 颜色、主题与对比度基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-002` | 字体、字号、行高与字重基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-003` | 间距、网格、圆角、边框与阴影基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-004` | 按钮、图标、Chip 与分段控件基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-005` | 输入框、验证码、搜索和表单基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-006` | 顶部栏、底部导航、侧栏和标签页基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-007` | 卡片、列表、表格、筛选和分页基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-008` | Dialog、Bottom Sheet、Snackbar 与菜单基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-009` | Loading、Skeleton、Empty、Offline 与 Error 基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-010` | 聊天消息、模型标签、代码、表格和引用基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-011` | 文件、附件、上传、解析和资料库基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-012` | 图片工作台、任务、结果和编辑基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-DS-013` | PPT 工作台、缩略图、预览和后台/开发者组件基准板 | DESIGN_SYSTEM | `PAGE` | — | 1 | P00-W01 |
| `YL-M-001` | 系统设置/工程信息 | ADMIN | `PAGE` | — | 8 | P00-W01 |
| `YL-M-002` | 系统设置/环境配置 | ADMIN | `PAGE` | — | 11 | P00-W01 |
| `YL-M-003` | App发布/构建记录 | ADMIN | `PAGE` | — | 8 | P00-W01 |
| `YL-M-004` | 内容运营/设计配置 | ADMIN | `PAGE` | — | 11 | P00-W01 |
| `YL-M-005` | 发现与首页/导航配置 | ADMIN | `PAGE` | — | 11 | P00-W01 |
| `YL-M-006` | 运维/服务健康 | ADMIN | `PAGE` | — | 7 | P00-W02 |
| `YL-M-007` | 运维/任务队列 | ADMIN | `PAGE` | — | 8 | P00-W02, P09-W01, P12-W04 |
| `YL-M-008` | 运维/定时任务 | ADMIN | `PAGE` | — | 8 | P00-W02 |
| `YL-M-009` | 运维/数据库迁移 | ADMIN | `PAGE` | — | 8 | P00-W03, P13-W01 |
| `YL-M-010` | 运维/Redis | ADMIN | `PAGE` | — | 8 | P00-W03 |
| `YL-M-011` | 运维/事件流 | ADMIN | `PAGE` | — | 8 | P00-W03 |
| `YL-M-012` | 文件与存储/存储配置 | ADMIN | `PAGE` | — | 11 | P00-W03 |
| `YL-M-013` | 系统设置/API版本 | ADMIN | `PAGE` | — | 11 | P00-W03 |
| `YL-M-014` | 系统设置/功能追踪 | ADMIN | `PAGE` | — | 8 | P00-W04 |
| `YL-M-015` | 可观测性/总览 | ADMIN | `PAGE` | — | 7 | P00-W04 |
| `YL-M-016` | 运维/构建记录 | ADMIN | `PAGE` | — | 8 | P00-W04 |
| `YL-M-017` | App发布/阶段交付 | ADMIN | `PAGE` | — | 8 | P00-W04 |
| `YL-M-018` | 模型与供应商/能力探测 | ADMIN | `PAGE` | — | 11 | P00-W04, P00-W05, P04-W01, P04-W05 |
| `YL-M-019` | 安全与审计/秘密状态 | ADMIN | `PAGE` | — | 8 | P00-W05 |
| `YL-M-020` | 管理员与权限/菜单 | ADMIN | `PAGE` | — | 8 | P00-W05 |
| `YL-M-021` | 开发者平台/门户配置 | ADMIN | `PAGE` | — | 11 | P00-W05 |
| `YL-M-022` | 认证与用户/认证策略 | ADMIN | `PAGE` | — | 11 | P01-W01 |
| `YL-M-023` | 认证与用户/安全验证 | ADMIN | `PAGE` | — | 8 | P01-W01, P01-W02, P02-W01, P02-W03, P02-W05 |
| `YL-M-024` | 认证与用户/Turnstile配置 | ADMIN | `PAGE` | — | 11 | P01-W01, P02-W01 |
| `YL-M-025` | 认证与用户/邮件配置 | ADMIN | `PAGE` | — | 11 | P01-W01 |
| `YL-M-026` | 认证与用户/验证码策略 | ADMIN | `PAGE` | — | 11 | P01-W01, P01-W02, P01-W04, P02-W01, P02-W02, P02-W03 |
| `YL-M-027` | 认证与用户/用户列表 | ADMIN | `PAGE` | — | 8 | P01-W01, P01-W04, P01-W05, P02-W03 |
| `YL-M-028` | 认证与用户/异常注册 | ADMIN | `PAGE` | — | 8 | P01-W02 |
| `YL-M-029` | 认证与用户/登录日志 | ADMIN | `PAGE` | — | 7 | P01-W02, P02-W02 |
| `YL-M-030` | 认证与用户/会话管理 | ADMIN | `PAGE` | — | 8 | P01-W02, P01-W03, P02-W01, P02-W02, P02-W04 |
| `YL-M-031` | 认证与用户/设备管理 | ADMIN | `PAGE` | — | 8 | P01-W03, P02-W04 |
| `YL-M-032` | 安全与审计/风控规则 | ADMIN | `PAGE` | — | 8 | P01-W03 |
| `YL-M-033` | 安全与审计/认证事件 | ADMIN | `PAGE` | — | 8 | P01-W03 |
| `YL-M-034` | 系统设置/邮件服务 | ADMIN | `PAGE` | — | 11 | P01-W04 |
| `YL-M-035` | 系统设置/Turnstile | ADMIN | `PAGE` | — | 11 | P01-W04 |
| `YL-M-036` | 认证与用户/用户详情 | ADMIN | `PAGE` | — | 11 | P01-W04, P02-W03, P09-W03 |
| `YL-M-037` | 管理员与权限/角色 | ADMIN | `PAGE` | — | 8 | P01-W05 |
| `YL-M-038` | 管理员与权限/管理员 | ADMIN | `PAGE` | — | 8 | P01-W05 |
| `YL-M-039` | 安全与审计/高风险操作 | ADMIN | `PAGE` | — | 8 | P01-W05 |
| `YL-M-040` | 通知中心/邮件模板 | ADMIN | `PAGE` | — | 11 | P01-W05 |
| `YL-M-041` | 认证与用户/登录配置 | ADMIN | `PAGE` | — | 11 | P02-W01 |
| `YL-M-042` | 认证与用户/注册配置 | ADMIN | `PAGE` | — | 11 | P02-W02 |
| `YL-M-043` | 认证与用户/密码策略 | ADMIN | `PAGE` | — | 11 | P02-W02 |
| `YL-M-044` | 安全与审计/移动端安全 | ADMIN | `PAGE` | — | 8 | P02-W05 |
| `YL-M-045` | 内容运营/首页配置 | ADMIN | `PAGE` | — | 11 | P03-W01 |
| `YL-M-046` | 对话与内容/会话列表 | ADMIN | `PAGE` | — | 8 | P03-W01, P03-W05 |
| `YL-M-047` | 对话与内容/搜索配置 | ADMIN | `PAGE` | — | 11 | P03-W01 |
| `YL-M-048` | 对话与内容/会话详情 | ADMIN | `PAGE` | — | 11 | P03-W01, P04-W02 |
| `YL-M-049` | 对话与内容/删除策略 | ADMIN | `PAGE` | — | 11 | P03-W01 |
| `YL-M-050` | 对话与内容/标题模型 | ADMIN | `PAGE` | — | 8 | P03-W02 |
| `YL-M-051` | 模型与供应商/模型目录 | ADMIN | `PAGE` | — | 11 | P03-W02, P04-W01, P04-W02, P04-W04 |
| `YL-M-052` | 运维/流式连接 | ADMIN | `PAGE` | — | 8 | P03-W02 |
| `YL-M-053` | 对话与内容/运行记录 | ADMIN | `PAGE` | — | 8 | P03-W02, P03-W03, P03-W04, P04-W02 |
| `YL-M-054` | 对话与内容/消息查询 | ADMIN | `PAGE` | — | 8 | P03-W02 |
| `YL-M-055` | 对话与内容/引用记录 | ADMIN | `PAGE` | — | 8 | P03-W03 |
| `YL-M-056` | 对话与内容/导出记录 | ADMIN | `PAGE` | — | 8 | P03-W04, P03-W05 |
| `YL-M-057` | 对话与内容/用户反馈 | ADMIN | `PAGE` | — | 8 | P03-W04 |
| `YL-M-058` | 对话与内容/保留策略 | ADMIN | `PAGE` | — | 11 | P03-W04 |
| `YL-M-059` | AI工具/语音配置 | ADMIN | `PAGE` | — | 11 | P03-W05 |
| `YL-M-060` | 安全与审计/访问拒绝 | ADMIN | `PAGE` | — | 8 | P03-W05 |
| `YL-M-061` | 可观测性/AI运行 | ADMIN | `PAGE` | — | 8 | P03-W05 |
| `YL-M-062` | 模型与供应商/供应商 | ADMIN | `PAGE` | — | 8 | P04-W01 |
| `YL-M-063` | 模型与供应商/推理映射 | ADMIN | `PAGE` | — | 11 | P04-W01, P04-W04 |
| `YL-M-064` | 模型与供应商/默认配置 | ADMIN | `PAGE` | — | 11 | P04-W02, P09-W04 |
| `YL-M-065` | 对话与内容/分支 | ADMIN | `PAGE` | — | 8 | P04-W02, P04-W03 |
| `YL-M-066` | AI工具/多模型对比 | ADMIN | `PAGE` | — | 8 | P04-W03 |
| `YL-M-067` | 模型与供应商/服务状态 | ADMIN | `PAGE` | — | 7 | P04-W03 |
| `YL-M-068` | 模型与供应商/路由策略 | ADMIN | `PAGE` | — | 11 | P04-W04, P13-W02 |
| `YL-M-069` | 模型与供应商/通道 | ADMIN | `PAGE` | — | 11 | P04-W04, P13-W02 |
| `YL-M-070` | 模型与供应商/能力 | ADMIN | `PAGE` | — | 11 | P04-W04, P05-W01 |
| `YL-M-071` | 模型与供应商/健康探测 | ADMIN | `PAGE` | — | 7 | P04-W05 |
| `YL-M-072` | 模型与供应商/运行策略 | ADMIN | `PAGE` | — | 11 | P04-W05 |
| `YL-M-073` | 商业化/用量记录 | ADMIN | `PAGE` | — | 8 | P04-W05 |
| `YL-M-074` | 商业化/模型价格 | ADMIN | `PAGE` | — | 8 | P04-W05, P10-W03, P10-W05, P11-W04 |
| `YL-M-075` | 运维/事故管理 | ADMIN | `PAGE` | — | 7 | P04-W05 |
| `YL-M-076` | 文件与存储/文件列表 | ADMIN | `PAGE` | — | 8 | P05-W01, P05-W03, P05-W04, P07-W01 |
| `YL-M-077` | 文件与存储/上传记录 | ADMIN | `PAGE` | — | 8 | P05-W01 |
| `YL-M-078` | 文件与存储/安全策略 | ADMIN | `PAGE` | — | 11 | P05-W01 |
| `YL-M-079` | 文件与存储/隔离区 | ADMIN | `PAGE` | — | 8 | P05-W01 |
| `YL-M-080` | 文件与存储/处理任务 | ADMIN | `PAGE` | — | 8 | P05-W01 |
| `YL-M-081` | 文件与存储/解析任务 | ADMIN | `PAGE` | — | 8 | P05-W02, P05-W04 |
| `YL-M-082` | 文件与存储/文件详情 | ADMIN | `PAGE` | — | 11 | P05-W02, P05-W03 |
| `YL-M-083` | 对话与内容/附件 | ADMIN | `PAGE` | — | 8 | P05-W03 |
| `YL-M-084` | 文件与存储/下载记录 | ADMIN | `PAGE` | — | 8 | P05-W03 |
| `YL-M-085` | 文件与存储/删除策略 | ADMIN | `PAGE` | — | 11 | P05-W03 |
| `YL-M-086` | 商业化/权益配置 | ADMIN | `PAGE` | — | 11 | P05-W04, P06-W04 |
| `YL-M-087` | 文件与存储/清理任务 | ADMIN | `PAGE` | — | 8 | P05-W04 |
| `YL-M-088` | 文件与存储/供应商引用 | ADMIN | `PAGE` | — | 8 | P05-W04 |
| `YL-M-089` | 项目与知识库/项目列表 | ADMIN | `PAGE` | — | 8 | P06-W01 |
| `YL-M-090` | 项目与知识库/项目详情 | ADMIN | `PAGE` | — | 11 | P06-W01, P06-W02 |
| `YL-M-091` | 项目与知识库/项目指令 | ADMIN | `PAGE` | — | 8 | P06-W02 |
| `YL-M-092` | 项目与知识库/文件 | ADMIN | `PAGE` | — | 8 | P06-W02 |
| `YL-M-093` | 项目与知识库/索引任务 | ADMIN | `PAGE` | — | 8 | P06-W02, P06-W03, P06-W04 |
| `YL-M-094` | 项目与知识库/检索诊断 | ADMIN | `PAGE` | — | 11 | P06-W03 |
| `YL-M-095` | 项目与知识库/上下文策略 | ADMIN | `PAGE` | — | 11 | P06-W03 |
| `YL-M-096` | 项目与知识库/引用 | ADMIN | `PAGE` | — | 8 | P06-W03 |
| `YL-M-097` | 项目与知识库/搜索 | ADMIN | `PAGE` | — | 8 | P06-W04 |
| `YL-M-098` | 项目与知识库/诊断 | ADMIN | `PAGE` | — | 11 | P06-W04 |
| `YL-M-099` | 项目与知识库/检索测试 | ADMIN | `PAGE` | — | 11 | P06-W04 |
| `YL-M-100` | AI工具/图片配置 | ADMIN | `PAGE` | — | 11 | P07-W01, P07-W04 |
| `YL-M-101` | AI工具/图片模型 | ADMIN | `PAGE` | — | 8 | P07-W01 |
| `YL-M-102` | AI工具/图片任务 | ADMIN | `PAGE` | — | 8 | P07-W02, P07-W04 |
| `YL-M-103` | 商业化/冻结记录 | ADMIN | `PAGE` | — | 8 | P07-W02, P10-W03 |
| `YL-M-104` | 通知中心/模板 | ADMIN | `PAGE` | — | 11 | P07-W03 |
| `YL-M-105` | 作品中心/图片作品 | ADMIN | `PAGE` | — | 8 | P07-W03 |
| `YL-M-106` | 作品中心/作品详情 | ADMIN | `PAGE` | — | 11 | P07-W03, P09-W01 |
| `YL-M-107` | 作品中心/下载记录 | ADMIN | `PAGE` | — | 8 | P07-W03, P08-W04 |
| `YL-M-108` | 作品中心/作品版本 | ADMIN | `PAGE` | — | 11 | P07-W03, P08-W03 |
| `YL-M-109` | 安全与审计/内容安全 | ADMIN | `PAGE` | — | 8 | P07-W04 |
| `YL-M-110` | 商业化/账本 | ADMIN | `PAGE` | — | 8 | P07-W04, P08-W04, P10-W02 |
| `YL-M-111` | AI工具/PPT项目 | ADMIN | `PAGE` | — | 8 | P08-W01, P08-W03, P08-W04 |
| `YL-M-112` | AI工具/PPT模板 | ADMIN | `PAGE` | — | 11 | P08-W01, P08-W03, P08-W04 |
| `YL-M-113` | AI工具/PPT任务 | ADMIN | `PAGE` | — | 8 | P08-W01, P08-W02, P08-W03, P08-W04 |
| `YL-M-114` | AI工具/PPT诊断 | ADMIN | `PAGE` | — | 11 | P08-W02, P08-W04 |
| `YL-M-115` | 作品中心/PPT作品 | ADMIN | `PAGE` | — | 8 | P08-W02 |
| `YL-M-116` | 内容运营/工作页配置 | ADMIN | `PAGE` | — | 11 | P09-W01 |
| `YL-M-117` | AI工具/工具目录 | ADMIN | `PAGE` | — | 11 | P09-W01 |
| `YL-M-118` | 作品中心/作品列表 | ADMIN | `PAGE` | — | 8 | P09-W01 |
| `YL-M-119` | 内容运营/发现页 | ADMIN | `PAGE` | — | 8 | P09-W02, P09-W03 |
| `YL-M-120` | 内容运营/Prompt模板 | ADMIN | `PAGE` | — | 11 | P09-W02 |
| `YL-M-121` | 模型与供应商/模型展示 | ADMIN | `PAGE` | — | 8 | P09-W02 |
| `YL-M-122` | 内容运营/公告 | ADMIN | `PAGE` | — | 8 | P09-W02 |
| `YL-M-123` | 内容运营/收藏统计 | ADMIN | `PAGE` | — | 8 | P09-W03 |
| `YL-M-124` | 对话与内容/指令策略 | ADMIN | `PAGE` | — | 11 | P09-W04 |
| `YL-M-125` | 对话与内容/记忆管理 | ADMIN | `PAGE` | — | 8 | P09-W04 |
| `YL-M-126` | 通知中心/用户设置 | ADMIN | `PAGE` | — | 8 | P09-W04 |
| `YL-M-127` | 系统设置/默认外观 | ADMIN | `PAGE` | — | 11 | P09-W04 |
| `YL-M-128` | 安全与审计/数据导出 | ADMIN | `PAGE` | — | 8 | P09-W05 |
| `YL-M-129` | 安全与审计/删除请求 | ADMIN | `PAGE` | — | 8 | P09-W05 |
| `YL-M-130` | 系统运营/App版本 | ADMIN | `PAGE` | — | 11 | P09-W05, P13-W03 |
| `YL-M-131` | 系统运营/反馈工单 | ADMIN | `PAGE` | — | 8 | P09-W05 |
| `YL-M-132` | 可观测性/事故管理 | ADMIN | `PAGE` | — | 7 | P09-W05 |
| `YL-M-133` | 商业化/产品与套餐 | ADMIN | `PAGE` | — | 8 | P10-W01 |
| `YL-M-134` | 商业化/订单 | ADMIN | `PAGE` | — | 8 | P10-W01 |
| `YL-M-135` | 商业化/支付回调 | ADMIN | `PAGE` | — | 8 | P10-W01 |
| `YL-M-136` | 商业化/钱包 | ADMIN | `PAGE` | — | 8 | P10-W01, P10-W02 |
| `YL-M-137` | 商业化/订阅 | ADMIN | `PAGE` | — | 8 | P10-W02 |
| `YL-M-138` | 商业化/权益 | ADMIN | `PAGE` | — | 8 | P10-W02 |
| `YL-M-139` | 商业化/用量 | ADMIN | `PAGE` | — | 8 | P10-W03 |
| `YL-M-140` | Sub2API与通道/用户映射 | ADMIN | `PAGE` | — | 11 | P10-W03 |
| `YL-M-141` | Sub2API与通道/API Key映射 | ADMIN | `PAGE` | — | 11 | P10-W04 |
| `YL-M-142` | Sub2API与通道/余额同步 | ADMIN | `PAGE` | — | 11 | P10-W04 |
| `YL-M-143` | Sub2API与通道/套餐同步 | ADMIN | `PAGE` | — | 11 | P10-W04 |
| `YL-M-144` | Sub2API与通道/用量同步 | ADMIN | `PAGE` | — | 11 | P10-W04 |
| `YL-M-145` | 商业化/对账 | ADMIN | `PAGE` | — | 8 | P10-W04 |
| `YL-M-146` | Sub2API与通道/同步任务 | ADMIN | `PAGE` | — | 11 | P10-W04 |
| `YL-M-147` | 商业化/交易管理 | ADMIN | `PAGE` | — | 8 | P10-W05 |
| `YL-M-148` | 商业化/钱包与账本 | ADMIN | `PAGE` | — | 8 | P10-W05 |
| `YL-M-149` | 商业化/退款 | ADMIN | `PAGE` | — | 8 | P10-W05 |
| `YL-M-150` | 商业化/额度策略 | ADMIN | `PAGE` | — | 8 | P10-W05 |
| `YL-M-151` | Sub2API与通道/通道配置 | ADMIN | `PAGE` | — | 11 | P10-W05 |
| `YL-M-152` | 开发者平台/概览 | ADMIN | `PAGE` | — | 8 | P11-W01 |
| `YL-M-153` | 开发者平台/开发者用户 | ADMIN | `PAGE` | — | 8 | P11-W01, P11-W05 |
| `YL-M-154` | Sub2API与通道/OIDC | ADMIN | `PAGE` | — | 11 | P11-W01 |
| `YL-M-155` | 开发者平台/API Key | ADMIN | `PAGE` | — | 8 | P11-W01 |
| `YL-M-156` | 开发者平台/预算 | ADMIN | `PAGE` | — | 8 | P11-W02 |
| `YL-M-157` | 开发者平台/API Key策略 | ADMIN | `PAGE` | — | 11 | P11-W02, P11-W05 |
| `YL-M-158` | 开发者平台/安全事件 | ADMIN | `PAGE` | — | 8 | P11-W02 |
| `YL-M-159` | 开发者平台/限流策略 | ADMIN | `PAGE` | — | 11 | P11-W02 |
| `YL-M-160` | 开发者平台/API版本 | ADMIN | `PAGE` | — | 11 | P11-W02, P11-W03, P11-W05 |
| `YL-M-161` | 开发者平台/API日志 | ADMIN | `PAGE` | — | 7 | P11-W02, P11-W03, P11-W04 |
| `YL-M-162` | 开发者平台/模型别名 | ADMIN | `PAGE` | — | 8 | P11-W03 |
| `YL-M-163` | 开发者平台/错误码 | ADMIN | `PAGE` | — | 8 | P11-W04 |
| `YL-M-164` | 开发者平台/用量 | ADMIN | `PAGE` | — | 8 | P11-W04 |
| `YL-M-165` | 开发者平台/Playground | ADMIN | `PAGE` | — | 8 | P11-W04 |
| `YL-M-166` | 开发者平台/API文档 | ADMIN | `PAGE` | — | 8 | P11-W04 |
| `YL-M-167` | 开发者平台/告警 | ADMIN | `PAGE` | — | 7 | P11-W05 |
| `YL-M-168` | 开发者平台/Webhook | ADMIN | `PAGE` | — | 8 | P11-W05 |
| `YL-M-169` | 安全与审计/WAF | ADMIN | `PAGE` | — | 8 | P12-W01 |
| `YL-M-170` | 安全与审计/扫描结果 | ADMIN | `PAGE` | — | 8 | P12-W01 |
| `YL-M-171` | 安全与审计/密钥管理 | ADMIN | `PAGE` | — | 8 | P12-W01 |
| `YL-M-172` | 安全与审计/操作审计 | ADMIN | `PAGE` | — | 8 | P12-W01 |
| `YL-M-173` | 安全与审计/风险事件 | ADMIN | `PAGE` | — | 8 | P12-W02 |
| `YL-M-174` | 安全与审计/审批 | ADMIN | `PAGE` | — | 8 | P12-W02 |
| `YL-M-175` | 可观测性/指标 | ADMIN | `PAGE` | — | 7 | P12-W02 |
| `YL-M-176` | 可观测性/日志 | ADMIN | `PAGE` | — | 7 | P12-W02 |
| `YL-M-177` | 可观测性/链路 | ADMIN | `PAGE` | — | 7 | P12-W02 |
| `YL-M-178` | 可观测性/告警 | ADMIN | `PAGE` | — | 7 | P12-W03 |
| `YL-M-179` | 可观测性/事故 | ADMIN | `PAGE` | — | 7 | P12-W03 |
| `YL-M-180` | 可观测性/压测 | ADMIN | `PAGE` | — | 7 | P12-W03 |
| `YL-M-181` | 可观测性/数据库 | ADMIN | `PAGE` | — | 8 | P12-W04 |
| `YL-M-182` | 可观测性/恢复演练 | ADMIN | `PAGE` | — | 8 | P12-W04, P12-W05, P13-W01 |
| `YL-M-183` | 文件与存储/生命周期 | ADMIN | `PAGE` | — | 8 | P12-W04 |
| `YL-M-184` | 可观测性/备份 | ADMIN | `PAGE` | — | 8 | P12-W05 |
| `YL-M-185` | 安全与审计/隐私请求 | ADMIN | `PAGE` | — | 8 | P12-W05 |
| `YL-M-186` | 系统设置/数据保留 | ADMIN | `PAGE` | — | 8 | P12-W05 |
| `YL-M-187` | 可观测性/运行手册 | ADMIN | `PAGE` | — | 11 | P12-W05 |
| `YL-M-188` | 质量与发布/测试 | ADMIN | `PAGE` | — | 8 | P13-W01 |
| `YL-M-189` | 质量与发布/真机验收 | ADMIN | `PAGE` | — | 8 | P13-W01 |
| `YL-M-190` | 管理员与权限/审计 | ADMIN | `PAGE` | — | 8 | P13-W01 |
| `YL-M-191` | 模型与供应商/故障切换 | ADMIN | `PAGE` | — | 8 | P13-W02 |
| `YL-M-192` | 可观测性/容量规划 | ADMIN | `PAGE` | — | 7 | P13-W03 |
| `YL-M-193` | 质量与发布/商业就绪 | ADMIN | `PAGE` | — | 8 | P13-W03 |
| `YL-M-194` | 质量与发布/最终交付 | ADMIN | `PAGE` | — | 8 | P13-W03 |
| `YL-M-195` | 质量与发布/最终验收 | ADMIN | `PAGE` | — | 8 | P13-W03 |
| `YL-W-001` | 安全验证承载页 | WEB | `PAGE` | — | 7 | P01-W01, P02-W01, P02-W05 |
| `YL-W-002` | 邮箱验证状态页 | WEB | `PAGE` | — | 7 | P01-W01, P02-W02, P02-W03 |
| `YL-W-003` | YLVEN OIDC 授权与回调页 | WEB | `PAGE` | — | 7 | P10-W04, P11-W01 |
| `YL-W-004` | APK 下载与更新页 | WEB | `PAGE` | — | 7 | P09-W05, P13-W03 |
| `YL-W-005` | 公开服务状态页 | WEB | `PAGE` | — | 7 | P04-W05, P09-W05, P12-W03 |
| `YL-W-006` | 会话或作品安全分享页 | WEB | `PAGE` | — | 7 | P03-W04, P08-W04 |
| `YL-W-007` | 维护、错误与不兼容提示页 | WEB | `PAGE` | — | 7 | P12-W03, P13-W03 |
