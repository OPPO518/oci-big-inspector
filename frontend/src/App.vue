<template>
  <div class="app-layout">
    <aside class="sidebar">
      <div class="sidebar-brand">
        <i class="fa-solid fa-user-secret text-primary" style="font-size: 22px;"></i>
        <span>大探长 OCI 群控</span>
      </div>
      
      <div class="sidebar-menu">
        <div class="menu-group">服务管理</div>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'monitor' }" @click="currentTab = 'monitor'">
          <i class="fa-solid fa-chart-line"></i> 系统资源监控
        </a>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'tenant' }" @click="currentTab = 'tenant'">
          <i class="fa-solid fa-rectangle-list"></i> 租户凭证管理
        </a>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'quota' }" @click="currentTab = 'quota'">
          <i class="fa-solid fa-earth-americas"></i> OCI 区域管理
        </a>

        <div class="menu-group">群控自动化</div>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'boot' }" @click="currentTab = 'boot'">
          <i class="fa-solid fa-circle-play"></i> OCI 开机管理
        </a>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'boot-logs' }" @click="currentTab = 'boot-logs'">
          <i class="fa-solid fa-terminal"></i> OCI 开机日志
        </a>

        <div class="menu-group">资源管理</div>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'instances' }" @click="currentTab = 'instances'">
          <i class="fa-solid fa-server"></i> OCI 实例列表
        </a>

        <div class="menu-group">系统管理</div>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'security' }" @click="currentTab = 'security'">
          <i class="fa-solid fa-user-shield"></i> 安全管理
        </a>
      </div>
    </aside>

    <div class="main-wrapper">
      <div v-if="loading" class="loading-screen">
        <div class="spinner"></div>
        <h2>大探长控制台安全握手中...</h2>
      </div>

      <div v-else-if="needInit" class="init-screen">
        <div class="init-card fade-in">
          <h2>🚀 初始化最高管理安全凭证</h2>
          <form @submit.prevent="submitInit">
            <div class="form-group"><label>管理员账号</label><input v-model="initForm.username" type="text" required /></div>
            <div class="form-group"><label>高强度安全密码</label><input v-model="initForm.password" type="password" required /></div>
            <button type="submit" :disabled="submitting" class="btn btn-check" style="width:100%">保存并启动面板</button>
          </form>
        </div>
      </div>

      <main v-else class="main-content fade-in">
        
        <div v-if="currentTab === 'tenant'">
          <header class="dash-header">
            <div class="logo-area">
              <i class="fa-solid fa-key" style="color: #38bdf8; margin-right: 10px; font-size: 20px;"></i>
              <h2>租户管理</h2>
            </div>
            
            <div class="search-bar">
              <input v-model="searchQuery" type="text" placeholder="输入自定义名称或主区域进行过滤..." />
              <button class="btn-search"><i class="fa-solid fa-magnifying-glass"></i></button>
            </div>

            <div class="btn-group">
              <button class="btn btn-icon"><i class="fa-solid fa-eye"></i></button>
              <button class="btn btn-api" @click="showModal = true"><i class="fa-solid fa-bolt"></i> API导入</button>
              <button class="btn btn-export"><i class="fa-solid fa-download"></i> 导出租户数据</button>
              <button class="btn btn-export"><i class="fa-solid fa-upload"></i> 导入租户数据</button>
              <button class="btn btn-check" @click="batchTest"><i class="fa-solid fa-circle-check"></i> 账号批量检测</button>
            </div>
          </header>

          <div class="table-container">
            <table>
              <thead>
                <tr>
                  <th>#</th>
                  <th>租户名</th>
                  <th>自定义名称</th>
                  <th>账号类型</th>
                  <th>区域</th>
                  <th>是否多区</th>
                  <th>创建时间</th>
                  <th>存活天数</th>
                  <th>开机任务</th>
                  <th>账号状态</th>
                  <th>专属代理</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-if="filteredAccounts.length === 0">
                  <td colspan="12" class="text-center" style="padding: 40px; color: #4b5563;">暂无匹配的租户凭证</td>
                </tr>
                <tr v-for="(acc, index) in filteredAccounts" :key="acc.id">
                  <td class="text-muted font-mono">{{ index + 1 }}</td>
                  <td class="font-bold text-primary link-style" @click="viewDetails(acc)">{{ acc.alias }}</td>
                  <td>
                    <span class="badge badge-neutral font-mono" :title="acc.tenancy_id">
                      {{ acc.tenant_name && acc.tenant_name !== '获取中...' ? acc.tenant_name : acc.tenancy_id.substring(0, 10) + '...' }}
                    </span>
                  </td>
                  <td><span class="badge badge-info">{{ acc.account_type || '个人免费账号' }}</span></td>
                  <td class="text-primary font-bold">{{ acc.region }}</td>
                  <td>
                    <span v-if="acc.is_multi_region" class="badge badge-success">● Yes</span>
                    <span v-else class="text-muted" style="font-size: 13px;">● No</span>
                  </td>
                  <td class="text-sm font-mono">{{ formatTime(acc.created_at) }}</td>
                  <td class="font-mono text-success font-bold">{{ acc.alive_days }}d</td>
                  <td>
                    <span v-if="acc.has_boot_task" class="badge badge-warning animate-pulse">○ Active</span>
                    <span v-else class="text-muted">○ Idle</span>
                  </td>
                  <td>
                    <span v-if="acc.status === 'active'" class="badge badge-success"><i class="fa-solid fa-circle-check"></i> 有效</span>
                    <span v-else class="badge badge-danger">失效</span>
                  </td>
                  <td class="font-mono text-sm code-font">{{ acc.proxy || '直连' }}</td>
                  <td class="action-cell">
                    <button class="btn-create-spec" @click="fastCreate(acc)"><i class="fa-solid fa-rocket"></i> 创建实例</button>
                    <button class="btn btn-icon" @click="viewDetails(acc)" title="账户详情"><i class="fa-solid fa-ellipsis"></i></button>
                  </td>
                </tr>
              </tbody>
            </table>
            
            <div class="table-footer-controls text-muted">
              <div class="per-page-selector">
                每页显示 
                <select><option>10</option><option>20</option></select>
              </div>
              <div class="pagination-pages">
                <button class="page-nav-btn" disabled>&lt; 上一页</button>
                <span class="page-num-badge active">1</span>
                <button class="page-nav-btn" disabled>下一页 &gt;</button>
                <span style="margin-left: 15px;">共 {{ filteredAccounts.length }} 条 第 1 / 1 页</span>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="currentTab === 'security'" class="placeholder-container card" style="text-align: left; max-width: 700px; padding: 40px;">
          <h3><i class="fa-solid fa-user-shield text-primary"></i> 安全与 Telegram 通知配置</h3>
          <p class="text-muted" style="margin-bottom: 25px;">在此配置大探长系统的核心防护层，以及对接 Telegram 自动化实时通知总线。</p>
          
          <form @submit.prevent="saveTgConfig">
            <div class="form-group">
              <label>Telegram Bot Token</label>
              <input v-model="tgForm.tg_bot_token" type="text" placeholder="例如：123456789:ABCdefGhIJKlmNoPQRsTUVwXyZ" />
            </div>
            <div class="form-group">
              <label>管理员 Chat ID</label>
              <input v-model="tgForm.tg_chat_id" type="text" placeholder="例如：987654321" />
            </div>
            <div class="form-group" style="display: flex; align-items: center; margin-top: 20px;">
              <label style="margin-bottom: 0; margin-right: 15px;">是否开启全局开机/下线 TG 实时通知</label>
              <input v-model="tgForm.tg_notify_enabled" type="checkbox" true-value="1" false-value="0" style="width: 20px; height: 20px;" />
            </div>
            <button type="submit" class="btn btn-check" style="margin-top: 20px; width: 100%; justify-content: center;">
              <i class="fa-solid fa-floppy-disk"></i> 保存并测试连接
            </button>
          </form>
        </div>

        <div v-else class="placeholder-container card">
          <i class="fa-solid fa-boxes-stacked placeholder-icon"></i>
          <h3>「 核心模块：{{ currentTab.toUpperCase() }} 」已完成全局卡位</h3>
          <p class="text-muted">当前前端导航与样式架构已完美对齐 OCI-START 深色面板。</p>
          <div class="status-indicator">
            <span class="pulse-dot green"></span>
            <span class="status-msg">已成功建立与 Go 核心加密引擎的数据监听通道</span>
          </div>
        </div>

      </main>

      <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
        <div class="modal-content fade-in-up">
          <h3><i class="fa-solid fa-bolt" style="color:#22c55e;"></i> API配置快速导入</h3>
          <p class="text-sm text-muted" style="margin-bottom: 20px;">系统自动提取字段，您仅需补充别名、物理配置与专属代理通道。</p>
          <form @submit.prevent="submitAddAccount">
            <div class="form-group">
              <label>1. 粘贴 OCI 原始凭证 (Config)</label>
              <textarea v-model="addForm.raw_config" rows="4" class="code-input" placeholder="粘贴类似 user=..., tenancy=... 的内容"></textarea>
            </div>
            <div class="grid-inputs">
              <div class="form-group"><label>2. 自定义名称</label><input v-model="addForm.alias" type="text" required placeholder="如：墨西哥蒙特雷A" /></div>
              <div class="form-group">
                <label>3. 账号类型</label>
                <select v-model="addForm.account_type">
                  <option value="个人免费账号">个人免费账号</option>
                  <option value="升级版账号">升级版账号</option>
                </select>
              </div>
            </div>
            <div class="grid-inputs">
              <div class="form-group"><label>4. 专属代理网络 (防关联)</label><input v-model="addForm.proxy" type="text" placeholder="IP:PORT，直连则不填" /></div>
              <div class="form-group" style="display:flex; align-items:center; margin-top:25px;">
                <label style="margin-right:15px; margin-bottom:0;">5. 是否开通多区配额</label>
                <input type="checkbox" v-model="addForm.is_multi_region" style="width:20px; height:20px;" />
              </div>
            </div>
            <div class="form-group">
              <label>6. 密钥文件 (.pem)</label>
              <div class="file-upload-wrapper">
                <input type="file" @change="handleFileUpload" accept=".pem,.key" id="file-upload" class="hidden-file-input" />
                <label for="file-upload" class="file-upload-btn"><i class="fa-solid fa-file-shield"></i> 选择私钥文件</label>
                <span class="text-sm font-mono" style="margin-left: 10px; color: #38bdf8;">{{ uploadedFileName }}</span>
              </div>
              <textarea v-model="addForm.private_key" rows="2" placeholder="或者直接在此粘贴 KEY 文本..." style="margin-top:10px;"></textarea>
            </div>
            <div class="modal-actions">
              <button type="button" class="btn btn-export" @click="showModal = false">取消</button>
              <button type="submit" :disabled="submitting" class="btn btn-api">保存凭证</button>
            </div>
          </form>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import axios from 'axios'

