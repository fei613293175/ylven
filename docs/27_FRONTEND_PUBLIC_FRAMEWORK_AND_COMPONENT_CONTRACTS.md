# 27 — 前端公共框架与组件合同

## 1. Android 公共框架

Android 使用 Kotlin、Jetpack Compose、Material 3 与单向数据流。所有页面必须建立在下列公共层上：

```text
YlvenApp
└── YlvenTheme (YL-DS-1.2.0)
    └── YlvenNavHost
        └── YlvenScaffold
            ├── YlvenTopBar
            ├── PageStateHost
            ├── Screen Content
            ├── YlvenBottomNavigation（仅一级四栏）
            └── SnackbarHost
```

### 不允许的做法

- 每个页面自己创建 Scaffold、状态栏颜色和 Snackbar Host；
- 每个页面重新定义按钮、输入框、卡片和错误页；
- 在 Composable 中出现裸 `Color(0x...)`、未经 Token 包装的 `dp/sp` 视觉常量；
- 直接用效果图像素值绕过 density 和字体缩放；
- 业务 ViewModel 持有 Android Context 或直接访问供应商 API。

## 2. 公共状态容器

`YlvenStateLayout` 必须统一覆盖：Loading、Empty、Offline、Timeout、Server Error、Permission Denied、Rate Limited 和 Content Blocked。页面只提供状态内容和恢复事件，不得自由改变图标尺寸、文案层级或按钮位置。

## 3. 聊天公共框架

聊天必须拆为：ConversationHeader、MessageTimeline、MessagePartRenderer、StreamingCursor、ToolCallCard、MessageActions、AttachmentStrip、ChatComposer。不同模型回答共享同一渲染器，仅能力与元数据不同。每条 AI 回答必须显示真实模型、推理档位、工具/联网状态；禁止把供应商内部推理链伪装成可见内容。

## 4. 工作与作品公共框架

图片、PPT、文件解析均使用统一 Job 数据模型和 `YlvenJobCard`。任务页面只展示后端真实状态：queued、preparing、running、success、failed、cancelled、retrying。不得用本地计时器伪造进度。

## 5. 管理后台公共框架

后台使用固定 `AdminShell`：248px 侧栏、64px 顶栏、24px 内容边距。列表页统一由 PageHeader、FilterBar、DataTable、Pagination、BulkActionBar、Empty/Error State 组成。配置修改必须有草稿、版本、差异、保存、冲突和回滚，不允许提交后直接覆盖且无审计。

## 6. 开发者中心公共框架

开发者中心使用 240px 侧栏、64px 顶栏和 32px 内容边距。API Key 只在创建成功状态展示完整值一次；日志、价格、预算和文档使用统一表格、代码块和错误展示。

## 7. 组件编号和测试

公共组件编号位于 `contracts/ui-component-catalog.yaml`。每个组件必须有 Compose Preview 或 Web Story/fixture，覆盖 catalog 中列出的全部状态，并以 `YL-DS-*` 基准板进行视觉回归。页面不得建立同功能的私有替代组件。
