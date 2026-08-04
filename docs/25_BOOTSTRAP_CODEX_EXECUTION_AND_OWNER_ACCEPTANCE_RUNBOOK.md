# 25 初始化、Codex 连续开发与项目所有者验收操作手册

## 1. 项目所有者首次操作

1. 创建空仓库并将本 ZIP 全部内容解压到仓库根目录；
2. 在 Windows PowerShell 进入仓库；
3. 执行：

```powershell
Set-ExecutionPolicy -Scope Process Bypass
.\scripts\00_BOOTSTRAP_REPOSITORY.ps1
```

4. 脚本自动检查或安装 Git、Python、uv，固定安装 Spec Kit v0.15.2，初始化 Codex Skills，部署 YLVEN 模板，安装 CI 模板并校验合同；
5. 重新打开 Codex 当前仓库；
6. 将 `codex/START_HERE_PROMPT.md` 原文发送给 Codex；
7. 后续仅按 `OWNER_ACTIONS.md` 完成真正需要账号所有者执行的配置。

不要将本包作为另一个子目录放在仓库内；所有根目录文件应位于真正项目根目录。

## 2. Codex 首次执行顺序

Codex 必须：

1. 确认仓库根目录；
2. 运行 bootstrap/doctor；
3. 读取固定决策和 P00；
4. 只在 P00 执行一次 `$speckit-constitution`；
5. 对 P00 执行 `$speckit-specify`、`$speckit-plan`、`$speckit-tasks`；
6. 立即进入 `$speckit-implement`，不能不断重新规划；
7. 按垂直切片交付可运行代码；
8. 更新 Feature Status；
9. 执行真实测试和发布脚本；
10. 返回 APK、后台入口、完成项、问题和 Owner Actions，然后停止。

## 3. 每个后续阶段

建议一个大阶段使用一个新 Codex 会话，避免早期对话压缩和历史噪声。新会话只需粘贴 `codex/CONTINUE_CURRENT_PHASE_PROMPT.md`。Codex从根目录机器状态而不是聊天历史恢复上下文。

每阶段原则：

- 先读取当前阶段，不重读所有历史聊天；
- Spec Kit 只生成当前阶段规格/计划/任务；
- 普通修复直接在当前 tasks 下执行；
- 两轮没有代码、测试或合同变化时停止泛化检查并给出具体阻塞；
- 外部配置缺失时完成适配器、mock、测试和待办，不阻塞无关功能；
- 不自动进入下一阶段。

## 4. 项目所有者阶段验收

桌面目录默认：

```text
Desktop/YLVEN_交付/Pxx/
```

正式权威目录：

```text
dist/releases/Pxx/
```

项目所有者主要检查：

1. 安装 `YLVEN-Pxx-test.apk`；
2. 按 `OWNER_ACCEPTANCE.md` 执行真机步骤；
3. 体验管理后台当前阶段菜单；
4. 对照 `FEATURES.md` 查看完成/推迟/外部阻塞；
5. 对照 `TESTS.md` 查看真实命令和结果；
6. 检查 `OWNER_ACTIONS.md` 是否只包含必要外部操作；
7. 通过时把结果改为 `APPROVED`；不通过时记录复现步骤、截图、设备和预期。

项目所有者不需要检查每一行 Go/Kotlin/Vue 代码；机器合同、自动测试和构建负责基础质量，项目所有者重点验证真实产品体验和业务方向。

## 5. 验收失败

将 `codex/FIX_ACCEPTANCE_ISSUES_PROMPT.md` 发送给新或当前 Codex 会话，并附上：

- 阶段；
- Feature ID（能定位时）；
- 设备型号、Android 版本；
- 页面和操作步骤；
- 实际结果；
- 预期结果；
- 截图/录屏；
- 后台对应异常；
- 是否阻断阶段。

Codex 应修复同一阶段、重新构建和覆盖发布目录，不创建新的无意义小版本或治理恢复任务。

## 6. 验收通过并推进

项目所有者在交付文件写入：

```text
- Result: APPROVED
```

然后让 Codex使用 `codex/ADVANCE_AFTER_OWNER_ACCEPTANCE_PROMPT.md`。阶段状态脚本会验证批准文件和顺序，随后只推进到紧邻的下一阶段。

## 7. 项目所有者需要准备的外部资源

当前已知：

- `orbexa.cc` Cloudflare DNS 管理权限；
- staging/production R2 Bucket 和访问凭据；
- Turnstile staging/production Site Key 与 Secret；
- 邮件发送服务、发件域名和 API/SMTP Secret；
- 已部署 Sub2API Base URL 和受控管理凭据；
- Linux/Docker 部署环境；
- 后期支付渠道、官方 API Key（进入商业生产前）。

所有 Secret 只写入环境或 Secret Manager，不通过聊天、Markdown、Git 或 APK 保存完整值。

## 8. 何时不应该继续自动开发

Codex 只在以下情况停止并返回明确报告：

- 需要项目所有者完成不可替代的外部账号操作；
- 必需的产品决策在现有文档中相互矛盾且会造成不可逆财务/数据影响；
- 真实编译或测试暴露无法在当前仓库修复的外部故障；
- 当前阶段已经发布并等待项目所有者真机/后台验收。

普通编译错误、测试失败、依赖问题、页面 Bug、接口不一致或文档更新都由 Codex继续解决，不应转交项目所有者。