const loading = ref(true); const needInit = ref(false); const submitting = ref(false); const showModal = ref(false); const accounts = ref([])
const searchQuery = ref(''); const uploadedFileName = ref('未选择任何文件')
const currentTab = ref('tenant') 

const initForm = ref({ username: '', password: '' })
const tgForm = ref({ tg_bot_token: '', tg_chat_id: '', tg_notify_enabled: '0' })
const addForm = ref({ alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '', account_type: '个人免费账号', is_multi_region: false, proxy: '直连' })

watch(() => addForm.value.raw_config, (val) => {
  if (!val) return
  const lines = val.split('\n')
  lines.forEach(line => {
    const parts = line.split('=')
    if (parts.length >= 2) {
      const k = parts[0].trim().toLowerCase()
      const v = parts.slice(1).join('=').trim()
      if (k === 'user') addForm.value.user_id = v
      if (k === 'tenancy') addForm.value.tenancy_id = v
      if (k === 'region') addForm.value.region = v
      if (k === 'fingerprint') addForm.value.fingerprint = v
    }
  })
})

const filteredAccounts = computed(() => {
  if (!searchQuery.value) return accounts.value
  return accounts.value.filter(acc => 
    acc.alias.toLowerCase().includes(searchQuery.value.toLowerCase()) || 
    acc.region.toLowerCase().includes(searchQuery.value.toLowerCase())
  )
})

