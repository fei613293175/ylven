# Android Build Coordinator v1.3.0 验证报告

验证日期：2026-08-05（Asia/Shanghai）  
服务器：`obx-test` / `ebs-178264`  
范围：四个并行开发项目及同服务器历史 Android 项目的 APK 构建协调。YLVEN P01 保持暂停，未恢复产品功能开发。

## 原需求与完成对比

| 原需求/风险 | 优化前 | 当前结果 | 状态 |
|---|---|---|---|
| 跨项目 APK 构建不能并发抢资源 | 各项目可直接启动 Gradle/Docker | 全部重型任务共用 `/run/lock/orbexa-android-build.lock`；第二任务超时返回 75 | 完成 |
| 构建环境不能随机缺失或漂移 | 依赖本地 tag 和历史 SDK | 基础镜像、构建镜像均进入仅本机 registry，并锁定 manifest；`doctor` 校验本地 image ID | 完成 |
| Gradle 缓存和构建产物不能跨项目污染 | STK、TG、Hhy 曾共享缓存或源码路径 | 六个项目分别登记；`docker-run` 自动挂载项目独立 cache、run 和 artifacts | 完成 |
| 构建不能拖垮线上服务 | 历史容器无统一资源上限 | 默认 4 CPU、6 GiB 内存/同值 swap、2048 PIDs；启动前检查负载、可用内存和磁盘 | 完成 |
| 绕过协调器必须可发现 | 无统一告警 | systemd timer 每分钟检测并记录；只告警，不自动杀进程 | 完成 |
| 空闲时监控器必须稳定 | `pgrep` 无匹配时被 `pipefail` 误判失败 | 空进程分支修复；空闲记录 `OK`，systemd 返回 `success/0` | 完成 |
| 现有项目入口需要兼容 | XMJL/TG 自己管理锁和缓存，STK 使用手工容器 | XMJL/TG 已接入全局锁；STK 基础镜像改为本地摘要；统一入口和全局门禁覆盖手工任务 | 完成 |
| YLVEN 模拟器验收位置 | GitHub Actions 是项目合同权威 | 不在服务器部署 YLVEN 模拟器；本次尝试已取消，未生成镜像，相关入口已删除 | 完成（范围纠正） |
| 版本交付材料不能缺失 | 过去曾缺少原功能/完成对比/测试清单 | 全局 Codex 门禁和本报告均要求三张清单、APK SHA-256、后台/API、安装截图和未通过项 | 完成 |

## 固定工具链证据

- 基础镜像 manifest：`sha256:0b4a09d836f3508e68440f231591677704301e42dccec2996fbb88dca7e91738`
- 构建镜像 ID：`sha256:fdd6bc537b1f3141bdbaf99cb4f0bcdbd0bb388574a9b7783f3ce1cccd3eefb3`
- 构建镜像 manifest：`sha256:531539ed21dfc6707d71813f28504927b174ef350be12f505736cd5954472351`
- 本地 registry：`127.0.0.1:5000`，仅服务器本机可访问，容器上限 0.5 CPU、256 MiB、256 PIDs。
- 构建镜像包含 Java 21、Gradle 8.9、Android API 35/36/37、Build Tools 34.0.0/36.0.0 和 platform-tools。

## 测试清单

| ID | 测试 | 预期 | 实际 | 结果 |
|---|---|---|---|---|
| COORD-001 | `bash -n` 检查协调器和监控器 | 无语法错误 | 无错误 | PASS |
| COORD-002 | `android-build doctor` | registry、image ID、两个 manifest、CPU/内存/磁盘全部通过 | `ANDROID_DOCTOR_PASS version=1.3.0` | PASS |
| COORD-003 | `android-build status` | 全局锁空闲，无重型进程/容器 | `global_lock=free`，两个列表为空 | PASS |
| COORD-004 | builder 工具链冒烟 | Gradle 8.9、SDK 文件和容器资源限制存在 | `TOOLCHAIN_RESOURCE_SMOKE_PASS` | PASS |
| COORD-005 | `docker-run` 环境注入 | 容器收到 project/kind/run/artifact/cache 环境 | 全部环境变量存在 | PASS |
| COORD-006 | `docker-run` 独立挂载可写性 | `/artifacts`、`/gradle-cache`、`/run/android-build` 可写 | `DOCKER_RUN_ISOLATION_PASS` | PASS |
| COORD-007 | 容器资源限制 | 内存上限 6 GiB，CPU/PID 限制存在 | memory limit `6442450944` 字节 | PASS |
| COORD-008 | 跨项目并发锁 | 第一个任务运行时第二个任务不能启动 | 第二个任务 `REMOTE_RC=75` | PASS |
| COORD-009 | 绕过协调器检测 | 伪 Gradle 进程产生告警 | `UNCOORDINATED processes=1 containers=0` | PASS |
| COORD-010 | 监控器空闲恢复 | 无重型任务时写入 OK，systemd 成功 | `OK ... processes=0 containers=0`，`Result=success` | PASS |
| COORD-011 | timer 持久状态 | timer enabled 且 active | `enabled` / `active` | PASS |
| COORD-012 | XMJL/TG 脚本语法 | 修改后的脚本可由 Bash 解析 | 三个脚本 `bash -n` 通过 | PASS |
| COORD-013 | YLVEN 服务器模拟器清理 | 无 runner、无 Dockerfile、无镜像、无残留进程 | 四项均不存在 | PASS |
| COORD-014 | 线上业务容器保护 | 不停止、不删除其他项目容器 | 现有业务容器保持运行 | PASS |
| COORD-015 | 日志轮转配置 | TSV 日志每周轮转、压缩并保留 12 期 | `logrotate -d` 无配置错误 | PASS |
| COORD-016 | 每任务摘要硬门禁 | 即使未手工运行 doctor，任务也先校验三层镜像证据 | `ANDROID_TOOLCHAIN_INTEGRITY_PASS` 后才进入 preflight | PASS |
| COORD-017 | 本地运维源与服务器文件一致 | 五个已部署文件 SHA-256 一致 | coordinator、monitor、两个 Dockerfile、logrotate 全部一致 | PASS |

## 当前限制与后续使用规则

- Root 用户技术上仍可绕过任意脚本；通过全局 Codex 门禁、项目入口和每分钟监控告警治理，监控器不会擅自杀掉未知业务进程。
- XMJL/TG 旧镜像暂不强制切换到新 builder；应在各项目下一次正式构建中逐一验证后迁移，避免基础设施优化破坏已工作的发布链。
- GitHub Actions 的 YLVEN 模拟器、视觉回归和 Artifact 验收仍是版本交付必过门禁；本报告不替代 P01 的功能测试、后台测试、APK 安装或所有者真机验收。
- P01 当前代码和交付状态仍为未完成；本次只完成共享 Android 构建基础设施优化，不得据此宣称 P01 完成。

## 回滚点

服务器备份：`/opt/android-build-coordinator/backups/20260805T162325Z-global-optimization/`。  
禁止使用 `docker system prune`。需要回滚时只恢复该备份内明确列出的协调器、监控器、项目配置和 STK Dockerfile，并重载 systemd。
