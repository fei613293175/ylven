# 项目所有者最小工作流

## 首次

1. 解压开发包到新仓库根目录。
2. 在 Codex 打开仓库，发送 `codex/START_HERE_PROMPT.md`。
3. 只在 `OWNER_ACTIONS.md` 出现外部账号待办时完成 Cloudflare、邮件、R2、Sub2API 等配置。

## 每个阶段

1. 从桌面 `YLVEN_交付/Pxx/` 或仓库 `dist/releases/Pxx/` 安装 APK。
2. 依据 `templates/APK_ACCEPTANCE_TEMPLATE.md` 测试该阶段关键流程。
3. 打开管理后台，依据阶段文件的“管理后台范围”体验菜单、配置和日志。
4. 把问题按页面、操作、实际结果、预期结果、截图/录屏、时间写入验收记录，发送 `FIX_ACCEPTANCE_ISSUES_PROMPT.md`。
5. 全部通过后保存 `OWNER_ACCEPTANCE.md`，发送 `ADVANCE_AFTER_OWNER_ACCEPTANCE_PROMPT.md`。

项目所有者无需手工运行单元测试、维护数据库迁移、生成清单或复制 APK；这些由脚本和 Codex完成。