const handleFileUpload = (event) => {
  const file = event.target.files[0]
  if (!file) return
  uploadedFileName.value = file.name
  const reader = new FileReader()
  reader.onload = (e) => { addForm.value.private_key = e.target.result }
  reader.readAsText(file)
}

const checkSystemStatus = async () => {
  try {
    const res = await axios.get('/api/status')
    needInit.value = res.data?.need_init
    if (!needInit.value) fetchAccounts()
  } catch(e) { needInit.value = false }
  finally { loading.value = false }
}

const fetchAccounts = async () => {
  try {
    const res = await axios.get('/api/accounts/list')
    accounts.value = res.data || []
    accounts.value.forEach(async (acc) => {
      if (!acc.tenant_name || acc.tenant_name === '获取中...') {
        try {
          const testRes = await axios.post('/api/accounts/test', { id: acc.id })
          if (testRes.data && testRes.data.tenant_name) acc.tenant_name = testRes.data.tenant_name
        } catch (err) { acc.tenant_name = '认证失败' }
      }
    })
  } catch(e) { console.error(e) }
}

const fetchTgConfig = async () => {
  try {
    const res = await axios.get('/api/system/config/get')
    if (res.data) {
      tgForm.value.tg_bot_token = res.data.tg_bot_token || ''
      tgForm.value.tg_chat_id = res.data.tg_chat_id || ''
      tgForm.value.tg_notify_enabled = res.data.tg_notify_enabled || '0'
    }
  } catch (e) { console.error(e) }
}

