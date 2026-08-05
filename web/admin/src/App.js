import { computed, onMounted, ref, watch } from 'vue'
import { canAccess } from './router.js'

const pages = [
  { path: '/admin/018', title: '模型与供应商', group: '模型与供应商', pageId: 'YL-M-018', detail: '能力探测状态可回读' },
  { path: '/admin/019', title: '安全与审计', group: '安全与审计', pageId: 'YL-M-019', detail: '秘密仅显示配置状态' },
  { path: '/admin/020', title: '管理员与权限', group: '管理员与权限', pageId: 'YL-M-020', detail: 'RBAC 策略已加载' },
  { path: '/admin/021', title: '开发者平台', group: '开发者平台', pageId: 'YL-M-021', detail: '开发者平台入口' },
  { path: '/admin/026', title: '验证码策略', group: '认证与用户', pageId: 'YL-M-026', detail: '时效、频率与冷却配置', endpoint: '/admin/v1/settings/otp-policy' },
  { path: '/admin/027', title: '用户列表', group: '认证与用户', pageId: 'YL-M-027', detail: '用户身份、账号状态与高风险控制', endpoint: '/admin/v1/users' },
  { path: '/admin/034', title: '邮件服务', group: '系统设置', pageId: 'YL-M-034', detail: '邮件服务商、发件人与密钥引用', endpoint: '/admin/v1/settings/email' },
  { path: '/admin/035', title: 'Turnstile', group: '系统设置', pageId: 'YL-M-035', detail: '站点与 Secret 引用', endpoint: '/admin/v1/settings/turnstile' },
  { path: '/admin/036', title: '用户详情', group: '认证与用户', pageId: 'YL-M-036', detail: '身份、设备和会话详情' },
  { path: '/admin/037', title: '角色', group: '管理员与权限', pageId: 'YL-M-037', detail: '角色和细粒度权限', endpoint: '/admin/v1/rbac/roles' },
  { path: '/admin/038', title: '管理员', group: '管理员与权限', pageId: 'YL-M-038', detail: '管理员账号与会话', endpoint: '/admin/v1/auth/sessions' },
  { path: '/admin/039', title: '高风险操作', group: '安全与审计', pageId: 'YL-M-039', detail: '高风险配置二次确认' },
  { path: '/admin/040', title: '邮件模板', group: '通知中心', pageId: 'YL-M-040', detail: '版本化模板与发送记录', endpoint: '/admin/v1/notifications/email-templates' }
]

const navItems = [
  { label: '运营总览', path: '/admin/018' }, { label: '认证与用户', path: '/admin/027' },
  { label: '模型与供应商', path: '/admin/018' }, { label: '对话与内容', path: '/admin/021' },
  { label: '文件与存储', path: '/admin/021' }, { label: '项目与知识库', path: '/admin/021' },
  { label: 'AI 工具', path: '/admin/021' }, { label: '商业化', path: '/admin/021' },
  { label: '开发者平台', path: '/admin/021' }, { label: '管理员与权限', path: '/admin/037' },
  { label: '安全与审计', path: '/admin/039' }, { label: '系统设置', path: '/admin/034' },
  { label: '通知中心', path: '/admin/040' }, { label: '质量与发布', path: '/admin/021' }
]

