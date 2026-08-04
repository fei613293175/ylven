# YLVEN Spec Kit Lite

- 固定 CLI：`v0.15.2`。
- Codex integration：`.agents/skills/`，显式调用 `$speckit-*`。
- `overrides/` 在 bootstrap 后复制到 `.specify/templates/overrides/`。
- `ylven-skills/` 是额外的交付与追踪 Skill，不替换 Spec Kit 核心 Skills。
- 项目只在 P00 创建一次 Constitution；每个大阶段一套 spec/plan/tasks；普通 Bug 不创建第二套规格。
- 升级 Spec Kit 只能在阶段之间的独立分支中进行，并先验证 `specify version`、`integration status` 和模板差异。