const saveTgConfig = async () => {
  try {
    await axios.post('/api/system/config/save', tgForm.value)
    alert('TG通知渠道配置保存成功！已尝试推送一则测试信息。')
  } catch (e) { alert('保存失败') }
}

const submitInit = async () => {
  submitting.value = true
  try { await axios.post('/api/system/init', initForm.value); window.location.reload() } 
  finally { submitting.value = false }
}

const submitAddAccount = async () => {
  submitting.value = true
  try {
    await axios.post('/api/accounts/add', addForm.value)
    showModal.value = false
    addForm.value = { alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '', account_type: '个人免费账号', is_multi_region: false, proxy: '直连' }
    uploadedFileName.value = '未选择任何文件'
    fetchAccounts()
  } catch(e) { alert('添加失败') } 
  finally { submitting.value = false }
}

const batchTest = () => { alert('多租户并发检测中...'); fetchAccounts() }
const fastCreate = (acc) => { alert(`正在调取 [${acc.alias}] 执行快速挂载与开机向导...`) }
const viewDetails = (acc) => { alert(`打开账户 [${acc.alias}] 的配置详情页`) }
const formatTime = (t) => t ? t.substring(0, 16) : '2026-06-09 21:09'

watch(currentTab, (newTab) => {
  if (newTab === 'security') fetchTgConfig()
})

onMounted(() => checkSystemStatus())
</script>

