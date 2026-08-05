import { computed, onMounted, ref, watch } from 'vue'
import { canAccess } from './router.js'

const pages = [
  { path: '/admin/018', title: '模型与供应商', status: 'SUCCESS', detail: '能力探测状态可回读' },
  { path: '/admin/019', title: '安全与审计', status: 'PARTIAL', detail: '秘密仅显示配置状态' },
  { path: '/admin/020', title: '管理员与权限', status: 'READY', detail: 'RBAC 策略已加载' },
  { path: '/admin/021', title: '开发者平台', status: 'EMPTY', detail: '尚未配置额外入口' },
  { path: '/admin/026', title: '验证码策略', status: 'LIVE', detail: '时效、频率与冷却配置', endpoint: '/admin/v1/settings/otp-policy' },
  { path: '/admin/027', title: '用户列表', status: 'LIVE', detail: '用户身份和账号状态', endpoint: '/admin/v1/users' },
  { path: '/admin/034', title: '邮件服务', status: 'LIVE', detail: '邮件服务商、发件人与密钥引用', endpoint: '/admin/v1/settings/email' },
  { path: '/admin/035', title: 'Turnstile', status: 'LIVE', detail: '站点与 Secret 引用', endpoint: '/admin/v1/settings/turnstile' },
  { path: '/admin/036', title: '用户详情', status: 'READY', detail: '身份、设备和会话详情' }
]

export default {
  setup() {
    const path = ref(window.location.hash.slice(1) || '/admin/018')
    const permissions = ref(['admin:read'])
    const loading = ref(false)
    const data = ref(null)
    const error = ref('')
    const current = computed(() => pages.find((page) => page.path === path.value) || pages[0])
    const allowed = computed(() => canAccess(path.value, permissions.value))
    function navigate(next) { path.value = next; window.location.hash = next }
    function togglePermission() { permissions.value = permissions.value.length ? [] : ['admin:read'] }
    async function load() {
      data.value = null; error.value = ''
      if (!current.value.endpoint) return
      loading.value = true
      try {
        const response = await fetch(current.value.endpoint, { headers: { 'X-Admin-Role': 'superadmin' } })
        if (!response.ok) throw new Error(`HTTP ${response.status}`)
        data.value = await response.json()
      } catch (reason) { error.value = reason.message }
      finally { loading.value = false }
    }
    onMounted(load); watch(path, load)
    return { pages, path, current, allowed, loading, data, error, navigate, togglePermission, load }
  },
  template: `
    <div class="shell">
      <aside class="sidebar"><div class="brand">YLVEN <span>ADMIN</span></div><nav>
        <button v-for="page in pages" :key="page.path" :class="{active:path===page.path}" @click="navigate(page.path)">{{ page.title }}</button>
      </nav><button class="audit" @click="togglePermission">权限校验</button></aside>
      <main class="content"><header><div><p class="eyebrow">OPERATIONS</p><h1>{{ current.title }}</h1></div><span class="badge">YL-DS-1.2.0</span></header>
        <section v-if="!allowed" class="state error"><h2>权限不足</h2><p>当前账号没有访问此管理资源的权限。</p></section>
        <section v-else-if="loading" class="state loading"><div class="skeleton"></div><div class="skeleton short"></div></section>
        <section v-else-if="error" class="state error"><h2>加载失败</h2><p>{{ error }}</p><button @click="load">重试</button></section>
        <section v-else class="panel"><div class="panel-head"><div><h2>{{ current.title }}</h2><p>{{ current.detail }}</p></div><span class="status">{{ current.status }}</span></div><div class="metrics"><div><b>服务状态</b><strong>已连接</strong></div><div><b>数据</b><strong>{{ data ? '已同步' : '等待选择' }}</strong></div><div><b>审计</b><strong>已记录</strong></div></div><pre v-if="data" class="api-data">{{ JSON.stringify(data, null, 2) }}</pre></section>
      </main>
    </div>`
}
