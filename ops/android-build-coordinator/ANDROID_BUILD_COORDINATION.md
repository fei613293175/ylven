# 四项目 Android 构建协调方案

## 范围

`ylven`、`stk`、`xmjl`、`tg`、`hhy` 和 `ylven-vpn` 共用服务器时，所有 Gradle、APK、lint、instrumentation、签名和 Android 模拟器任务必须通过 `/usr/local/bin/android-build`。产品开发是否暂停只由当前 `/goal` 与仓库状态决定，本协调器不改变版本范围。

## 强制入口

```bash
android-build doctor
android-build status
android-build run --project ylven --kind build -- ./gradlew assembleDebug
android-build docker-run --project tg --kind build -- -v /opt/tg-android-r01:/workspace:ro orbexa/android-builder:gradle8.9-api35-v1 gradle assembleDebug
```

同一时间只有一个重型 Android 任务。项目锁防止同一项目重入，全局锁防止跨项目竞争。等待超过队列超时返回 75；资源不足或检测到绕过协调器的任务会拒绝启动。

## 隔离边界

- 每个项目使用 `/opt/android-builds/<project>/` 下独立的 run、artifact、Gradle cache 和必要的历史 AVD 目录。
- `docker-run` 自动注入独立 `/gradle-cache`、`/artifacts` 和 `/run/android-build` 挂载，调用方不再自行复用全局 `/root/.gradle`。
- YLVEN 模拟器验收只在 GitHub Actions 执行并以 CI Artifact 为准；服务器不部署 YLVEN 模拟器。其他项目如有历史模拟器任务，仍必须占用同一全局重型锁，禁止与 APK 构建并发。
- 源码、签名材料、缓存、SDK 写路径和 AVD 不得跨项目共享。
- 构建容器默认 4 CPU、6 GiB 内存、6 GiB swap、2048 PIDs；模拟器默认 4 CPU、5 GiB 内存。
- 监控器只告警，不自动杀进程，以避免误伤业务容器。
- 协调器 TSV 日志按周轮转，保留 12 期并压缩，避免长期运行填满系统盘。

## 固定工具链

- 本地 registry 仅监听 `127.0.0.1:5000`，持久目录为 `/opt/android-build-coordinator/registry`。
- 基础镜像 manifest：`sha256:0b4a09d836f3508e68440f231591677704301e42dccec2996fbb88dca7e91738`。
- 构建镜像 ID：`sha256:fdd6bc537b1f3141bdbaf99cb4f0bcdbd0bb388574a9b7783f3ce1cccd3eefb3`。
- 构建镜像 manifest：`sha256:531539ed21dfc6707d71813f28504927b174ef350be12f505736cd5954472351`。
- `android-build doctor` 会同时验证 registry、基础镜像 manifest、构建镜像 manifest 和本地 image ID；任一漂移即失败。
- 每次 `run` / `docker-run` 在获得锁后都会再次执行同一摘要校验，不能靠跳过手工 `doctor` 绕过工具链完整性门禁。

## 变更与回滚

服务器变更前将 `/etc/android-build-coordinator`、`/usr/local/bin/android-build*`、systemd 单元和镜像构建文件复制到 `/opt/android-build-coordinator/backups/<UTC>/`。回滚时恢复同一备份并执行 `systemctl daemon-reload && systemctl restart android-build-monitor.timer`，不要运行 `docker system prune`。

## 交付门禁

每次版本交付必须同时提供：原开发功能清单、完成对比清单、测试清单、真实构建产物与 SHA-256、服务器/后台可访问性证据、安装运行截图和未通过项。任何缺项都只能报告为未完成或外部阻塞。
