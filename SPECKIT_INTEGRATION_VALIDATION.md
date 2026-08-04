# Spec Kit 与 Codex 自动集成校验报告

- 结果：**PASS**
- 固定版本：`v0.15.2`
- Codex 集成：`.agents/skills`，调用格式 `$speckit-<command>`
- 当前打包沙箱执行联网安装：否
- 目标 Windows 仓库执行：`scripts/00_BOOTSTRAP_REPOSITORY.ps1` 自动安装并由 `scripts/03_VERIFY_SPECKIT.ps1` 验证

## 校验项目

| 项目 | 结果 | 证据 |
|---|---:|---|
| `pinned_version` | PASS | expected=v0.15.2; actual=v0.15.2 |
| `windows_bootstrap_contract` | PASS | all required pinned commands present |
| `linux_bootstrap_contract` | PASS | all required pinned commands present |
| `target_runtime_verification` | PASS | version, integration state, core skill and overrides are verified after installation |
| `codex_invocation_contract` | PASS | uses $speckit-* and rejects legacy /speckit.* |
| `ylven_custom_skills` | PASS | speckit/ylven-skills/ylven-phase-delivery/SKILL.md, speckit/ylven-skills/ylven-contract-traceability/SKILL.md |
| `ylven_lite_overrides` | PASS | constitution-template.md, spec-template.md, plan-template.md, tasks-template.md |
| `local_uv_cli_flag_parser` | PASS | uv=/opt/pyvenv/bin/uv; offline help exposes tool-install --force and tool-run/uvx --from |
| `official_source_register` | PASS | official release, CLI, integration and Codex Skills references recorded |

## 真实性边界

本报告验证包内自动安装方法、固定版本、命令参数、Codex Skills 路径、Lite 模板和安装后检查逻辑。它不声称当前 Linux 打包沙箱已经替你的 Windows/Codex 环境完成联网安装。解压到真实仓库后，引导脚本会执行安装；版本、集成状态、核心 Skill 和覆盖模板任一不符合都会立即失败。
