import { computed, onMounted, ref, watch } from 'vue'
import { canAccess } from './router.js'
import { catalogPayload, createCatalogDraft } from './catalog.js'

const pages = [
  { path: '/admin/026', title: '验证码策略', group: '认证与用户', pageId: 'YL-M-026', detail: '时效、频率与冷却配置', endpoint: '/admin/v1/settings/otp-policy' },
  { path: '/admin/027', title: '用户列表', group: '认证与用户', pageId: 'YL-M-027', detail: '用户身份、账号状态与高风险控制', endpoint: '/admin/v1/users' },
  { path: '/admin/034', title: '邮件服务', group: '系统设置', pageId: 'YL-M-034', detail: '邮件服务商、发件人与密钥引用', endpoint: '/admin/v1/settings/email' },
  { path: '/admin/035', title: 'Turnstile', group: '系统设置', pageId: 'YL-M-035', detail: '站点与 Secret 引用', endpoint: '/admin/v1/settings/turnstile' },
  { path: '/admin/036', title: '用户详情', group: '认证与用户', pageId: 'YL-M-036', detail: '身份、设备和会话详情' },
  { path: '/admin/037', title: '角色', group: '管理员与权限', pageId: 'YL-M-037', detail: '角色和细粒度权限', endpoint: '/admin/v1/rbac/roles' },
  { path: '/admin/038', title: '管理员', group: '管理员与权限', pageId: 'YL-M-038', detail: '管理员账号与会话', endpoint: '/admin/v1/auth/sessions' },
  { path: '/admin/039', title: '高风险操作', group: '安全与审计', pageId: 'YL-M-039', detail: '高风险配置二次确认' },
  { path: '/admin/040', title: '邮件模板', group: '通知中心', pageId: 'YL-M-040', detail: '版本化模板与发送记录', endpoint: '/admin/v1/notifications/email-templates' },
  { path: '/admin/018', title: '能力探测', group: '模型与供应商', pageId: 'YL-M-018', detail: '模型能力探测证据与有效期', endpoint: '/admin/v1/model-catalog/capability-probes' },
  { path: '/admin/051', title: '模型目录', group: '模型与供应商', pageId: 'YL-M-051', detail: '模型用途、速度与上游标识', endpoint: '/admin/v1/model-catalog/models' },
  { path: '/admin/062', title: '供应商', group: '模型与供应商', pageId: 'YL-M-062', detail: '供应商分组、状态与排序', endpoint: '/admin/v1/model-catalog/providers' },
  { path: '/admin/063', title: '推理映射', group: '模型与供应商', pageId: 'YL-M-063', detail: '用户档位到上游参数的版本化映射', endpoint: '/admin/v1/model-catalog/reasoning-profiles' },
  { path: '/admin/064', title: '默认配置', group: '模型与供应商', pageId: 'YL-M-064', detail: '用户默认模型和推理档位可选范围', endpoint: '/admin/v1/model-catalog/models' },
  { path: '/admin/065', title: '会话分支', group: '对话与内容', pageId: 'YL-M-065', detail: '会话当前分支与分支状态', endpoint: '/admin/v1/conversations' },
  { path: '/admin/066', title: '多模型比较', group: 'AI 工具', pageId: 'YL-M-066', detail: '比较候选、采纳和综合运行状态', endpoint: '/admin/v1/comparisons' },
  { path: '/admin/067', title: '服务状态', group: '模型与供应商', pageId: 'YL-M-067', detail: '模型可用性和回退依据', endpoint: '/admin/v1/model-health' },
  { path: '/admin/068', title: '路由策略', group: '模型与供应商', pageId: 'YL-M-068', detail: '模型主备通道与重试上限', endpoint: '/admin/v1/routing-policies' },
  { path: '/admin/069', title: '供应商通道', group: '模型与供应商', pageId: 'YL-M-069', detail: '端点和凭据引用，明文不会回显', endpoint: '/admin/v1/provider-channels' },
  { path: '/admin/070', title: '模型能力', group: '模型与供应商', pageId: 'YL-M-070', detail: '模型能力与上下文限制', endpoint: '/admin/v1/model-catalog/models' },
  { path: '/admin/071', title: '健康探测', group: '模型与供应商', pageId: 'YL-M-071', detail: '模型可用性、延迟和已探测能力', endpoint: '/admin/v1/model-health' },
  { path: '/admin/072', title: '运行策略', group: '模型与供应商', pageId: 'YL-M-072', detail: '供应商并发、超时和熔断配置', endpoint: '/admin/v1/provider-runtime-policies' },
  { path: '/admin/073', title: '用量记录', group: '商业化', pageId: 'YL-M-073', detail: '输入、输出和推理 Token 记录', endpoint: '/admin/v1/usage-events' },
  { path: '/admin/074', title: '模型价格', group: '商业化', pageId: 'YL-M-074', detail: '每百万 Token 的版本化价格快照', endpoint: '/admin/v1/price-snapshots' },
  { path: '/admin/075', title: '运行事件', group: '运维', pageId: 'YL-M-075', detail: '模型健康和可用性事件', endpoint: '/admin/v1/model-health' },
  { path: '/admin/p03-home', title: '首页配置', group: '内容运营', pageId: 'YL-M-045', detail: '首页聚合数据与可变运营配置', endpoint: '/admin/v1/content/home' },
  { path: '/admin/p03-conversations', title: '会话列表', group: '对话与内容', pageId: 'YL-M-046', detail: '会话、归档状态和回收策略', endpoint: '/admin/v1/conversations' },
  { path: '/admin/p03-exports', title: '导出记录', group: '对话与内容', pageId: 'YL-M-056', detail: '会话与消息 Markdown 导出状态', endpoint: '/admin/v1/conversations' },
  { path: '/admin/p03-metrics', title: 'AI 运行', group: '可观测性', pageId: 'YL-M-061', detail: '聊天延迟与稳定错误指标', endpoint: '/internal/metrics/chat' }
]

