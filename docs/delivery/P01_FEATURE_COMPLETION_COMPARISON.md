# P01 原功能与完成情况对比

- 阶段：P01
- 版本：1.1.0
- 对比基线：`P01_FEATURES_ORIGINAL.md`
- 当前证据：`status/P01_FEATURE_STATUS.yaml`、`docs/evidence/P01-W01.md` 至 `P01-W05.md`

| Feature ID | 原开发功能 | 当前状态 | 完成证据/差异 |
|---|---|---|---|
| P01-001..P01-018 | 邮箱注册、登录、令牌、设备、限流与审计 | IMPLEMENTED | 后端 identity tests、P01-W01..W04；真实邮件/Turnstile 联调须按测试清单验证 |
| P01-019..P01-023 | 邮件、Turnstile、OTP 策略及用户查询 | IMPLEMENTED | 管理 API tests、P01-W04/W05、管理员页面截图 |
| P01-024..P01-028 | 用户状态、RBAC、管理员会话、step-up、模板记录 | IMPLEMENTED | P01-W05、`YL-M-027/037/038/039/040` 运行截图与 CI |

## 完成判定

- 28/28 Feature ID 在阶段状态中为最终状态并有证据引用。
- 自动验证：以本次交付目录的 `CI_PROVENANCE.json` 和 `AUTOMATED_TEST_REPORT.md` 为唯一来源，不接受旧 Run 或手工改名 APK。
- 外部部署差异：`auth.orbexa.cc` 当前无 DNS；`admin.orbexa.cc` 当前指向其他产品，不能冒充 YLVEN 正式环境。该外部事实阻止正式线上验收，不改变代码已通过的实现证据。
