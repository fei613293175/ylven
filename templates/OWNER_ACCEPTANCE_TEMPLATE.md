# 所有者验收

- Phase: Pxx
- APK: YLVEN-Pxx-test.apk
- Device model:
- Android version:
- Test date:
- Admin environment/URL:
- Result: PENDING

本文件只记录所有者最终结论。Codex 的后台实测、覆盖安装、中文交付文件、截图和校验门禁全部通过后，仍必须等待所有者亲自验收，不能自动改为 APPROVED。

## Required checks

- [ ] APK installs and launches without crash.
- [ ] Current-phase primary user flow completes with real backend data.
- [ ] Loading, empty, validation, network failure and retry states are understandable.
- [ ] No obvious layout overflow on the test device and enlarged font.
- [ ] Current-phase admin menus load, modify permitted configuration and show audit evidence.
- [ ] `原功能清单.md`、`功能完成对比清单.md`、`完整测试清单.md`、`部署证据.md`、`管理后台实测证据.md`、`校验文件_SHA256.txt` 与实际行为一致。

## Issues

| Severity | Feature ID | Page/menu | Steps | Actual | Expected | Screenshot/recording |
|---|---|---|---|---|---|---|

## Approval

Change `- Result: PENDING` to exactly `- Result: APPROVED` only after all P0/P1 issues are resolved. Use `- Result: REJECTED` when another repair release is required.