const navItems = [
  { label: '认证与用户', path: '/admin/027' },
  { label: '管理员与权限', path: '/admin/037' },
  { label: '安全与审计', path: '/admin/039' },
  { label: '系统设置', path: '/admin/034' },
  { label: '通知中心', path: '/admin/040' },
  { label: '模型与供应商', path: '/admin/051' },
  { label: 'AI 工具', path: '/admin/066' },
  { label: '商业化', path: '/admin/073' },
  { label: '运维', path: '/admin/075' },
  { label: '内容运营', path: '/admin/p03-home' },
  { label: '对话与内容', path: '/admin/p03-conversations' },
  { label: '可观测性', path: '/admin/p03-metrics' }
]

export default {
  setup() {
    const path = ref(window.location.hash.slice(1) || '/admin/027')
    const token = ref(sessionStorage.getItem('ylven_admin_token') || '')
    const permissions = computed(() => token.value ? ['admin:read'] : [])
    const username = ref('')
    const credential = ref('')
    const stepUpCredential = ref('')
    const stepUpToken = ref('')
    const stepUpOpen = ref(false)
    const roleEditor = ref(false)
    const role = ref({ id: '', name: '', permissions: 'users:read' })
    const templateDraft = ref(null)
    const settingDraft = ref(null)
    const settingEditor = ref(false)
    const selectedUser = ref(null)
    const catalogEditor = ref(false)
    const catalogDraft = ref(null)
    const catalogDirty = ref(false)
    const catalogQuery = ref('')
    const opsEditor = ref(false)
    const opsDraft = ref(null)
    const loading = ref(false)
    const saving = ref(false)
    const data = ref(null)
    const error = ref('')
    const notice = ref('')
    const current = computed(() => pages.find((page) => page.path === path.value) || pages[0])
    const catalogKind = computed(() => ({ '/admin/018': 'probe', '/admin/051': 'model', '/admin/062': 'provider', '/admin/063': 'reasoning' })[path.value] || '')
    const isCatalogPage = computed(() => Boolean(catalogKind.value))
    const opsKind = computed(() => ({ '/admin/068': 'routing', '/admin/069': 'channel', '/admin/072': 'runtime', '/admin/074': 'price' })[path.value] || '')
    const isOpsPage = computed(() => Boolean(opsKind.value))
    const allowed = computed(() => canAccess(path.value, permissions.value))
    const actionLabel = computed(() => path.value === '/admin/037' ? '新增角色' : path.value === '/admin/040' ? '编辑模板' : path.value === '/admin/039' ? '安全确认' : path.value === '/admin/018' ? '记录探测' : path.value === '/admin/051' ? '新增模型' : path.value === '/admin/062' ? '新增供应商' : path.value === '/admin/063' ? '新增映射' : isOpsPage.value ? '新增配置' : ['/admin/026','/admin/034','/admin/035','/admin/p03-home'].includes(path.value) ? '编辑配置' : '')

    function navigate(next) { if ((catalogEditor.value && catalogDirty.value || opsEditor.value) && !window.confirm('当前修改尚未保存，确定离开吗？')) return; catalogEditor.value = false; catalogDirty.value = false; opsEditor.value = false; opsDraft.value = null; path.value = next; window.location.hash = next }
    function navActive(item) { return current.value.group === item.label }
    function authHeaders(extra = {}) { return { Authorization: `Bearer ${token.value}`, ...extra } }
    async function parseResponse(response) { const body = await response.json(); if (!response.ok) { const reason = new Error(body.error?.message || `HTTP ${response.status}`); reason.code = body.error?.code || ''; reason.status = response.status; throw reason } return body }
    async function login() {
      loading.value = true; error.value = ''
      try {
        const payload = { username: username.value }; payload['password'] = credential.value
        const body = await parseResponse(await fetch('/admin/v1/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) }))
        token.value = body.access_token; sessionStorage.setItem('ylven_admin_token', token.value); credential.value = ''; await load()
      } catch (reason) { error.value = reason.message } finally { loading.value = false }
    }
    async function logout() {
      const currentToken = token.value
      token.value = ''; stepUpToken.value = ''; data.value = null; sessionStorage.removeItem('ylven_admin_token')
      if (currentToken) await fetch('/admin/v1/auth/logout', { method: 'POST', headers: { Authorization: `Bearer ${currentToken}` } }).catch(() => {})
    }
    async function load() {
      data.value = null; error.value = ''; notice.value = ''; roleEditor.value = false; templateDraft.value = null; settingDraft.value = null; settingEditor.value = false; selectedUser.value = null; catalogEditor.value = false; catalogDraft.value = null; catalogDirty.value = false; opsEditor.value = false; opsDraft.value = null
      if (!token.value || !current.value.endpoint) return
      loading.value = true
      try { data.value = await parseResponse(await fetch(current.value.endpoint, { headers: authHeaders() })); if (data.value.templates?.length) templateDraft.value = { ...data.value.templates[0] }; if (data.value.value) settingDraft.value = { ...data.value.value }; if (data.value.config) settingDraft.value = { ...data.value.config } }
      catch (reason) { error.value = reason.message }
      finally { loading.value = false }
    }
    function runAction() { if (isCatalogPage.value) openCatalogEditor(); else if (isOpsPage.value) openOpsEditor(); else if (path.value === '/admin/037') roleEditor.value = true; else if (path.value === '/admin/039') stepUpOpen.value = true; else if (path.value === '/admin/040') document.querySelector('.template-subject')?.focus(); else if (settingDraft.value) settingEditor.value = true }
    async function confirmStepUp() {
      saving.value = true; error.value = ''
      try { const payload = {}; payload['password'] = stepUpCredential.value; const body = await parseResponse(await fetch('/admin/v1/security/step-up', { method: 'POST', headers: authHeaders({ 'Content-Type': 'application/json' }), body: JSON.stringify(payload) })); stepUpToken.value = body.step_up_token; stepUpCredential.value = ''; stepUpOpen.value = false; notice.value = '高风险操作确认已通过。' }
      catch (reason) { error.value = reason.message } finally { saving.value = false }
    }
    async function highRiskRequest(url, method, body) {
      if (!stepUpToken.value) { stepUpOpen.value = true; throw new Error('请先完成高风险操作确认') }
      const response = await fetch(url, { method, headers: authHeaders({ 'Content-Type': 'application/json', 'X-Step-Up-Token': stepUpToken.value }), body: JSON.stringify(body) })
      stepUpToken.value = ''
      return parseResponse(response)
    }
    async function updateUser(user) { saving.value = true; error.value = ''; try { await highRiskRequest(`/admin/v1/users/${user.id}/status`, 'POST', { status: user.status === 'active' ? 'disabled' : 'active' }); await load(); notice.value = '账号状态已更新并写入审计记录。' } catch (reason) { error.value = reason.message } finally { saving.value = false } }
    async function createRole() { saving.value = true; error.value = ''; try { await highRiskRequest('/admin/v1/rbac/roles', 'POST', { id: role.value.id, name: role.value.name, permissions: role.value.permissions.split(',').map((item) => item.trim()).filter(Boolean) }); role.value = { id: '', name: '', permissions: 'users:read' }; await load(); notice.value = '角色已保存。' } catch (reason) { error.value = reason.message } finally { saving.value = false } }
    async function saveTemplate() { saving.value = true; error.value = ''; try { await highRiskRequest('/admin/v1/notifications/email-templates', 'PUT', templateDraft.value); await load(); notice.value = '邮件模板已保存。' } catch (reason) { error.value = reason.message } finally { saving.value = false } }
    async function saveSetting() { saving.value = true; error.value = ''; try { await highRiskRequest(current.value.endpoint, 'PUT', settingDraft.value); await load(); notice.value = '配置已保存并完成回读。' } catch (reason) { error.value = reason.message } finally { saving.value = false } }
    async function openUser(user) { error.value = ''; try { selectedUser.value = await parseResponse(await fetch(`/admin/v1/users/${user.id}`, { headers: authHeaders() })) } catch (reason) { error.value = reason.message } }
    function openCatalogEditor(item = {}) {
      const defaults = { ...item }
      if (catalogKind.value === 'model' && !defaults.provider_id) defaults.provider_id = data.value?.providers?.[0]?.id || ''
      if (['probe', 'reasoning'].includes(catalogKind.value) && !defaults.model_id) defaults.model_id = data.value?.models?.[0]?.id || ''
      catalogDraft.value = createCatalogDraft(catalogKind.value, defaults); catalogEditor.value = true; catalogDirty.value = false; error.value = ''; notice.value = ''
      if (!stepUpToken.value) stepUpOpen.value = true
    }
    function closeCatalogEditor() { if (catalogDirty.value && !window.confirm('放弃尚未保存的修改吗？')) return; catalogEditor.value = false; catalogDraft.value = null; catalogDirty.value = false }
    async function saveCatalog() {
      saving.value = true; error.value = ''; notice.value = ''
      try {
        const kind = catalogKind.value
        const endpoint = { probe: '/admin/v1/model-catalog/capability-probes', model: '/admin/v1/model-catalog/models', provider: '/admin/v1/model-catalog/providers', reasoning: '/admin/v1/model-catalog/reasoning-profiles' }[kind]
        const method = kind === 'probe' ? 'POST' : 'PUT'
        await highRiskRequest(endpoint, method, catalogPayload(kind, catalogDraft.value))
        await load(); notice.value = kind === 'probe' ? '探测证据已记录，历史结果保持不变。' : '目录配置已保存并完成版本回读。'
      } catch (reason) {
        error.value = reason.code === 'version_conflict' ? '配置已被其他管理员更新，请重新加载后比较版本。' : reason.message
      } finally { saving.value = false }
    }
    function openOpsEditor(item = {}) {
      const defaults = {
        routing: { id: item.id || '', model_id: item.model_id || '', primary_channel_id: item.primary_channel_id || '', fallback_channel_ids_text: (item.fallback_channel_ids || []).join(', '), max_attempts: item.max_attempts ?? 2, enabled: item.enabled ?? true, version: item.version ?? 0 },
        channel: { id: item.id || '', provider_id: item.provider_id || '', name: item.name || '', endpoint: item.endpoint || '', credential_reference: item.credential_reference || '', priority: item.priority ?? 1, enabled: item.enabled ?? true, version: item.version ?? 0 },
        runtime: { provider_id: item.provider_id || '', max_concurrency: item.max_concurrency ?? 4, timeout_seconds: item.timeout_seconds ?? 60, circuit_threshold: item.circuit_threshold ?? 3, cooldown_seconds: item.cooldown_seconds ?? 30, enabled: item.enabled ?? true, version: item.version ?? 0 },
        price: { id: item.id || '', model_id: item.model_id || '', input_per_million: item.input_per_million ?? 0, output_per_million: item.output_per_million ?? 0, reasoning_per_million: item.reasoning_per_million ?? 0, effective_at: item.effective_at || new Date().toISOString(), version: item.version ?? 0 }
      }
      opsDraft.value = defaults[opsKind.value]; opsEditor.value = true; error.value = ''; notice.value = ''; if (!stepUpToken.value) stepUpOpen.value = true
    }
    async function saveOps() {
      saving.value = true; error.value = ''; notice.value = ''
      try {
        const kind = opsKind.value; const draft = { ...opsDraft.value }
        if (kind === 'routing') { draft.fallback_channel_ids = draft.fallback_channel_ids_text.split(',').map((item) => item.trim()).filter(Boolean); delete draft.fallback_channel_ids_text }
        const endpoint = { routing: '/admin/v1/routing-policies', channel: '/admin/v1/provider-channels', runtime: '/admin/v1/provider-runtime-policies', price: '/admin/v1/price-snapshots' }[kind]
        await highRiskRequest(endpoint, 'POST', draft); await load(); notice.value = '配置已保存并写入审计记录。'
      } catch (reason) { error.value = reason.code === 'version_conflict' ? '配置已被其他管理员更新，请重新加载后再保存。' : reason.message } finally { saving.value = false }
    }
    function formatDate(value) { return value ? new Date(value).toLocaleString() : '—' }
    function matchesCatalog(item) { const query = catalogQuery.value.trim().toLowerCase(); return !query || JSON.stringify(item).toLowerCase().includes(query) }

    onMounted(load); watch(path, load)
    return { navItems, path, token, username, credential, stepUpCredential, stepUpToken, stepUpOpen, roleEditor, role, templateDraft, settingDraft, settingEditor, selectedUser, catalogEditor, catalogDraft, catalogDirty, catalogQuery, catalogKind, isCatalogPage, opsKind, isOpsPage, opsEditor, opsDraft, current, allowed, actionLabel, loading, saving, data, error, notice, navigate, navActive, login, logout, load, runAction, confirmStepUp, updateUser, createRole, saveTemplate, saveSetting, openUser, openCatalogEditor, closeCatalogEditor, saveCatalog, openOpsEditor, saveOps, formatDate, matchesCatalog }
  },
  template: `
    <div v-if="!token" class="login-shell"><form class="login-panel" @submit.prevent="login"><div class="login-mark">Y</div><p class="eyebrow">YLVEN ADMIN · P01</p><h1>管理员登录</h1><label>用户名<input v-model="username" type="text" autocomplete="username" required></label><label>密码<input v-model="credential" type="password" autocomplete="current-password" required></label><p v-if="error" class="inline-error">{{ error }}</p><button class="primary" :disabled="loading">{{ loading ? '登录中' : '登录' }}</button></form></div>
    <div v-else class="shell">
      <aside class="sidebar"><div class="brand"><span>Y</span><b>YLVEN ADMIN</b></div><nav><button v-for="item in navItems" :key="item.label" :class="{active:navActive(item)}" @click="navigate(item.path)"><i>·</i>{{ item.label }}</button></nav></aside>
      <section class="workspace-shell"><div class="topbar"><span>YLVEN / {{ current.group }} / {{ current.title }}</span><div><b class="environment">staging</b><button class="topbar-logout" @click="logout">退出</button><span class="avatar">管</span></div></div>
        <main class="content"><div class="page-head"><div><h1>{{ current.title }}</h1><p>{{ current.group }} · {{ current.pageId }}</p></div><button v-if="actionLabel" class="primary action" @click="runAction">{{ actionLabel }}</button></div>
          <div v-if="current.group==='管理员与权限'" class="subnav"><button :class="{active:path==='/admin/037'}" @click="navigate('/admin/037')">角色</button><button :class="{active:path==='/admin/038'}" @click="navigate('/admin/038')">管理员</button></div>
          <p v-if="notice" class="notice">{{ notice }}</p><p v-if="error" class="error-banner">{{ error }}</p>
          <section v-if="!allowed" class="empty-state"><h2>权限不足</h2><p>当前管理员没有访问此资源的权限。</p></section>
          <section v-else-if="loading" class="loading-panel"><div class="skeleton"></div><div class="skeleton short"></div></section>
          <template v-else>
            <div v-if="['/admin/027','/admin/037','/admin/038','/admin/p03-conversations'].includes(path)" class="filter-bar"><input placeholder="搜索名称或 ID"><button>状态：全部</button><button>类型：全部</button><button>最近 30 天</button></div>
            <div v-if="isOpsPage" class="catalog-workspace">
              <div class="filter-bar catalog-filter"><span>{{ (data?.items || []).length }} 条记录</span><button type="button" @click="load">刷新</button></div>
              <form v-if="opsEditor" class="catalog-editor" @submit.prevent="saveOps">
                <div class="catalog-editor-head"><div><h2>{{ opsKind === 'routing' ? '路由策略' : opsKind === 'channel' ? '供应商通道' : opsKind === 'runtime' ? '运行策略' : '模型价格快照' }}</h2><p>保存前需要二次确认；服务端会校验版本、端点和凭据引用，冲突不会静默覆盖。</p></div><button type="button" class="secondary" @click="opsEditor=false">取消</button></div>
                <div v-if="opsKind==='routing'" class="catalog-fields"><label>策略 ID<input v-model.trim="opsDraft.id" required></label><label>模型 ID<input v-model.trim="opsDraft.model_id" required></label><label>主通道 ID<input v-model.trim="opsDraft.primary_channel_id" required></label><label>备用通道 ID（逗号分隔）<input v-model.trim="opsDraft.fallback_channel_ids_text"></label><label>最大尝试次数<input v-model.number="opsDraft.max_attempts" type="number" min="1" max="6" required></label><label class="check-field"><input v-model="opsDraft.enabled" type="checkbox">启用策略</label></div>
                <div v-else-if="opsKind==='channel'" class="catalog-fields"><label>通道 ID<input v-model.trim="opsDraft.id" placeholder="留空自动生成"></label><label>供应商 ID<input v-model.trim="opsDraft.provider_id" required></label><label>通道名称<input v-model.trim="opsDraft.name" required></label><label>优先级<input v-model.number="opsDraft.priority" type="number" min="1" required></label><label class="wide-field">HTTPS 端点<input v-model.trim="opsDraft.endpoint" type="url" required></label><label class="wide-field">凭据引用<input v-model.trim="opsDraft.credential_reference" placeholder="仅安全引用，不填密钥明文" required></label><label class="check-field"><input v-model="opsDraft.enabled" type="checkbox">启用通道</label></div>
                <div v-else-if="opsKind==='runtime'" class="catalog-fields"><label>供应商 ID<input v-model.trim="opsDraft.provider_id" required></label><label>最大并发<input v-model.number="opsDraft.max_concurrency" type="number" min="1" required></label><label>超时（秒）<input v-model.number="opsDraft.timeout_seconds" type="number" min="1" required></label><label>熔断阈值<input v-model.number="opsDraft.circuit_threshold" type="number" min="1" required></label><label>冷却（秒）<input v-model.number="opsDraft.cooldown_seconds" type="number" min="1" required></label><label class="check-field"><input v-model="opsDraft.enabled" type="checkbox">启用运行策略</label></div>
                <div v-else class="catalog-fields"><label>价格 ID<input v-model.trim="opsDraft.id" placeholder="留空自动生成"></label><label>模型 ID<input v-model.trim="opsDraft.model_id" required></label><label>输入每百万 Token<input v-model.number="opsDraft.input_per_million" type="number" min="0" step="0.000001" required></label><label>输出每百万 Token<input v-model.number="opsDraft.output_per_million" type="number" min="0" step="0.000001" required></label><label>推理每百万 Token<input v-model.number="opsDraft.reasoning_per_million" type="number" min="0" step="0.000001" required></label><label class="wide-field">生效时间（ISO 8601）<input v-model.trim="opsDraft.effective_at" required></label></div>
                <div class="catalog-form-actions"><span>版本 {{ opsDraft.version || '新建' }}</span><button class="primary" :disabled="saving">{{ saving ? '保存中' : '保存并回读' }}</button></div>
              </form>
              <div v-if="opsKind==='routing'" class="data-table catalog-table"><div class="table-head"><span>模型</span><span>主通道</span><span>备用通道</span><span>重试</span><span>状态</span><span>操作</span></div><div v-for="item in data?.items || []" :key="item.id" class="table-row"><span>{{ item.model_id }}</span><span>{{ item.primary_channel_id }}</span><span>{{ (item.fallback_channel_ids || []).join('、') || '无' }}</span><span>{{ item.max_attempts }}</span><span :class="['state-label',item.enabled?'active':'disabled']">{{ item.enabled ? '启用' : '停用' }}</span><button class="link-button" @click="openOpsEditor(item)">编辑</button></div><p v-if="!(data?.items || []).length" class="empty-row">暂无路由策略</p></div>
              <div v-else-if="opsKind==='channel'" class="data-table catalog-table"><div class="table-head"><span>通道</span><span>供应商</span><span>端点</span><span>凭据引用</span><span>状态</span><span>操作</span></div><div v-for="item in data?.items || []" :key="item.id" class="table-row"><span><b>{{ item.name }}</b><small>{{ item.id }}</small></span><span>{{ item.provider_id }}</span><code>{{ item.endpoint }}</code><code>{{ item.credential_reference }}</code><span :class="['state-label',item.enabled?'active':'disabled']">{{ item.enabled ? '启用' : '停用' }}</span><button class="link-button" @click="openOpsEditor(item)">编辑</button></div><p v-if="!(data?.items || []).length" class="empty-row">暂无供应商通道</p></div>
              <div v-else-if="opsKind==='runtime'" class="data-table catalog-table"><div class="table-head"><span>供应商</span><span>并发</span><span>超时</span><span>熔断</span><span>状态</span><span>操作</span></div><div v-for="item in data?.items || []" :key="item.provider_id" class="table-row"><span>{{ item.provider_id }}</span><span>{{ item.max_concurrency }}</span><span>{{ item.timeout_seconds }} 秒</span><span>{{ item.circuit_threshold }} / {{ item.cooldown_seconds }} 秒</span><span :class="['state-label',item.enabled?'active':'disabled']">{{ item.enabled ? '启用' : '停用' }}</span><button class="link-button" @click="openOpsEditor(item)">编辑</button></div><p v-if="!(data?.items || []).length" class="empty-row">暂无运行策略</p></div>
              <div v-else class="data-table catalog-table"><div class="table-head"><span>模型</span><span>输入</span><span>输出</span><span>推理</span><span>生效时间</span><span>操作</span></div><div v-for="item in data?.items || []" :key="item.id" class="table-row"><span>{{ item.model_id }}</span><span>{{ item.input_per_million }}</span><span>{{ item.output_per_million }}</span><span>{{ item.reasoning_per_million }}</span><span>{{ formatDate(item.effective_at) }}</span><button class="link-button" @click="openOpsEditor(item)">编辑</button></div><p v-if="!(data?.items || []).length" class="empty-row">暂无价格快照</p></div>
            </div>
            <div v-else-if="path==='/admin/064' || path==='/admin/070'" class="data-table"><div class="table-head"><span>模型</span><span>能力</span><span>用途</span><span>速度</span><span>状态</span></div><div v-for="item in data?.models || []" :key="item.id" class="table-row"><span><b>{{ item.name }}</b><small>{{ item.id }}</small></span><span>{{ (item.capabilities || []).map(capability => capability.label).join('、') || '尚无通过探测的能力' }}</span><span>{{ item.purpose }}</span><span>{{ item.speed_tier }}</span><span :class="['state-label',item.enabled?'active':'disabled']">{{ item.enabled ? '可设为默认' : '已停用' }}</span></div><p v-if="!(data?.models || []).length" class="empty-row">暂无模型默认配置</p></div>
            <div v-else-if="path==='/admin/065'" class="data-table"><div class="table-head"><span>会话</span><span>用户</span><span>当前分支</span><span>模型覆盖</span><span>更新时间</span></div><div v-for="item in data?.conversations || []" :key="item.id" class="table-row"><span><b>{{ item.title }}</b><small>{{ item.id }}</small></span><span>{{ item.user_id }}</span><code>{{ item.active_branch_id || '—' }}</code><span>{{ item.default_model_id || '继承全局' }}</span><span>{{ formatDate(item.updated_at) }}</span></div><p v-if="!(data?.conversations || []).length" class="empty-row">暂无会话分支</p></div>
            <div v-else-if="path==='/admin/066'" class="data-table"><div class="table-head"><span>比较</span><span>会话</span><span>候选模型</span><span>状态</span><span>采纳/综合</span></div><div v-for="item in data?.items || []" :key="item.id" class="table-row"><span><b>{{ item.prompt }}</b><small>{{ item.id }}</small></span><code>{{ item.conversation_id }}</code><span>{{ (item.candidates || []).map(candidate => candidate.model_id).join('、') }}</span><span>{{ item.status }}</span><span>{{ item.adopted_run_id || item.synthesis_run_id || '—' }}</span></div><p v-if="!(data?.items || []).length" class="empty-row">暂无多模型比较</p></div>
            <div v-else-if="path==='/admin/067' || path==='/admin/071' || path==='/admin/075'" class="data-table"><div class="table-head"><span>模型</span><span>供应商</span><span>状态</span><span>延迟</span><span>能力</span></div><div v-for="item in data?.items || []" :key="item.model_id" class="table-row"><span>{{ item.model_id }}</span><span>{{ item.provider_id || '—' }}</span><span :class="['state-label',item.status==='available'?'active':'disabled']">{{ item.status }}</span><span>{{ item.latency_ms }} ms</span><span>{{ (item.capabilities || []).join('、') || '—' }}</span></div><p v-if="!(data?.items || []).length" class="empty-row">暂无健康状态</p></div>
            <div v-else-if="path==='/admin/073'" class="data-table"><div class="table-head"><span>模型</span><span>Run</span><span>输入</span><span>输出</span><span>推理</span><span>时间</span></div><div v-for="item in data?.items || []" :key="item.id" class="table-row"><span>{{ item.model_id }}</span><code>{{ item.run_id }}</code><span>{{ item.input_tokens }}</span><span>{{ item.output_tokens }}</span><span>{{ item.reasoning_tokens }}</span><span>{{ formatDate(item.created_at) }}</span></div><p v-if="!(data?.items || []).length" class="empty-row">暂无用量记录</p></div>
            <div v-else-if="isCatalogPage" class="catalog-workspace">
              <div class="filter-bar catalog-filter"><input v-model="catalogQuery" placeholder="搜索名称、ID 或证据"><button type="button" @click="catalogQuery=''">清除筛选</button><span>{{ catalogKind === 'probe' ? (data?.items || []).filter(matchesCatalog).length : catalogKind === 'model' ? (data?.models || []).filter(matchesCatalog).length : catalogKind === 'provider' ? (data?.providers || []).filter(matchesCatalog).length : (data?.items || []).filter(matchesCatalog).length }} 条</span></div>
              <form v-if="catalogEditor" class="catalog-editor" @submit.prevent="saveCatalog" @input="catalogDirty=true" @change="catalogDirty=true">
                <div class="catalog-editor-head"><div><h2>{{ catalogKind === 'probe' ? '记录能力探测' : catalogKind === 'model' ? '模型配置' : catalogKind === 'provider' ? '供应商配置' : '推理档位映射' }}</h2><p>{{ catalogKind === 'probe' ? '探测结果提交后不可覆盖，请提供可追溯证据。' : '保存时校验当前版本，冲突不会静默覆盖。' }}</p></div><button type="button" class="secondary" @click="closeCatalogEditor">取消</button></div>
                <div v-if="catalogKind==='provider'" class="catalog-fields">
                  <label>供应商 ID<input v-model.trim="catalogDraft.id" :readonly="catalogDraft.version>0" required></label><label>名称<input v-model.trim="catalogDraft.name" required></label><label>排序<input v-model.number="catalogDraft.sort_order" type="number" min="0" required></label><label class="check-field"><input v-model="catalogDraft.enabled" type="checkbox">启用供应商</label>
                </div>
                <div v-else-if="catalogKind==='model'" class="catalog-fields">
                  <label>模型 ID<input v-model.trim="catalogDraft.id" :readonly="catalogDraft.version>0" required></label><label>显示名称<input v-model.trim="catalogDraft.name" required></label><label>供应商<select v-model="catalogDraft.provider_id" required><option value="" disabled>请选择</option><option v-for="provider in data?.providers || []" :key="provider.id" :value="provider.id">{{ provider.name }}</option></select></label><label>上游模型标识<input v-model.trim="catalogDraft.upstream_model" required></label><label>用途<input v-model.trim="catalogDraft.purpose" required></label><label>速度<select v-model="catalogDraft.speed_tier"><option value="fast">快速</option><option value="balanced">均衡</option><option value="deliberate">深度</option></select></label><label>排序<input v-model.number="catalogDraft.sort_order" type="number" min="0"></label><label class="check-field"><input v-model="catalogDraft.enabled" type="checkbox">允许用户选择</label><label class="wide-field">说明<input v-model.trim="catalogDraft.description"></label>
                </div>
                <div v-else-if="catalogKind==='reasoning'" class="catalog-fields">
                  <label>模型<select v-model="catalogDraft.model_id" :disabled="catalogDraft.version>0" required><option value="" disabled>请选择</option><option v-for="model in data?.models || []" :key="model.id" :value="model.id">{{ model.name }}</option></select></label><label>档位<select v-model="catalogDraft.profile_id" :disabled="catalogDraft.version>0"><option value="auto">自动</option><option value="quick">快速</option><option value="standard">标准</option><option value="deep">深度</option></select></label><label>展示名称<input v-model.trim="catalogDraft.label" required></label><label>排序<input v-model.number="catalogDraft.ordinal" type="number" min="0"></label><label class="check-field"><input v-model="catalogDraft.enabled" type="checkbox">启用档位</label><label class="wide-field">上游参数 JSON<textarea v-model="catalogDraft.upstream_parameters_text" spellcheck="false" required></textarea></label>
                </div>
                <div v-else class="catalog-fields">
                  <label>模型<select v-model="catalogDraft.model_id" required><option value="" disabled>请选择</option><option v-for="model in data?.models || []" :key="model.id" :value="model.id">{{ model.name }}</option></select></label><label>能力 ID<input v-model.trim="catalogDraft.capability_id" placeholder="vision" required></label><label>结果<select v-model="catalogDraft.status"><option value="passed">通过</option><option value="failed">未通过</option></select></label><label>探测来源<input v-model.trim="catalogDraft.probe_source" required></label><label class="wide-field">证据引用<input v-model.trim="catalogDraft.evidence_reference" placeholder="evidence://..." required></label><label>探测时间<input v-model.trim="catalogDraft.probed_at" required></label><label>有效期至<input v-model.trim="catalogDraft.expires_at" placeholder="可留空"></label>
                </div>
                <div class="catalog-form-actions"><span>版本 {{ catalogDraft.version || '新建' }}</span><button class="primary" :disabled="saving">{{ saving ? '保存中' : '保存并回读' }}</button></div>
              </form>
              <div v-if="catalogKind==='provider'" class="data-table catalog-table provider-table"><div class="table-head"><span>供应商</span><span>ID</span><span>状态</span><span>模型数</span><span>版本</span><span>操作</span></div><div v-for="item in (data?.providers || []).filter(matchesCatalog)" :key="item.id" class="table-row"><span><b>{{ item.name }}</b></span><code>{{ item.id }}</code><span :class="['state-label',item.enabled?'active':'disabled']">{{ item.enabled ? '启用' : '停用' }}</span><span>{{ (data?.models || []).filter(model => model.provider_id === item.id).length }}</span><span>v{{ item.version }}</span><button class="link-button" @click="openCatalogEditor(item)">编辑</button></div><p v-if="!(data?.providers || []).filter(matchesCatalog).length" class="empty-row">暂无供应商</p></div>
              <div v-else-if="catalogKind==='model'" class="data-table catalog-table model-table"><div class="table-head"><span>模型</span><span>供应商</span><span>用途</span><span>速度/能力</span><span>版本</span><span>操作</span></div><div v-for="item in (data?.models || []).filter(matchesCatalog)" :key="item.id" class="table-row"><span><b>{{ item.name }}</b><small>{{ item.id }}</small></span><span>{{ item.provider_name }}</span><span>{{ item.purpose }}</span><span><b class="speed-tag">{{ item.speed_tier }}</b><small>{{ (item.capabilities || []).map(capability => capability.label).join('、') || '尚无通过探测的能力' }}</small></span><span>v{{ item.version }}</span><button class="link-button" @click="openCatalogEditor(item)">编辑</button></div><p v-if="!(data?.models || []).filter(matchesCatalog).length" class="empty-row">暂无模型</p></div>
              <div v-else-if="catalogKind==='reasoning'" class="data-table catalog-table reasoning-table"><div class="table-head"><span>模型</span><span>档位</span><span>名称</span><span>上游参数</span><span>版本</span><span>操作</span></div><div v-for="item in (data?.items || []).filter(matchesCatalog)" :key="item.model_id+'-'+item.profile_id" class="table-row"><code>{{ item.model_id }}</code><b>{{ item.profile_id }}</b><span>{{ item.label }}</span><code>{{ JSON.stringify(item.upstream_parameters) }}</code><span>v{{ item.version }}</span><button class="link-button" @click="openCatalogEditor(item)">编辑</button></div><p v-if="!(data?.items || []).filter(matchesCatalog).length" class="empty-row">暂无推理映射</p></div>
              <div v-else class="data-table catalog-table probe-table"><div class="table-head"><span>模型/能力</span><span>结果</span><span>来源</span><span>证据</span><span>探测时间</span><span>有效期</span></div><div v-for="item in (data?.items || []).filter(matchesCatalog)" :key="item.id" class="table-row"><span><b>{{ item.model_id }}</b><small>{{ item.capability_label || item.capability_id }}</small></span><span :class="['state-label',item.status==='passed'?'active':'disabled']">{{ item.status === 'passed' ? '通过' : '未通过' }}</span><span>{{ item.probe_source }}</span><code>{{ item.evidence_reference }}</code><span>{{ formatDate(item.probed_at) }}</span><span>{{ formatDate(item.expires_at) }}</span></div><p v-if="!(data?.items || []).filter(matchesCatalog).length" class="empty-row">暂无探测证据</p></div>
            </div>
            <div v-else-if="path==='/admin/027'" class="data-table"><div class="table-head"><span>用户/对象</span><span>ID</span><span>状态</span><span>创建时间</span><span>操作</span></div><div v-for="user in data?.users || []" :key="user.id" class="table-row"><button class="link-button user-link" @click="openUser(user)"><b>{{ user.email }}</b></button><span>{{ user.id }}</span><span :class="['state-label',user.status]">{{ user.status === 'active' ? '正常' : '已禁用' }}</span><span>{{ user.created_at ? new Date(user.created_at).toLocaleString() : '—' }}</span><button class="link-button" @click="updateUser(user)" :disabled="saving">{{ user.status === 'active' ? '禁用' : '恢复' }}</button></div><p v-if="!data?.users?.length" class="empty-row">暂无用户</p></div>
            <div v-else-if="path==='/admin/037'" class="data-table"><form v-if="roleEditor" class="inline-editor" @submit.prevent="createRole"><input v-model="role.id" placeholder="角色 ID" required><input v-model="role.name" placeholder="角色名称" required><input v-model="role.permissions" placeholder="权限，逗号分隔" required><button class="primary" :disabled="saving">保存</button></form><div class="table-head"><span>名称</span><span>ID</span><span>状态</span><span>权限</span><span>操作</span></div><div v-for="item in data?.roles || []" :key="item.id" class="table-row"><span><b>{{ item.name }}</b></span><span>{{ item.id }}</span><span class="state-label active">正常</span><span><code>{{ item.permissions.join(', ') }}</code></span><button class="link-button" @click="stepUpOpen=true">配置</button></div><p v-if="!data?.roles?.length" class="empty-row">暂无角色</p></div>
            <div v-else-if="path==='/admin/038'" class="data-table"><div class="table-head"><span>管理员</span><span>ID</span><span>状态</span><span>角色</span><span>会话</span></div><div v-for="admin in data?.admins || []" :key="admin.id" class="table-row"><span><b>{{ admin.email }}</b></span><span>{{ admin.id }}</span><span class="state-label active">{{ admin.status }}</span><span>{{ admin.role_ids.join(', ') }}</span><span>{{ data?.sessions?.filter((session) => session.admin_user_id === admin.id).length || 0 }}</span></div><p v-if="!data?.admins?.length" class="empty-row">暂无管理员</p></div>
            <div v-else-if="path==='/admin/p03-conversations'" class="data-table"><div class="table-head"><span>会话标题</span><span>ID</span><span>状态</span><span>更新时间</span><span>回收时间</span></div><div v-for="item in data?.conversations || []" :key="item.id" class="table-row"><span><b>{{ item.title }}</b></span><span>{{ item.id }}</span><span :class="['state-label', item.status === 'active' ? 'active' : 'disabled']">{{ item.status }}</span><span>{{ item.updated_at ? new Date(item.updated_at).toLocaleString() : '—' }}</span><span>{{ item.deleted_at ? new Date(item.deleted_at).toLocaleString() : '—' }}</span></div><p v-if="!data?.conversations?.length" class="empty-row">暂无会话</p></div>
            <div v-else-if="path==='/admin/p03-exports'" class="config-panel"><div class="config-title"><h2>导出与保留诊断</h2><span>YL-M-056</span></div><pre class="api-data">{{ JSON.stringify(data, null, 2) }}</pre></div>
            <div v-else-if="path==='/admin/p03-metrics'" class="config-panel"><div class="config-title"><h2>AI 运行指标</h2><span>YL-M-061</span></div><div class="data-table"><div class="table-head"><span>指标</span><span>值</span><span>错误码</span><span>时间</span></div><div v-for="item in data?.metrics || []" :key="item.created_at" class="table-row"><span>{{ item.name }}</span><span>{{ item.value }}</span><span>{{ item.error_code || '—' }}</span><span>{{ item.created_at }}</span></div><p v-if="!data?.metrics?.length" class="empty-row">暂无运行指标</p></div></div>
            <div v-else-if="path==='/admin/039'" class="approval-list"><article v-for="item in ['账号状态变更','角色权限调整','邮件模板发布']" :key="item"><div><span class="risk-tag">高风险</span><h2>{{ item }}需要二次确认</h2><p>确认结果将绑定当前管理员会话并写入审计记录</p></div><button class="outline-danger">拒绝</button><button class="primary" @click="stepUpOpen=true">确认</button></article></div>
            <div v-else-if="path==='/admin/040' && templateDraft" class="config-panel"><div class="config-title"><h2>配置与策略</h2><span>版本 {{ templateDraft.version }}</span></div><form @submit.prevent="saveTemplate"><label><span>模板键</span><input v-model="templateDraft.key" readonly></label><label><span>主题</span><input class="template-subject" v-model="templateDraft.subject" required></label><label><span>正文</span><textarea v-model="templateDraft.body" required></textarea></label><div class="config-footer"><small>发送记录 {{ data.deliveries.length }} 条</small><button class="primary" :disabled="saving">保存配置</button></div></form></div>
            <div v-else-if="settingDraft" class="config-panel"><div class="config-title"><h2>{{ current.title }}</h2><span>{{ current.pageId }}</span></div><form @submit.prevent="saveSetting"><label v-for="(_, key) in settingDraft" :key="key"><span>{{ key }}</span><input v-model="settingDraft[key]" :readonly="!settingEditor" required></label><div class="config-footer"><small>Secret 仅保存引用，不回显明文</small><button v-if="settingEditor" class="primary" :disabled="saving">保存配置</button></div></form></div>
            <div v-else class="config-panel"><div class="config-title"><h2>{{ current.title }}</h2><span>{{ current.pageId }}</span></div><pre v-if="data" class="api-data">{{ JSON.stringify(data, null, 2) }}</pre><p v-else>{{ current.detail }}</p></div>
          </template>
        </main>
      </section>
      <div v-if="stepUpOpen" class="modal-backdrop" @click.self="stepUpOpen=false"><form class="step-up-dialog" @submit.prevent="confirmStepUp"><span class="risk-tag">高风险确认</span><h2>再次验证管理员身份</h2><p>此确认仅用于当前会话中的下一次高风险写操作。</p><label>管理员密码<input v-model="stepUpCredential" type="password" autocomplete="current-password" required autofocus></label><div><button type="button" class="secondary" @click="stepUpOpen=false">取消</button><button class="primary" :disabled="saving">确认</button></div></form></div>
      <div v-if="selectedUser" class="modal-backdrop" @click.self="selectedUser=null"><section class="step-up-dialog user-detail"><span class="state-label active">用户详情</span><h2>{{ selectedUser.user.email }}</h2><p>ID：{{ selectedUser.user.id }}</p><p>状态：{{ selectedUser.user.status }}</p><p>会话：{{ selectedUser.sessions.length }} 个</p><button class="secondary" @click="selectedUser=null">关闭</button></section></div>
    </div>`
}
