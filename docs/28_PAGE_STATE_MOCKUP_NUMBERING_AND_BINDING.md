# 28. 页面、状态、效果图编号与绑定 V1.4

- 332 个 Page ID；
- 2,671 个 State ID / Mockup ID；
- 135 个审查批次；
- 所有独立 PNG 已生成，V1.6 包内正式效果图状态为 `APPROVED`；重新生成或修改后必须重新审查并更新 SHA-256。

旧版 2,909 个状态中有 238 个来源于错误的通用状态套用或重复页面身份。V1.4 删除这些无效绑定，并未删除任何 Feature ID。

每份页面合同必须声明 `visual_kind`、`visual_identity` 与可选的 `parent_page_id`。`OVERLAY` 必须叠加在指定父页面；`COMPONENT_BOARD` 是组件规范板，不是完整页面。不同 Page ID 不得只改文件夹名或标题后复用同一主体。

典型修正：`YL-A-011` 是注册安全验证 Overlay；`YL-A-012` 是独立注册验证码 Page；`YL-A-029` 是通用外部来源引用组件；`YL-A-060` 是项目知识引用组件。

生成不等于批准。只有 Manifest 行状态为 `APPROVED`、PNG 存在且 SHA-256 匹配时，Codex 才能完成该状态视觉实现。
