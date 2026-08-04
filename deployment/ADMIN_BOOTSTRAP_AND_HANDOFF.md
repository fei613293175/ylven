# 管理后台首个账号与一次性密码交付

## 交付版本

- P00 / App 1.0.0：后台工程壳和健康检查。
- **P01 / App 1.1.0：首次交付可访问后台、管理员邮箱和一次性密码。**

## 安全流程

1. 项目所有者在本机或 staging 私密环境设置 `OWNER_ADMIN_EMAIL`，不得提交 `.env`。
2. Codex 通过后端 bootstrap 管理命令幂等创建管理员。
3. 命令生成至少 24 字符的随机一次性密码；数据库只保存强密码哈希。
4. 一次性密码只写入本机桌面 `YLVEN-Releases/1.1.0/ADMIN_ACCESS_ONE_TIME.txt`，文件权限限制为当前用户。
5. 该文件不得进入 Git、GitHub Actions Artifact、日志、聊天、截图或部署报告。
6. 首次登录必须强制修改密码并失效 bootstrap 凭据；后台随后建议开启 MFA。
7. `ADMIN_ACCESS.md` 可以进入交付包，但只记录 URL、管理员邮箱、创建时间和密码交付状态，不包含密码明文。