export default {
  setup() {
    const path = ref(window.location.hash.slice(1) || '/admin/027')
    const token = ref(sessionStorage.getItem('ylven_admin_token') || '')
    const permissions = computed(() => token.value ? ['admin:read'] : [])
    const email = ref('')
    const credential = ref('')
    const stepUpCredential = ref('')
    const stepUpToken = ref('')
    const stepUpOpen = ref(false)
    const roleEditor = ref(false)
    const role = ref({ id: '', name: '', permissions: 'users:read' })
    const templateDraft = ref(null)
    const loading = ref(false)
    const saving = ref(false)
    const data = ref(null)
    const error = ref('')
    const notice = ref('')
    const current = computed(() => pages.find((page) => page.path === path.value) || pages[0])
    const allowed = computed(() => canAccess(path.value, permissions.value))
    const actionLabel = computed(() => path.value === '/admin/037' ? '新增角色' : path.value === '/admin/040' ? '编辑模板' : path.value === '/admin/039' ? '安全确认' : '')

    function navigate(next) { path.value = next; window.location.hash = next }
    function navActive(item) { return current.value.group === item.label }
    function authHeaders(extra = {}) { return { Authorization: `Bearer ${token.value}`, ...extra } }
    async function parseResponse(response) { const body = await response.json(); if (!response.ok) throw new Error(body.error?.message || `HTTP ${response.status}`); return body }
    async function login() {
      loading.value = true; error.value = ''
      try {
        const payload = { email: email.value }; payload['password'] = credential.value
        const body = await parseResponse(await fetch('/admin/v1/auth/login', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) }))
        token.value = body.access_token; sessionStorage.setItem('ylven_admin_token', token.value); credential.value = ''; await load()
      } catch (reason) { error.value = reason.message } finally { loading.value = false }
    }
    function logout() { token.value = ''; stepUpToken.value = ''; data.value = null; sessionStorage.removeItem('ylven_admin_token') }
    async function load() {
      data.value = null; error.value = ''; notice.value = ''; roleEditor.value = false; templateDraft.value = null
      if (!token.value || !current.value.endpoint) return
      loading.value = true
      try { data.value = await parseResponse(await fetch(current.value.endpoint, { headers: authHeaders() })); if (data.value.templates?.length) templateDraft.value = { ...data.value.templates[0] } }
      catch (reason) { error.value = reason.message }
      finally { loading.value = false }
    }
    function runAction() { if (path.value === '/admin/037') roleEditor.value = true; else if (path.value === '/admin/039') stepUpOpen.value = true; else if (path.value === '/admin/040') document.querySelector('.template-subject')?.focus() }
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

    onMounted(load); watch(path, load)
    return { navItems, path, token, email, credential, stepUpCredential, stepUpToken, stepUpOpen, roleEditor, role, templateDraft, current, allowed, actionLabel, loading, saving, data, error, notice, navigate, navActive, login, logout, load, runAction, confirmStepUp, updateUser, createRole, saveTemplate }
  },
  template: `
    <div v-if="!token" class="login-shell"><form class="login-panel" @submit.prevent="login"><div class="login-mark">Y</div><p class="eyebrow">YLVEN ADMIN</p><h1>管理员登录</h1><label>邮箱<input v-model="email" type="email" autocomplete="username" required></label><label>密码<input v-model="credential" type="password" autocomplete="current-password" required></label><p v-if="error" class="inline-error">{{ error }}</p><button class="primary" :disabled="loading">{{ loading ? '登录中' : '登录' }}</button></form></div>
    <div v-else class="shell">
      <aside class="sidebar"><div class="brand"><span>Y</span><b>YLVEN ADMIN</b></div><nav><button v-for="item in navItems" :key="item.label" :class="{active:navActive(item)}" @click="navigate(item.path)"><i>·</i>{{ item.label }}</button></nav></aside>
      <section class="workspace-shell"><div class="topbar"><span>YLVEN / {{ current.group }} / {{ current.title }}</span><div><b class="environment">staging</b><span class="avatar">管</span></div></div>
        <main class="content"><div class="page-head"><div><h1>{{ current.title }}</h1><p>{{ current.group }} · {{ current.pageId }}</p></div><button v-if="actionLabel" class="primary action" @click="runAction">{{ actionLabel }}</button></div>
          <div v-if="current.group==='管理员与权限'" class="subnav"><button :class="{active:path==='/admin/037'}" @click="navigate('/admin/037')">角色</button><button :class="{active:path==='/admin/038'}" @click="navigate('/admin/038')">管理员</button></div>
          <p v-if="notice" class="notice">{{ notice }}</p><p v-if="error" class="error-banner">{{ error }}</p>
          <section v-if="!allowed" class="empty-state"><h2>权限不足</h2><p>当前管理员没有访问此资源的权限。</p></section>
          <section v-else-if="loading" class="loading-panel"><div class="skeleton"></div><div class="skeleton short"></div></section>
          <template v-else>
            <div v-if="['/admin/027','/admin/037','/admin/038'].includes(path)" class="filter-bar"><input placeholder="搜索名称或 ID"><button>状态：全部</button><button>类型：全部</button><button>最近 30 天</button></div>
            <div v-if="path==='/admin/027'" class="data-table"><div class="table-head"><span>用户/对象</span><span>ID</span><span>状态</span><span>创建时间</span><span>操作</span></div><div v-for="user in data?.users || []" :key="user.id" class="table-row"><span><b>{{ user.email }}</b></span><span>{{ user.id }}</span><span :class="['state-label',user.status]">{{ user.status === 'active' ? '正常' : '已禁用' }}</span><span>{{ user.created_at ? new Date(user.created_at).toLocaleString() : '—' }}</span><button class="link-button" @click="updateUser(user)" :disabled="saving">{{ user.status === 'active' ? '禁用' : '恢复' }}</button></div><p v-if="!data?.users?.length" class="empty-row">暂无用户</p></div>
            <div v-else-if="path==='/admin/037'" class="data-table"><form v-if="roleEditor" class="inline-editor" @submit.prevent="createRole"><input v-model="role.id" placeholder="角色 ID" required><input v-model="role.name" placeholder="角色名称" required><input v-model="role.permissions" placeholder="权限，逗号分隔" required><button class="primary" :disabled="saving">保存</button></form><div class="table-head"><span>名称</span><span>ID</span><span>状态</span><span>权限</span><span>操作</span></div><div v-for="item in data?.roles || []" :key="item.id" class="table-row"><span><b>{{ item.name }}</b></span><span>{{ item.id }}</span><span class="state-label active">正常</span><span><code>{{ item.permissions.join(', ') }}</code></span><button class="link-button" @click="stepUpOpen=true">配置</button></div><p v-if="!data?.roles?.length" class="empty-row">暂无角色</p></div>
            <div v-else-if="path==='/admin/038'" class="data-table"><div class="table-head"><span>管理员</span><span>ID</span><span>状态</span><span>角色</span><span>会话</span></div><div v-for="admin in data?.admins || []" :key="admin.id" class="table-row"><span><b>{{ admin.email }}</b></span><span>{{ admin.id }}</span><span class="state-label active">{{ admin.status }}</span><span>{{ admin.role_ids.join(', ') }}</span><span>{{ data?.sessions?.filter((session) => session.admin_user_id === admin.id).length || 0 }}</span></div><p v-if="!data?.admins?.length" class="empty-row">暂无管理员</p></div>
            <div v-else-if="path==='/admin/039'" class="approval-list"><article v-for="item in ['账号状态变更','角色权限调整','邮件模板发布']" :key="item"><div><span class="risk-tag">高风险</span><h2>{{ item }}需要二次确认</h2><p>确认结果将绑定当前管理员会话并写入审计记录</p></div><button class="outline-danger">拒绝</button><button class="primary" @click="stepUpOpen=true">确认</button></article></div>
            <div v-else-if="path==='/admin/040' && templateDraft" class="config-panel"><div class="config-title"><h2>配置与策略</h2><span>版本 {{ templateDraft.version }}</span></div><form @submit.prevent="saveTemplate"><label><span>模板键</span><input v-model="templateDraft.key" readonly></label><label><span>主题</span><input class="template-subject" v-model="templateDraft.subject" required></label><label><span>正文</span><textarea v-model="templateDraft.body" required></textarea></label><div class="config-footer"><small>发送记录 {{ data.deliveries.length }} 条</small><button class="primary" :disabled="saving">保存配置</button></div></form></div>
            <div v-else class="config-panel"><div class="config-title"><h2>{{ current.title }}</h2><span>{{ current.pageId }}</span></div><pre v-if="data" class="api-data">{{ JSON.stringify(data, null, 2) }}</pre><p v-else>{{ current.detail }}</p></div>
          </template>
        </main>
      </section>
      <div v-if="stepUpOpen" class="modal-backdrop" @click.self="stepUpOpen=false"><form class="step-up-dialog" @submit.prevent="confirmStepUp"><span class="risk-tag">高风险确认</span><h2>再次验证管理员身份</h2><p>此确认仅用于当前会话中的下一次高风险写操作。</p><label>管理员密码<input v-model="stepUpCredential" type="password" autocomplete="current-password" required autofocus></label><div><button type="button" class="secondary" @click="stepUpOpen=false">取消</button><button class="primary" :disabled="saving">确认</button></div></form></div>
    </div>`
}
