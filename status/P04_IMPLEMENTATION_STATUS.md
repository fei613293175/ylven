# P04 实施状态

- 阶段：P04 - 多模型目录、推理强度、路由与对比回答
- 当前工作包：P04-W01
- 当前状态：DOING
- 最后更新：2026-08-17

## 已证实事实

- `P04-001` 至 `P04-005` 的目录、能力过滤、推理档位和 Android 选择器代码已存在，但尚未完成当前候选的全量验收。
- 精确提交 `ce19b067ae77bb7b4503b42e7096cb694b30742b` 已在连接服务器构建。`testDebugUnitTest`、`lintDebug`、`assembleDebug` 和 `assembleDebugAndroidTest` 均通过。APK SHA-256 为 `43d4e49ba60205acb5a0e4c352e217c3097f25882b5fd6104516783bd520aa92`，instrumentation APK SHA-256 为 `ab43269eff2e1c5452041e19ad30dc17e93cb6a1d9242cd215f765527fbd96e4`。
- 前两次同版本定向真机回归已分别定位到 Xiaomi 的 `Espresso.closeSoftKeyboard()` 动画超时，以及测试发送返回键后使聊天页面退出。两者发生在测试编排而非模型接口或应用崩溃。测试现改为保留输入法焦点、等待发送控件存在后直接点击，并将重新在服务器构建。每次失败设备均已释放：应用与测试进程均不存在，前台已回到 Launcher，队列锁和票据已释放。
- staging 的 `ylven-default` 与 `p04-reasoner` 当前仅有 `auto` 推理档位；没有 p04-reasoner provider channel、routing policy 或 capability probe。因此 `P04-006` 不能以测试夹具中的非自动映射替代真实上游配置验证。

## 未完成项

- 真机：当前候选必须在严格 `device` 状态的物理设备上完成同版本定向回归，包括 provisioning、P04W01 live flow、12 个状态截图、日志/ANR 审查与设备释放。此刻 ADB 在多个项目客户端之间重启后未稳定枚举设备，按规则暂停真机测试，禁止改用模拟器。
- 视觉：重新采集 `YL-A-033` 和 `YL-A-034` 的 12 个真实 Android 状态，并按照 `similarity >= 0.985`、`mismatch <= 0.005` 完成服务器视觉比较和正式审查。旧报告的 12/12 REVIEW 不可作为通过证据。
- 管理后台：`YL-M-018`、`YL-M-051`、`YL-M-062`、`YL-M-063` 共 41 个状态需要真实登录、点击、保存、回读与审计证据。当前仅有前端单元测试；没有可用后台凭据时不可伪造。
- P04-006：需获得有真实上游依据的非自动档位配置，完成健康/能力探测、参数回读和真实请求；否则保持 `BLOCKED_EXTERNAL`。
- Go：前一次服务器 `go test -count=1 ./internal/identity` 被首次模块下载卡住，未得到退出结果，不能报告通过。新的候选工作区已确认可连 `proxy.golang.org`，仍需受控重跑。

## 结论

P04-W01 没有完成，P04-W02 至 P04-W05 尚未开始。旧状态文件中“W01 已完成/真机通过”的表述与当前有效证据矛盾，已撤销。不得执行 `close-packet`、不得清除 P04 目标，也不得把本阶段交付为通过。
