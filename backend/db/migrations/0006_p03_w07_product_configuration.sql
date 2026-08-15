-- P03-W07: durable consumer copy, home composer tools and model selection data.
CREATE TABLE IF NOT EXISTS system_configs (
  config_key TEXT PRIMARY KEY,
  value JSONB NOT NULL,
  version BIGINT NOT NULL DEFAULT 1,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_configs (config_key, value)
VALUES (
  'home_config',
  jsonb_build_object(
    'announcement', '',
    'featured_project_ids', jsonb_build_array(),
    'composer_tools', jsonb_build_array(
      jsonb_build_object('id','camera','label','拍照','enabled',false,'prompt','请分析我接下来拍摄的内容：'),
      jsonb_build_object('id','image','label','选择图片','enabled',false,'prompt','请分析我接下来选择的图片：'),
      jsonb_build_object('id','file','label','上传文件','enabled',false,'prompt','请分析我接下来上传的文件：'),
      jsonb_build_object('id','image-generation','label','生成图片','enabled',false,'prompt','请帮我生成图片：'),
      jsonb_build_object('id','presentation','label','制作演示','enabled',false,'prompt','请帮我制作演示文稿：'),
      jsonb_build_object('id','deep-research','label','深度研究','enabled',false,'prompt','请帮我深入研究：')
    ),
    'consumer_copy', jsonb_build_object(
      'thinking','正在思考',
      'tool_file_parse','正在阅读文件',
      'tool_image','正在生成图片',
      'tool_presentation','正在制作演示文稿',
      'reconnecting','连接不稳定，正在恢复…',
      'rate_limited','当前请求较多，请稍后再试',
      'provider_error','暂时无法完成回答，请重试',
      'offline','当前网络不可用',
      'content_blocked','这个请求暂时无法处理，请调整后重试'
    )
  )
)
ON CONFLICT (config_key) DO NOTHING;

CREATE TABLE IF NOT EXISTS models (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  description TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS reasoning_profiles (
  model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
  profile_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  PRIMARY KEY (model_id, profile_id),
  UNIQUE (model_id, ordinal),
  CHECK (profile_id IN ('auto','quick','standard','deep'))
);

INSERT INTO models (id, name, enabled, description)
VALUES ('ylven-default', 'YLVEN 默认模型', TRUE, '自动匹配当前可用能力')
ON CONFLICT (id) DO NOTHING;

INSERT INTO reasoning_profiles (model_id, profile_id, ordinal)
VALUES ('ylven-default', 'auto', 0)
ON CONFLICT (model_id, profile_id) DO NOTHING;
