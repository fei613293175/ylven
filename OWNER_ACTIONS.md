# OWNER_ACTIONS

当前开发尚未开始。以下资源在对应阶段需要由项目所有者提供；Codex 必须在真正需要时将其拆成可验证的具体条目，而不是要求提前把所有 Secret 写进聊天或仓库。

| 阶段 | Feature ID | 外部资源 | 当前状态 | 安全提供方式 | 验证标准 |
|---|---|---|---|---|---|
| P00/P01 | 待具体绑定 | Cloudflare Turnstile staging Site Key/Secret | NOT_REQUIRED_YET | Secret Manager/服务器环境变量 | 后端 siteverify 测试通过且 Token 不可重放 |
| P00/P01 | 待具体绑定 | 邮件服务与发件域名 | NOT_REQUIRED_YET | Secret Manager/服务器环境变量 | 测试邮箱收到验证码，退信和限流可观测 |
| P00/P05 | 待具体绑定 | Cloudflare R2 staging Bucket/凭据 | NOT_REQUIRED_YET | Secret Manager/服务器环境变量 | 预签名上传、校验、下载与删除通过 |
| P00/P04 | 待具体绑定 | Sub2API Base URL/受控集成凭据 | NOT_REQUIRED_YET | Secret Manager/服务器环境变量 | 能力探测矩阵记录文本、流式、取消、视觉、文件和图片结果 |
| P00/P12 | 待具体绑定 | Linux/Docker staging 主机 | NOT_REQUIRED_YET | SSH Agent/部署平台 Secret | 部署、健康检查、日志和回滚命令通过 |

完整格式见 `docs/24_EXTERNAL_CONFIGURATION_OWNER_ACTIONS_AND_SECRET_HANDLING.md`。
