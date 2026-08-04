# Codex UI 实施固定提示

你正在实现 YLVEN `YL-DS-1.2.0`。开始任何 UI 工作前：

1. 读取当前 Work Packet；
2. 读取 `contracts/ui-design-tokens.yaml`；
3. 读取绑定的 `ui/pages/.../<PageID>.yaml`；
4. 运行 `.\ylven.ps1 py scripts/20_CHECK_UI_VISUAL_GATE.py --packet <PacketID>`；
5. 门禁未通过时，不得凭聊天或想象实现页面，只能继续非视觉工作；
6. 门禁通过后，只使用公共组件与语义 Token；
7. 效果图只决定视觉，功能只由 Feature ID 决定；
8. 每个 State ID 生成真实截图并完成视觉差异报告；
9. 不得省略 Offline、Error、Empty、Loading、Success、Permission、Rate Limit 等适用状态；
10. 不得因为效果图样例文字而增加、删除或写死功能。

## 阶段 UI 证据固定目录

真实阶段发布前，Codex 必须把视觉证据放在以下固定位置，不得临时改名或散落到桌面：

```text
build/ui-evidence/<Phase>/
├── screenshots/
│   ├── <StateID>.png
│   └── ...
└── VISUAL_DIFF_REPORT.md
```

- 每个绑定 State ID 必须恰好对应一张独立运行截图；文件名必须是 `<StateID>.png`。
- `VISUAL_DIFF_REPORT.md` 必须逐 State ID 记录已批准效果图 SHA、运行截图、设备/主题/字体缩放、差异和处理结果，并包含 `- Result: PASS`。
- `scripts/05_RELEASE_PHASE.ps1` 默认读取 `build/ui-evidence/<Phase>`；缺少任一状态、效果图未 APPROVED、SHA 不一致或存在未解释差异时，阶段发布失败。
- 桌面交付是发布脚本生成的副本，不是视觉证据的权威来源。

## V1.5 页面身份与重复图强制规则

1. 页面主体必须来自当前 Page ID 的 `visual_identity`，不得只改标题或文件夹名后复用其他页面。
2. `YL-A-011` 是注册安全验证 Overlay；`YL-A-012` 是独立注册验证码 Page。
3. `OVERLAY` 必须在 `parent_page_id` 指定父页面上实现；`COMPONENT_BOARD` 只定义组件。
4. 开发前运行 `scripts/26_VALIDATE_GENERATED_MOCKUPS.py` 与 `scripts/27_AUDIT_VISUAL_DUPLICATES.py`。
5. 只有 Manifest 状态为 `APPROVED` 的 PNG 才可作为实现基线；V1.6 包内正式效果图已批准，但最终 APK 仍需真实截图和项目所有者验收。

