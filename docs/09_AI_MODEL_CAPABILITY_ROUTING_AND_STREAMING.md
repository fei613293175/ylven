# 09 AI 模型、能力、路由与流式运行时

## 1. 动态模型目录

模型不能写死在 APK。后台每个模型至少包含：

- 内部模型 ID；
- 上游模型 ID；
- 供应商和通道；
- 展示名称、描述、排序和状态；
- 文本、视觉、文件、工具、联网、图片、音频、结构化输出能力；
- 推理控制模式和支持档位；
- 上下文、输出限制和超时；
- 价格版本；
- 已验证能力和最后探测时间；
- 灰度和套餐权限。

## 2. 推理强度映射

前端统一使用用户可理解的标签：`快速、标准、深度、极致、专业`。后台针对每个模型映射到真实参数；未支持的档位不显示。

映射示例仅用于说明，实际以能力探测为准：

```yaml
ui_level: deep
provider: openai
request:
  reasoning.effort: high
```

```yaml
ui_level: deep
provider: anthropic
request:
  thinking_mode: adaptive
  effort: high
```

```yaml
ui_level: deep
provider: xai
request:
  reasoning_effort: high
```

不得把同一个字段和值原样发送给所有供应商。

## 3. 能力探测

P00 必须对每个当前 Sub2API 模型执行：

- 普通文本；
- SSE 流式；
- 请求取消；
- 推理档位；
- 图片输入；
- PDF；
- 工具调用；
- 结构化 JSON；
- 图片生成/编辑；
- 超时、限流和认证错误。

探测结果保存历史，并由后台展示。App 只展示最近探测成功且配置允许的能力。

## 4. 路由策略

路由顺序：

1. 用户明确选择的模型；
2. 套餐和权限校验；
3. 模型能力匹配；
4. 通道健康和并发；
5. 测试/生产环境；
6. 路由目标优先级；
7. 用户授权的回退策略。

禁止静默跨模型回退。可以在同一模型的多个等价通道之间自动切换，但必须保存实际通道。

## 5. 供应商隔离

OpenAI、Anthropic、xAI 和 Sub2API 分别配置：

- 并发池；
- 超时；
- 重试；
- 熔断；
- 限流；
- 连接和响应指标。

某一家限流不能占满全部 AI Runtime 连接。

## 6. 流式协议

Android 与 AI Runtime 使用 SSE。事件至少包括：

- `run.created`
- `message.started`
- `content.delta`
- `tool.started`
- `tool.delta`
- `tool.completed`
- `artifact.created`
- `usage.updated`
- `message.completed`
- `run.failed`
- `run.cancelled`
- `heartbeat`

每个事件有单调递增序号。Redis 保存短期事件缓存，断线后客户端通过 `after` 游标续传。回答结束后将最终结构化消息写 PostgreSQL，不逐 Token 写库。

## 7. 上下文管理

上下文编译器按模型预算选择：

- 系统/项目/用户指令；
- 最近消息；
- 会话摘要；
- 相关项目知识；
- 当前附件；
- 工具结果。

超长上下文先压缩非关键历史并保留引用，不直接截断当前用户问题或关键系统规则。

## 8. 多模型对比与接力

- 对比任务为每个模型创建独立 Run，共享 comparison_group_id。
- 手机端以标签页切换答案，不做三列挤压。
- 用户可选择某个答案继续、让其他模型审查或生成综合答案。
- 接力工作流每一步保存输入来源和输出，不把内部思维链作为下一模型输入。

## 9. 计量

每次 Run 保存输入、输出、缓存、推理 Token（如可用）、上游成本、销售价格、倍率、价格版本和最终扣费。历史费用不能使用当前价格重新计算。
