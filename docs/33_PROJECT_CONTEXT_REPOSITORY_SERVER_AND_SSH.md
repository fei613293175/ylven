# 33 — 仓库、服务器与 SSH 环境事实

## 固定事实

- 唯一仓库：`https://github.com/fei613293175/ylven.git`；remote 为 `origin`，主分支为 `main`。
- 线上服务器已部署 Docker 和 Android 构建环境。
- 用户已在 Codex 客户端配置服务器 SSH 连接；Codex后续复用该连接。
- 主域名为 `orbexa.cc`；Cloudflare 负责 DNS/CDN，R2 负责对象存储。

## 强制边界

开发包只记录上述事实，不记录 IP、SSH 用户名、密码、私钥路径、API Key、Cloudflare Token 或数据库密码。首次服务器操作必须先做只读盘点；禁止直接升级宿主机、覆盖已有容器、占用未知端口或在服务器中修改未提交源码。

服务器首期角色是 staging。正式生产部署默认关闭，必须在商业渠道、官方 API、支付、安全、备份和恢复验收后单独启用。
