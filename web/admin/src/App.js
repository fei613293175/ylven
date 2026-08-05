import { computed, ref } from 'vue'
import { canAccess } from './router.js'

const pages = [
  { path: '/admin/018', title: '模型与供应商', status: 'SUCCESS', detail: '能力探测状态可回读' },
  { path: '/admin/019', title: '安全与审计', status: 'PARTIAL', detail: '秘密仅显示配置状态' },
  { path: '/admin/020', title: '管理员与权限', status: 'READY', detail: 'RBAC 策略已加载' },
  { path: '/admin/021', title: '开发者平台', status: 'EMPTY', detail: '尚未配置额外入口' }
]

export default {
  setup() {
    const path = ref(window.location.hash.slice(1) || '/admin/018')
    const permissions = ref(['admin:read'])
    const loading = ref(false)
    const current = computed(() => pages.find((page) => page.path === path.value) || pages[0])
    const allowed = computed(() => canAccess(path.value, permissions.value))
    function navigate(next) { path.value = next; window.location.hash = next }
    function togglePermission() { permissions.value = permissions.value.length ? [] : ['admin:read'] }
    return { pages, path, current, allowed, loading, navigate, togglePermission }
  },
  template: `
    <div class="shell">
      <aside class="sidebar"><div class="brand">YLVEN <span>ADMIN</span></div><nav>
        <button v-for="page in pages" :key="page.path" :class="{active:path===page.path}" @click="navigate(page.path)">{{ page.title }}</button>
      </nav><button class="audit" @click="togglePermission">权限校验</button></aside>
      <main class="content"><header><div><p class="eyebrow">OPERATIONS</p><h1>{{ current.title }}</h1></div><span class="badge">YL-DS-1.2.0</span></header>
        <section v-if="!allowed" class="state error"><h2>权限不足</h2><p>当前账号没有访问此管理资源的权限。</p></section>
        <section v-else-if="loading" class="state loading"><div class="skeleton"></div><div class="skeleton short"></div></section>
        <section v-else class="panel"><div class="panel-head"><div><h2>{{ current.title }}</h2><p>{{ current.detail }}</p></div><span class="status">{{ current.status }}</span></div><div class="metrics"><div><b>服务状态</b><strong>已连接</strong></div><div><b>最近检查</b><strong>刚刚</strong></div><div><b>审计</b><strong>已记录</strong></div></div></section>
      </main>
    </div>`
}
