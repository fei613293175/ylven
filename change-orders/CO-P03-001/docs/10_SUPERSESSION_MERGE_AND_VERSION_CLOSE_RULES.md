# 替代、合并与版本关闭

## 替代关系

- `YL-A-019`、`YL-A-031` 不再是可导航页面；旧深链统一进入 `YL-A-023` 草稿状态。
- 本包列出的七个 Android Page ID 的 V1.6 效果图和视觉身份被全部替代。
- 旧图片可备份，但不得继续存在于正式 `mockup-manifest.csv` 的 APPROVED 记录中。

## 合并

安装脚本把变更单写入 `change-orders/CO-P03-001/`，更新指定页面合同/效果图、追加 Feature 与 Work Packet，并把全局生产规则写入 AGENTS、阶段与工作包。Codex随后运行项目原有合同再生成和校验脚本。

## 关闭

P03 完成时必须在发布总账中记录变更单 ID、上下文架构版本、效果图 SHA、CI Run、Artifact 和 APK SHA。下一版完整开发计划包应将本变更单折叠回主合同，然后把 `CO-P03-001` 标记为 MERGED；不得永久把它作为第二套权威来源。
