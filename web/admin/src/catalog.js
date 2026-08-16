const kinds = new Set(['probe', 'model', 'provider', 'reasoning'])

export function createCatalogDraft(kind, item = {}) {
  if (!kinds.has(kind)) throw new Error('未知目录类型')
  if (kind === 'provider') return {
    id: item.id || '', name: item.name || '', enabled: item.enabled ?? true,
    sort_order: item.sort_order ?? 0, version: item.version ?? 0
  }
  if (kind === 'model') return {
    id: item.id || '', name: item.name || '', provider_id: item.provider_id || '',
    upstream_model: item.upstream_model || item.id || '', purpose: item.purpose || '',
    speed_tier: item.speed_tier || 'balanced', description: item.description || '',
    enabled: item.enabled ?? true, sort_order: item.sort_order ?? 0, version: item.version ?? 0
  }
  if (kind === 'reasoning') return {
    model_id: item.model_id || '', profile_id: item.profile_id || 'auto', label: item.label || '',
    ordinal: item.ordinal ?? 0, enabled: item.enabled ?? true,
    upstream_parameters_text: JSON.stringify(item.upstream_parameters || {}, null, 2), version: item.version ?? 0
  }
  return {
    model_id: item.model_id || '', capability_id: item.capability_id || '', status: item.status || 'passed',
    probe_source: item.probe_source || '', evidence_reference: item.evidence_reference || '',
    probed_at: item.probed_at || new Date().toISOString(), expires_at: item.expires_at || ''
  }
}

function required(value, message) {
  const normalized = String(value || '').trim()
  if (!normalized) throw new Error(message)
  return normalized
}

export function catalogPayload(kind, draft) {
  if (!kinds.has(kind)) throw new Error('未知目录类型')
  if (kind === 'provider') return {
    id: required(draft.id, '请填写供应商 ID'), name: required(draft.name, '请填写供应商名称'),
    enabled: Boolean(draft.enabled), sort_order: Number(draft.sort_order || 0), version: Number(draft.version || 0)
  }
  if (kind === 'model') return {
    id: required(draft.id, '请填写模型 ID'), name: required(draft.name, '请填写模型名称'),
    provider_id: required(draft.provider_id, '请选择供应商'), upstream_model: required(draft.upstream_model, '请填写上游模型标识'),
    purpose: required(draft.purpose, '请填写模型用途'), speed_tier: required(draft.speed_tier, '请选择速度等级'),
    description: String(draft.description || '').trim(), enabled: Boolean(draft.enabled),
    sort_order: Number(draft.sort_order || 0), version: Number(draft.version || 0)
  }
  if (kind === 'reasoning') {
    let parameters
    try { parameters = JSON.parse(draft.upstream_parameters_text || '{}') } catch { throw new Error('上游参数必须是有效 JSON') }
    if (!parameters || Array.isArray(parameters) || typeof parameters !== 'object') throw new Error('上游参数必须是 JSON 对象')
    return {
      model_id: required(draft.model_id, '请选择模型'), profile_id: required(draft.profile_id, '请选择推理档位'),
      label: required(draft.label, '请填写展示名称'), ordinal: Number(draft.ordinal || 0), enabled: Boolean(draft.enabled),
      upstream_parameters: parameters, version: Number(draft.version || 0)
    }
  }
  const payload = {
    model_id: required(draft.model_id, '请选择模型'), capability_id: required(draft.capability_id, '请填写能力 ID'),
    status: required(draft.status, '请选择探测结果'), probe_source: required(draft.probe_source, '请填写探测来源'),
    evidence_reference: required(draft.evidence_reference, '请填写证据引用'), probed_at: required(draft.probed_at, '请填写探测时间')
  }
  if (String(draft.expires_at || '').trim()) payload.expires_at = String(draft.expires_at).trim()
  return payload
}

