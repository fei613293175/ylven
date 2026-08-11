请完整读取本补充包，不要只根据本条消息自由发挥。

1. 先在真实 YLVEN 仓库执行 `./ylven.ps1 resume`（Windows 使用 `.\ylven.ps1 resume`），确认当前阶段仍为 P03 且 P03 未 CLOSED。
2. 完成当前原 P03 Work Packet并形成干净基线提交；不得关闭 P03、不得推进 P04。
3. 执行：`python scripts/install_change_order.py --repo <真实仓库根目录>`。
4. 读取 `change-orders/CO-P03-001/00_READ_ME_FIRST.md`、全部 docs、contracts、P03-W06～W08 和 replacement page contracts。
5. 只按 `P03-W06 → P03-W07 → P03-W08` 实施，不重新规划整套项目，不创建第二套功能/版本/页面事实源。
6. 用户界面视觉必须像素级依据 replacement mockups；功能依据 Feature 合同。旧 YL-A-019、YL-A-031 不得保留为独立页面。
7. 前后端必须真实运营可用；严禁 Demo、Mock、固定假数据、伪成功和技术调试文案。
8. GitHub Actions和项目所有者验收通过前，不得执行 `close-release P03`。
