# P00 真机测试方法清单

## 0. 测试边界

本清单用于项目所有者在真实 Android 手机上验收 P00 `1.0.0`。它验证当前 APK 的 Android 壳、导航和状态恢复；后台合同、Worker、迁移、缓存、事件、存储和 Sub2API 边界仍以 CI/命令行证据为准。不要把 P00 当作登录或聊天版本。

## 1. 准备

- [ ] 使用桌面交付包中的 `YLVEN-1.0.0-P00.apk`，不要使用本地重新编译 APK。
- [ ] 在电脑安装 Android Platform Tools，手机开启开发者选项和 USB 调试。
- [ ] 连接后执行 `adb devices`，确认设备状态为 `device`。
- [ ] 安装：`adb install -r YLVEN-1.0.0-P00.apk`。
- [ ] 确认包名：`adb shell pm list packages | findstr cc.orbexa.ylven`。
- [ ] 记录手机型号、Android 版本、屏幕分辨率、字体大小、系统语言、网络类型和测试时间。

## 2. Android 真机步骤

- [ ] 冷启动：打开 App，确认无崩溃、无伪造进度和明显白屏。
- [ ] 四栏导航：依次点击“首页 / 工作 / 发现 / 我的”，每栏可达、选中态唯一、返回不退出 App。
- [ ] 状态恢复：选中任一栏后旋转屏幕；再将 App 切到后台 30 秒后恢复，确认选中栏和基础布局稳定。
- [ ] 系统重启恢复：从最近任务划掉 App 后重新打开，确认可正常启动；记录是否出现状态丢失。
- [ ] 深色/浅色：切换系统主题各一次，确认文字、图标、底部导航可读且无重叠。
- [ ] 字体放大：将系统字体调至较大档位，确认四栏标签和内容不截断、不遮挡。
- [ ] 弱网：开启飞行模式后重新进入各栏，确认 App 不崩溃；恢复网络后再次打开确认可恢复。
- [ ] 返回键：在每个主栏按系统返回键，确认行为稳定且无重复退出。

## 3. 后台合同验证（电脑执行）

在已启动的测试环境中，对每个服务执行：

```powershell
curl.exe -i http://localhost:8080/healthz
curl.exe -i http://localhost:8080/readyz
curl.exe -i http://localhost:8080/version
curl.exe -i http://localhost:8080/internal/metrics
```

- [ ] 健康、就绪、版本返回稳定合同和当前版本信息。
- [ ] 重复 Worker 请求保持幂等；超过有限重试后进入失败归档。
- [ ] Scheduler 旧 fencing token 被拒绝，租约过期可接管且不重复执行。
- [ ] 未配置 Sub2API/R2 时返回明确 `UNCONFIGURED`/受控错误，不记录 Secret。
- [ ] 运行 `scripts/40_PUBLIC_REPOSITORY_SAFETY.py` 和 OpenAPI drift 检查均通过。

## 4. 结果与证据

- [ ] 保存 `adb logcat -d > P00-logcat.txt`，确认无崩溃堆栈和 Secret。
- [ ] 保存每项失败的复现步骤、截图、设备信息和时间；不要只写“体验正常”。
- [ ] 将结果填写到 `OWNER_ACCEPTANCE.md`，明确 `PASS`、`PASS_WITH_LIMITATIONS` 或 `FAIL`。
- [ ] P00 当前已知限制：自动验收未生成全部 mockup state 的运行时截图；该限制必须在验收结论中保留。
- [ ] 只有所有者确认通过后，才允许后续执行 `close-release`；本次交付不执行该命令。
