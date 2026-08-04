# YLVEN UI 视觉合同与效果图 V1.4

## 规模

- 页面/组件/设计板：332
- 独立状态效果图：2,671
- 快速审查板：135
- 设计系统：`YL-DS-1.2.0`

## 关键更正

旧版将部分弹层、组件和页面错误套入同一通用模板。V1.5 已重新绑定：`YL-A-011` 是注册安全验证弹层，父页面 `YL-A-010`；`YL-A-012` 是独立六位注册验证码页面。组件规范板不再伪装成重复的完整业务页面。

## 状态与审批

全部 PNG 已生成、SHA-256 已登记，并在 V1.5 视觉审查后于 Manifest 中标记为 `APPROVED`。仅在重新生成或变更效果图后，才需要重新执行批准流程：

```powershell
.\.venv-tools\Scripts\python.exe scripts/25_APPROVE_MOCKUPS.py --all --approved-by OWNER
```

## 权威边界

- 功能：`contracts/feature-map.yaml`；
- 数值：`contracts/ui-design-tokens.yaml` 与逐页合同；
- 视觉：已批准且 SHA 匹配的独立 PNG；
- 共享后台 Shell 合法，但业务主内容不得复用错绑。
