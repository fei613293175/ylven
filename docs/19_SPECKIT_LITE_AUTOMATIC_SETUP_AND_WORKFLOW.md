# 19 Spec Kit Lite 自动安装与 Codex 使用流程

## 1. 选型

YLVEN 使用 GitHub Spec Kit 的 Codex Skills 集成，但不使用 Kiro，也不安装复杂自治/治理预设。固定版本为 `v0.15.2`，避免上游升级在开发中途改变命令和模板。

项目本地使用 `.specify/templates/overrides/` 覆盖规格、计划和任务模板。项目级覆盖优先于预设和核心模板，且不需要维护第二个远程预设仓库。

## 2. 自动安装

Windows 仓库根目录执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\00_BOOTSTRAP_REPOSITORY.ps1
```

脚本负责：

1. 检查 Git、Python、PowerShell；
2. 安装或检查 `uv`；
3. 固定安装 `specify-cli` v0.15.2；
4. 在已有仓库中初始化 Codex integration；
5. 将 Skills 安装到 `.agents/skills/`；
6. 将 YLVEN 模板复制到 `.specify/templates/overrides/`；
7. 将短版 `AGENTS.md`、阶段状态和合同复制到根目录；
8. 执行 Spec Kit、合同和目录自检；
9. 不创建或运行 Governance 状态机。

Linux 脚本用于服务器或非 Windows 开发环境，但 Codex Windows 客户端优先使用 PowerShell。

## 3. Codex 中的调用形式

Codex Skills 使用：

```text
$speckit-constitution
$speckit-specify
$speckit-plan
$speckit-tasks
$speckit-implement
```

可选：

```text
$speckit-clarify
$speckit-analyze
$speckit-checklist
$speckit-converge
```

不得使用另一套 Kiro requirements/design/tasks 作为并行事实来源。

## 4. 第一次项目初始化

Codex 读取总开发包后执行一次：

```text
$speckit-constitution
```

Constitution 只固定长期原则：产品边界、技术栈、数据权威、身份安全、财务强一致、动态模型能力、测试和交付合同。不得把几百个 Feature ID 全部复制进 Constitution。

## 5. 每个阶段的标准流程

### 5.1 Specify

```text
$speckit-specify
```

输入必须引用当前阶段文档和 Feature IDs，生成阶段规格，明确用户故事、正常/异常状态、范围外内容和可验证验收。不得将下一阶段功能顺带加入。

### 5.2 Plan

```text
$speckit-plan
```

计划必须列出 Android 页面/状态、后端模块/API、数据迁移、后台菜单、异步任务、安全/计费、可观测性、测试和回滚。若与总架构冲突，先修计划，不能擅自改架构。

### 5.3 Tasks

```text
$speckit-tasks
```

任务按可交付垂直切片组织，包含 Feature ID、文件路径、依赖和测试；不能把“实现后端”“开发 App”写成无法验收的超大任务，也不能拆成大量无价值微任务。

### 5.4 Implement

```text
$speckit-implement
```

Codex按任务直接实现、测试和提交，不再重复完整分析。发现规格错误时更新对应文档并记录变更，不以反复 Analyze 取代实现。

### 5.5 Release

执行统一发布和校验脚本；项目所有者只做 APK/后台体验。

## 6. 何时使用可选命令

- Clarify：阶段文档确实存在会改变实现的关键歧义；
- Analyze：P01/P10/P11/P12 等跨身份、资金、API 或安全模块，最多在实现前按需一次；
- Checklist：身份安全、支付、账本、数据删除和公共 API；
- Converge：P13 或阶段验收确认有跨文档漏项。

不对按钮间距、文本修改、单一崩溃等小任务运行完整流程。

## 7. 防止 Spec Kit 自身变成负担

- 规划/检查工作量原则上不超过阶段工作量约 20%；
- 文档足以开始实现时立即编码，不追求措辞完美；
- 已有明确决策不得反复询问项目所有者；
- Feature Map、OpenAPI 和测试是机械合同，Spec 文档不重复抄写全部内容；
- 一阶段一个 Spec 主目录，bug 修复在原阶段跟踪，不无限创建新规格；
- Spec Kit 升级仅在阶段之间进行，并先在分支验证。
