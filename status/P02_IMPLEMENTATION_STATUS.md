# P02 实施状态

- 阶段：P02 — Android 注册登录与账号安全体验
- 当前状态：IN_PROGRESS（代码与合同已完成，精确构建、现网部署和桌面交付仍待完成）
- 最后更新：2026-08-07

## 已完成的纵向切片

已完成身份注册登录纵向链路：本地登录恢复、应用内一次性算式验证、登录/注册邮箱验证码、逐项密码规则提示、加密保存登录信息、注册后自动登录、登录状态更新/退出/设备管理和网络错误恢复。算式验证限时、一次性、答案摘要存储并限制五次错误；它是基础拦截，不宣称具备第三方高级风控能力。

## 当前进行中

- 等待包含本轮 P02 修复的精确 HEAD 通过 GitHub Actions Android acceptance，旧 APK 不得替代。
- 后端 first-party 算式验证变更尚未部署到 staging；部署后需要实测 challenge、验证、OTP 和设备聚合接口。

## 外部阻塞

Android build coordinator 已连续两次在 registry 检查阶段失败（`local registry unavailable` / `127.0.0.1:5000` 无法连接）；按所有者规则不再重复该路径。GitHub Actions 是独立的构建证据路径。assets/download 域名和生产第三方凭据仍按 `BLOCKED_EXTERNAL` 记录；所有者验收继续异步进行。

## 关键命令与证据

- `uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/19_VALIDATE_UI_CONTRACTS.py --mode planning`：PASS，332 pages / 2671 states。
- `uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/20_CHECK_UI_VISUAL_GATE.py --packet P02-W01`：PASS，10 pages。
- `uv run --python 3.12 --with-requirements requirements-tools.txt python scripts/07_VALIDATE_CONTRACTS.py --phase P02`：PASS，26 features / 1086 tests。
- `ssh obx-test "/usr/local/go/bin/go test ./..."`（独立源码目录）：PASS；Turnstile CSP/回调修复 Commit `25150c4` 已通过全量 Go 测试和 Docker 构建内测试。
- `ssh obx-test "android-build docker-run ..."`：两次在协调器 registry 预检失败；不作为代码失败，也不作为构建通过。
- 历史 Run 31106248150（`1335e1b`）的 APK 只作历史证据，禁止绑定本轮交付；必须由当前 HEAD 生成新 Artifact。
- 公网 `auth.orbexa.cc`：修复前 CSP nonce 未闭合，导致内联样式和 `turnstileSuccess` 被浏览器拦截；Commit `25150c4` 已修正 nonce 并确保回调先于 Cloudflare API 注册。浏览器复测页面样式和成功回调 PASS。
- P02 旧发布目录的合同校验仍可通过，但其 APK/部署证据属于旧提交，不能证明本轮交付；精确 Artifact 到位后必须重新生成发布目录和校验文件。
- Admin/Developer `npm test && npm run build`：PASS；P02 合同、状态连续性、公开仓库安全和 OpenAPI 漂移门禁：PASS。
