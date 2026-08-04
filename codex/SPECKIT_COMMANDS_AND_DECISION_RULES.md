# Codex 使用 Spec Kit Lite 的命令和判断规则

## 首次项目

只在 P00 调用一次 `$speckit-constitution`。宪法来自本包固定决策，不通过问答重新选择技术栈。

## 每个重大阶段

```text
$speckit-specify
$speckit-plan
$speckit-tasks
$speckit-implement
```

输入必须明确指定当前阶段文件和 Feature ID 范围。`specify` 写“做什么、为什么、成功标准”；`plan` 写架构、接口、数据、迁移、性能、安全；`tasks` 拆成能够独立验证的纵向切片；`implement` 执行代码和测试。

## 仅按需调用

- `$speckit-clarify`：只有会影响合同、财务、安全或用户体验的关键歧义无法从文档解决时。
- `$speckit-analyze`：身份、支付、钱包、公共 API、迁移等高风险模块在实现前最多一次，或发现合同互相矛盾时。
- `$speckit-checklist`：安全、财务、隐私和最终回归专项。
- `$speckit-converge`：大型阶段末发现范围遗漏时最多一次；结果必须转成具体任务，不得无限收敛。

## 禁止模式

- 每个按钮调整都走完整 Spec Kit 流程；
- 连续运行 clarify/analyze/converge 而没有代码变化；
- 生成第二份不一致的总体产品规划；
- 把 Spec Kit tasks 当成唯一状态源而忽略 `CURRENT_PHASE.yaml` 和 Feature Status；
- 同时安装 Kiro specs 或另一套治理状态机。
