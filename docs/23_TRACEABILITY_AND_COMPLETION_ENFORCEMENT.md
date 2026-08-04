# 23 追踪矩阵与完成判定机械化

## 1. 单一 Feature ID

每个产品能力从需求到交付都使用 `Pxx-nnn`。Feature Map 同时记录页面/菜单、动作、状态、接口或事件、服务、实体、后台控制、权限、计费、任务、验收和测试。代码提交、OpenAPI operation、迁移、测试和发布清单应引用这些 ID。

## 2. 完成不是布尔口号

`IMPLEMENTED` 至少需要：真实代码路径；必要迁移；接口或内部事件；Android/后台状态；权限；日志/指标；自动测试证据；可部署配置。只存在 UI、只存在后端、只存在 mock、只写文档或只运行检查都不能标记 IMPLEMENTED。

`DEFERRED_WITH_REASON` 仅用于项目所有者明确接受的范围延期，必须写原因、影响、目标阶段和兼容措施。`BLOCKED_EXTERNAL` 仅用于不可由仓库解决的 DNS、凭据、供应商故障等，并附 Owner Action 和已经完成的离线部分。

## 3. 自动校验

`scripts/07_VALIDATE_CONTRACTS.py` 校验 YAML/JSON、Feature 唯一性、CSV 对齐、实体、后台菜单、测试、OpenAPI、阶段文档、根状态和发布合同。`--release` 额外要求阶段 Feature 状态全部终态并有证据/原因。`scripts/13_VALIDATE_PACKAGE.py` 校验包内引用、模板、脚本、编码和清单。

## 4. 发布门槛

`05_RELEASE_PHASE.ps1` 构建真实 Gradle APK并运行相关测试，`11_GENERATE_RELEASE.py` 从状态和日志生成交付文件，`06_VERIFY_RELEASE.ps1` 检查 APK、JSON、必需文件和 SHA-256。桌面副本由 `08_SYNC_DESKTOP_DELIVERY.ps1` 从正式 `dist/releases/Pxx` 原子复制。

## 5. 防漂移

每阶段一个新 Codex 会话，只读取当前阶段和直接相关文档。AGENTS 保持短小；详细规则由机器合同与脚本执行。连续两轮无代码、测试或具体阻塞产出时停止泛化检查。Spec Kit 只负责阶段规格/计划/任务，不成为第二套状态机。

## 6. 变更同步

新增或改变功能时，先更新 Feature Map，再更新 API/实体/后台/测试合同和阶段文档，最后实现。自动生成文件通过 `12_REGENERATE_CONTRACTS.py` 重建，禁止手工只改 OpenAPI 而不改来源清单。
