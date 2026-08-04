# YLVEN 开发计划包内容索引 V1.6.0

## 包规模

- 14 个可独立验收阶段：P00～P13；
- 64 个可连续执行的 Work Packet；
- 362 个跨层 Feature ID；
- 1,086 个预定义测试 ID；
- 201 个数据库/基础设施实体；
- 225 个 OpenAPI operation；
- 332 个前端页面、弹层、组件板与设计系统合同；
- 2,671 个独立状态效果图，全部真实生成；
- 135 张快速审查总览板；
- 28 个编号脚本（00～27）；
- 34 份连续编号的详细开发文档；
- Spec Kit v0.15.2 自动安装、Codex Skills、轻量模板与机械化发布。

## 阅读顺序

1. `00_READ_ME_FIRST.md`；
2. `YLVEN_MASTER_DEVELOPMENT_PLAN.md`；
3. `UPGRADE_NOTES_V1.4.md`；
4. `docs/01`～`docs/34`；
5. 当前 `phases/Pxx_*.md` 与 `work-packets/Pxx-Wxx.md`；
6. `contracts/feature-map.yaml`、`contracts/work-packet-map.yaml` 与 UI 合同；
7. `codex/START_HERE_PROMPT.md`。

## 目录用途

| 目录 | 用途 |
|---|---|
| `docs/` | 产品、UI、后台、架构、数据、身份、商业化、性能、部署、视觉审计和流程规范 |
| `contracts/` | Feature、API、数据、页面、状态、效果图、交互、工作包和测试机器合同 |
| `ui/pages/` | 332 份逐页 YAML 合同 |
| `ui/mockups/` | 2,671 张独立状态 PNG |
| `ui/reference-boards/` | 135 张审查总览板；不得替代独立 PNG |
| `ui/visual-review/` | 可按平台、页面、状态与视觉类型筛选的本地审查网页 |
| `work-packets/` | 64 个小版本；每份重复固定 UI 数值和当前页面绑定 |
| `scripts/` | 安装、合同、效果图、去重审计、审批、发布、ZIP 与校验 |
| `speckit/` | Spec Kit Lite 模板覆盖、固定版本与 YLVEN Skills |

## 当前视觉状态

- 效果图审批状态：`{'APPROVED': 2671}`；
- 精确重复审计：PASS；
- 项目所有者审批：尚未批量执行；
- Codex 最终视觉门禁：仍要求 `APPROVED`。


## V1.6 状态闭环入口

- 当前阶段：`CURRENT_PHASE.yaml`
- 当前小版本：`CURRENT_WORK_PACKET.yaml`
- 工作包状态：`contracts/work-packet-map.yaml`
- 工作包历史：`status/WORK_PACKET_HISTORY.jsonl`
- 正式版本总账：`status/RELEASE_LEDGER.jsonl`
- 新会话恢复：`.\ylven.ps1 resume`
- 本机/CI/服务器分工：`contracts/execution-environment.yaml`

<!-- YLVEN_V1_6_STATE -->
## V1.6 new continuity authorities

- `CURRENT_WORK_PACKET.yaml`
- `status/WORK_PACKET_HISTORY.jsonl`
- `status/RELEASE_LEDGER.jsonl`
- `contracts/state-transition-contract.yaml`
- `contracts/execution-environment.yaml`
- `ylven.ps1`
- `scripts/37_PROJECT_STATE.py` through `scripts/42_RUN_PYTHON.ps1`


## V1.6 版本连续性与执行环境

- 新 Codex 会话第一条命令固定为 `.\ylven.ps1 resume`；不得根据聊天中的“下个版本”猜测。
- 每个 Work Packet 必须通过 `.\ylven.ps1 close-packet -Packet <ID>` 关闭；控制器验证 Feature 最终状态、理由、证据、测试和实现 Commit，追加哈希链历史，自动选择下一包，并自动提交状态变更。
- 每个正式 APK 版本只有在精确 CI Artifact、项目所有者 `APPROVED` 后，才能通过 `.\ylven.ps1 close-release -Phase <Pxx>` 关闭；控制器写入发布总账、创建不可变 Tag、自动提交状态并推进下一阶段。
- 仓库公开是项目所有者确认的合法事实；缺少 `.git`、远程为空或没有默认分支由 Bootstrap 自动修复，不再视为业务阻塞。
- 本机为轻量控制端，只要求 Git、PowerShell、uv、Spec Kit、OpenSSH，以及发布时使用的 GitHub CLI；Java、Go、Gradle、ADB、Android SDK、Docker、Node 和系统 Python均不是本机启动门槛。
- GitHub Actions承担构建、模拟器与视觉回归；线上服务器承担 Docker、数据库及 staging 集成。

## V1.6 continuity authorities

- `CURRENT_PHASE.yaml` and `CURRENT_WORK_PACKET.yaml`: current pointers.
- `status/WORK_PACKET_STATUS.yaml`: only packet lifecycle table.
- `status/WORK_PACKET_HISTORY.jsonl`: append-only hash chain.
- `status/RELEASE_LEDGER.jsonl`: append-only owner-approved release ledger.
- `scripts/37_PROJECT_STATE.py`: only state transition controller.
- `contracts/execution-environment.yaml`: thin local, GitHub Actions and SSH server split.