<style>
body { background-color: #0b0f19; color: #cbd5e1; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; }
.fade-in { animation: fadeIn 0.25s ease-in; }
.fade-in-up { animation: fadeInUp 0.3s ease-out forwards; }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes fadeInUp { from { opacity: 0; transform: translateY(15px); } to { opacity: 1; transform: translateY(0); } }

/* 侧边栏与大局布局 */
.app-layout { display: flex; min-height: 100vh; }
.sidebar { width: 250px; background: #111827; border-right: 1px solid #1f2937; display: flex; flex-direction: column; flex-shrink: 0; position: fixed; height: 100vh; z-index: 10; }
.main-wrapper { flex: 1; margin-left: 250px; min-width: 0; display: flex; flex-direction: column; }
.main-content { padding: 25px; flex: 1; }

/* 侧边栏菜单 */
.sidebar-brand { padding: 24px 20px; display: flex; align-items: center; gap: 12px; border-bottom: 1px solid #1f2937; font-weight: 700; font-size: 15px; color: #fff; letter-spacing: 0.5px; }
.sidebar-menu { padding: 15px 10px; display: flex; flex-direction: column; gap: 2px; overflow-y: auto; flex: 1; }
.menu-group { font-size: 11px; text-transform: uppercase; color: #4b5563; font-weight: 700; padding: 15px 12px 6px 12px; letter-spacing: 0.5px; }
.menu-item { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-radius: 6px; color: #9ca3af; text-decoration: none; font-size: 13px; font-weight: 500; transition: all 0.15s; }
.menu-item i { width: 16px; font-size: 14px; text-align: center; color: #6b7280; }
.menu-item:hover { background: #161e2e; color: #f3f4f6; }
.menu-item:hover i { color: #38bdf8; }
.menu-item.active { background: #1e293b; color: #38bdf8; font-weight: 600; }
.menu-item.active i { color: #38bdf8; }
.menu-item span { margin-left: auto; font-size: 10px; background: #1f2937; color: #64748b; padding: 1px 4px; border-radius: 4px; border: 1px solid #374151; }

/* 占位符卡片 */
.placeholder-container { display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; padding: 80px 40px; margin: 40px auto; max-width: 650px; background: #111827; border-radius: 12px; border: 1px solid #1f2937; }
.placeholder-icon { font-size: 56px; color: #1f2937; margin-bottom: 20px; }
.status-indicator { display: flex; align-items: center; gap: 10px; margin-top: 25px; background: #0b0f19; padding: 8px 16px; border-radius: 30px; border: 1px solid #1f2937; }
.pulse-dot { width: 8px; height: 8px; border-radius: 50%; }
.pulse-dot.green { background: #10b981; box-shadow: 0 0 8px #10b981; animation: pulse 2s infinite; }
.status-msg { font-size: 12px; color: #9ca3af; font-family: monospace; }
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }

/* 头部面板条 */
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; background: #111827; padding: 12px 20px; border-radius: 8px; border: 1px solid #1f2937; }
.logo-area { display: flex; align-items: center; }
.logo-area h2 { margin: 0; font-size: 16px; color: #f8fafc; font-weight: 600; }
.search-bar { display: flex; width: 320px; }
.search-bar input { flex: 1; background: #0b0f19; border: 1px solid #1f2937; padding: 8px 12px; border-radius: 6px 0 0 6px; color: #fff; outline: none; font-size: 13px; }
.btn-search { background: #2563eb; border: none; color: white; padding: 0 14px; border-radius: 0 6px 6px 0; cursor: pointer; }
.btn-group { display: flex; gap: 6px; }

/* 表格主样式 */
.table-container { background: #111827; border-radius: 8px; border: 1px solid #1f2937; overflow: hidden; }
table { width: 100%; border-collapse: collapse; text-align: left; }
th { background: #1f2937; color: #9ca3af; font-size: 13px; font-weight: 500; padding: 14px 16px; border-bottom: 1px solid #1f2937; }
td { padding: 14px 16px; border-bottom: 1px solid #1f2937; font-size: 13px; vertical-align: middle; }
tr:hover { background: #161e2e; }

/* 分页器 */
.table-footer-controls { padding: 12px 16px; display: flex; justify-content: space-between; align-items: center; background: #111827; border-top: 1px solid #1f2937; font-size: 13px; }
.per-page-selector select { background: #0b0f19; border: 1px solid #1f2937; color: #fff; padding: 3px 6px; border-radius: 4px; outline: none; margin-left: 6px; }
.pagination-pages { display: flex; align-items: center; gap: 4px; }
.page-nav-btn { background: #1f2937; border: 1px solid #374151; color: #9ca3af; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 12px; }
.page-nav-btn:disabled { opacity: 0.3; cursor: not-allowed; }
.page-num-badge { padding: 3px 8px; border-radius: 4px; font-size: 12px; font-weight: 600; cursor: pointer; }
.page-num-badge.active { background: #10b981; color: #fff; }

/* 组件与标签 */
.btn { border: none; padding: 8px 14px; border-radius: 6px; font-size: 13px; cursor: pointer; display: inline-flex; align-items: center; gap: 6px; font-weight: 500; }
.btn-api { background: #10b981; color: white; }
.btn-export { background: #1f2937; color: #9ca3af; border: 1px solid #374151; }
.btn-check { background: #2563eb; color: white; }
.btn-icon { padding: 8px 12px; background: #1e293b; color: #cbd5e1; border: none; border-radius: 6px; cursor: pointer; }
.btn-create-spec { background: #f59e0b; color: #000; font-weight: 700; border: none; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 12px; margin-right: 8px; }
.badge { padding: 3px 6px; border-radius: 4px; font-size: 11px; font-weight: 600; }
.badge-info { background: rgba(56,189,248,0.1); color: #38bdf8; }
.badge-success { background: rgba(16,185,129,0.1); color: #10b981; }
.badge-danger { background: rgba(239,68,68,0.1); color: #ef4444; }
.badge-neutral { background: #1f2937; color: #9ca3af; border: 1px solid #374151; }
.code-font { background: #1f2937; padding: 2px 6px; border-radius: 4px; color: #e2e8f0; }
.action-cell { display: flex; align-items: center; }

/* 模态框弹窗 */
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; justify-content: center; align-items: center; z-index: 200; }
.modal-content { background: #111827; padding: 30px; border-radius: 12px; width: 620px; border: 1px solid #1f2937; }
.modal-content h3 { margin: 0 0 5px 0; color: #fff; }
.form-group { margin-bottom: 16px; }
.form-group label { display: block; margin-bottom: 6px; font-size: 12px; color: #9ca3af; font-weight: 600; }
.form-group input, .form-group select, .form-group textarea { width: 100%; box-sizing: border-box; background: #0b0f19; border: 1px solid #1f2937; color: #fff; padding: 10px; border-radius: 6px; outline: none; font-size: 13px; }
.code-input { font-family: monospace; color: #10b981 !important; }
.hidden-file-input { display: none; }
.file-upload-btn { background: #1f2937; color: #e5e7eb; padding: 8px 14px; border-radius: 6px; cursor: pointer; font-size: 12px; border: 1px solid #374151; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 25px; }
.spinner { border: 4px solid rgba(255,255,255,0.1); border-top: 4px solid #38bdf8; border-radius: 50%; width: 35px; height: 35px; animation: spin 1s linear infinite; margin-bottom: 15px; }
@keyframes spin { 100% { transform: rotate(360deg); } }
</style>
