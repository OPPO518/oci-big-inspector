<template>
  <div v-if="!loading && needInit" class="init-pure-wrapper">
    <div class="init-card fade-in">
      <h2>🚀 初始化最高管理安全凭证</h2>
      <p class="text-sm text-muted">系统尚未初始化，请设定本地管理员凭证以对齐群控安全策略。</p>
      <form @submit.prevent="submitInit">
        <div class="form-group"><label>管理员账号</label><input v-model="initForm.username" type="text" required /></div>
        <div class="form-group"><label>管理员密码</label><input v-model="initForm.password" type="password" required /></div>
        <button type="submit" class="btn btn-check" style="width:100%; justify-content: center;">保存并启动系统</button>
      </form>
    </div>
  </div>

  <div v-else-if="!loading && !needInit && !isLoggedIn" class="init-pure-wrapper">
    <div class="init-card fade-in">
      <h2><i class="fa-solid fa-lock text-primary"></i> 大探长安全身份验证</h2>
      <p class="text-sm text-muted">请验证身份后进入控制台。</p>
      <form @submit.prevent="submitLogin">
        <div class="form-group"><label>管理员账号</label><input v-model="loginForm.username" type="text" required /></div>
        <div class="form-group"><label>管理员密码</label><input v-model="loginForm.password" type="password" required /></div>
        <button type="submit" class="btn btn-check" style="width:100%; justify-content: center;">安全登入</button>
      </form>
    </div>
  </div>

  <div v-else-if="loading" class="loading-screen-full">
    <div class="spinner"></div>
    <h2>大探长控制台安全握手中...</h2>
  </div>

  <div v-else-if="isLoggedIn" class="app-layout fade-in">
    <aside class="sidebar">
      <div class="sidebar-brand"><i class="fa-solid fa-user-secret text-primary" style="font-size: 22px;"></i><span>大探长 OCI 群控</span></div>
      <div class="sidebar-menu">
        <div class="menu-group">服务 management</div>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'monitor' }" @click="currentTab = 'monitor'"><i class="fa-solid fa-chart-line"></i> 系统资源监控</a>
        <a href="#" class="menu-item" :class="{ active: currentTab === 'tenant' }" @click="currentTab = 'tenant'"><i class="fa-solid fa-rectangle-list"></i> 租户凭证管理</a>
        <div class="menu-group">系统管理</div>
        <a href="#" class="menu-item" style="color: #ef4444; margin-top: auto;" @click.prevent="logout"><i class="fa-solid fa-right-from-bracket"></i> 退出面板</a>
      </div>
    </aside>
    <div class="main-wrapper">
      <main class="main-content">
        <div v-if="currentTab === 'monitor'">
          <div class="monitor-header">
            <div class="m-title"><i class="fa-solid fa-laptop-code" style="color: #38bdf8;"></i><h2>系统资源监控</h2></div>
            <div class="realtime-clock font-mono">● {{ currentTimeStr }}</div>
          </div>
          <div class="monitor-grid-top">
            <div class="m-card-mini"><div class="mini-icon blue"><i class="fa-solid fa-list-check"></i></div><div class="mini-info"><span class="title">总 API 数</span><span class="value font-mono">{{ monitorData.total_apis }}</span></div></div>
            <div class="m-card-mini"><div class="mini-icon green"><i class="fa-solid fa-microchip"></i></div><div class="mini-info"><span class="title">总 BOOT 实例数</span><span class="value font-mono">{{ monitorData.total_boots }}</span></div></div>
            <div class="m-card-mini"><div class="mini-icon orange"><i class="fa-solid fa-arrows-spin"></i></div><div class="mini-info"><span class="title">总抢机次数</span><span class="value font-mono">{{ monitorData.total_runs }}</span></div></div>
          </div>
          <div class="monitor-grid-main">
            <div class="m-box">
              <div class="box-head"><i class="fa-solid fa-gauge-high text-primary"></i> <span>系统信息</span></div>
              <div class="box-body split-layout">
                <div class="details-list font-mono">
                  <div class="item"><span class="lbl">操作系统:</span><span class="val">{{ monitorData.os_info || 'Linux' }}</span></div>
                  <div class="item"><span class="lbl">运行时间:</span><span class="val text-primary">{{ monitorData.uptime }}</span></div>
                </div>
              </div>
            </div>
            <div class="m-box">
              <div class="box-head"><i class="fa-solid fa-memory text-success"></i> <span>内存使用</span></div>
              <div class="box-body split-layout">
                <div class="details-list font-mono">
                  <div class="item"><span class="lbl">已用内存:</span><span class="val">{{ monitorData.mem_used ? monitorData.mem_used.toFixed(2) : 0 }} GB</span></div>
                  <div class="item"><span class="lbl">内存使用率:</span><span class="val">{{ monitorData.mem_usage_pct }}%</span></div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="currentTab === 'tenant'">
          <header class="dash-header">
            <div class="logo-area"><i class="fa-solid fa-key" style="color: #38bdf8; margin-right: 10px; font-size: 20px;"></i><h2>租户凭证管理</h2></div>
            <div class="btn-group">
              <button class="btn btn-api" @click="showModal = true"><i class="fa-solid fa-plus"></i> 添加 API 凭证</button>
              <button class="btn btn-check" @click="batchTest"><i class="fa-solid fa-circle-check"></i> 账号批量检测</button>
            </div>
          </header>
          <div class="table-container">
            <table>
              <thead><tr><th>#</th><th>自定义名称</th><th>租户名</th><th>账号类型</th><th>区域</th><th>是否多区</th><th>创建时间</th><th>存活天数</th><th>开机任务</th><th>账号状态</th><th>专属代理</th><th>操作</th></tr></thead>
              <tbody>
                <tr v-for="(acc, index) in accounts" :key="acc.id">
                  <td class="font-mono">{{ index + 1 }}</td>
                  <td class="font-bold text-primary">{{ acc.alias }}</td>
                  <td><span class="badge badge-neutral font-mono">{{ acc.tenant_name }}</span></td>
                  <td><span class="badge badge-info">{{ acc.account_type || '个人免费账号' }}</span></td>
                  <td class="text-primary">{{ acc.region }}</td>
                  <td><span v-if="acc.is_multi_region" class="badge badge-success">● Yes</span><span v-else class="text-muted">● No</span></td>
                  <td class="font-mono">{{ formatTime(acc.created_at) }}</td>
                  <td class="text-success font-bold font-mono">{{ acc.alive_days || 1 }}d</td>
                  <td><span class="text-muted">○ Idle</span></td>
                  <td><span class="badge badge-success">有效</span></td>
                  <td class="font-mono code-font">{{ acc.proxy }}</td>
                  <td class="action-cell">
                    <button class="btn-create-spec">创建</button>
                    <button class="btn-delete-spec" @click="deleteAccount(acc)"><i class="fa-solid fa-trash-can"></i> 删除</button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </main>

      <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
        <div class="modal-content fade-in-up">
          <h3><i class="fa-solid fa-bolt" style="color:#22c55e;"></i> API 凭证自动化纳管</h3>
          <form @submit.prevent="submitAddAccount">
            <div class="form-group"><label>1. Config</label><textarea v-model="addForm.raw_config" rows="3" class="code-input"></textarea></div>
            <div class="form-group"><label>2. 别名</label><input v-model="addForm.alias" type="text" required /></div>
            <div class="form-group"><label>3. 代理</label><input v-model="addForm.proxy" type="text" /></div>
            <div class="form-group"><label>4. 私钥</label><textarea v-model="addForm.private_key" rows="2" class="code-input"></textarea></div>
            <div class="modal-actions">
              <button type="button" class="btn btn-export" @click="showModal = false">取消</button>
              <button type="submit" :disabled="submitting" class="btn btn-api">立即存盘并触发自动体征探测</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, watch } from 'vue'
import axios from 'axios'

const loading = ref(true); const needInit = ref(false); const isLoggedIn = ref(false); 
const submitting = ref(false); const showModal = ref(false); const accounts = ref([])
const currentTab = ref('tenant'); const currentTimeStr = ref('')
const monitorData = ref({ total_apis: 0, total_boots: 0, total_runs: 0, success_runs: 0, fail_runs: 0, cpu_usage: 0, mem_total: 0, mem_used: 0, mem_usage_pct: 0, disk_total: 0, disk_used: 0, disk_usage_pct: 0, cpu_model: '', os_info: '', arch_info: '', uptime: '', processes: 0, threads: 0 })

let clockTimer = null; let monitorTimer = null
const initForm = ref({ username: '', password: '' }); const loginForm = ref({ username: '', password: '' })
const addForm = ref({ alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '', proxy: '直连' })

axios.interceptors.response.use(
  response => response,
  error => {
    if (error.response && error.response.status === 401) {
      isLoggedIn.value = false; localStorage.removeItem('big_inspector_auth'); delete axios.defaults.headers.common['Authorization']
    }
    return Promise.reject(error)
  }
)

const startTimers = () => {
  clockTimer = setInterval(() => { const d = new Date(); currentTimeStr.value = d.getFullYear() + '-' + String(d.getMonth()+1).padStart(2, '0') + '-' + String(d.getDate()).padStart(2, '0') + ' ' + String(d.getHours()).padStart(2, '0') + ':' + String(d.getMinutes()).padStart(2, '0') + ':' + String(d.getSeconds()).padStart(2, '0') }, 1000)
  fetchMonitorData(); monitorTimer = setInterval(fetchMonitorData, 5000)
}

const fetchMonitorData = async () => { if (currentTab.value === 'monitor' && !needInit.value && isLoggedIn.value) try { const res = await axios.get('/api/system/monitor'); if (res.data) monitorData.value = res.data } catch (e) {} }

const runSingleAccountTest = async (acc) => {
  const originalName = acc.tenant_name; acc.tenant_name = '探测同步中...'
  try {
    const testRes = await axios.post('/api/accounts/test', { id: acc.id })
    if (testRes.data && testRes.data.status === 'success') {
      acc.tenant_name = testRes.data.tenant_name; acc.created_at = testRes.data.created_at; acc.account_type = testRes.data.account_type; acc.is_multi_region = testRes.data.is_multi_region
      if (acc.created_at) { const t = new Date(acc.created_at.replace(' ', 'T')); const diff = Math.floor((new Date() - t) / (1000 * 60 * 60 * 24)); acc.alive_days = diff <= 0 ? 1 : diff }
    } else { acc.tenant_name = originalName }
  } catch (err) { acc.tenant_name = '认证失败' }
}

const fetchAccounts = async () => {
  if (needInit.value || !isLoggedIn.value) return
  try {
    const res = await axios.get('/api/accounts/list')
    accounts.value = res.data || []
    accounts.value.forEach(async (acc) => {
      if (!acc.tenant_name || acc.tenant_name === '获取中...') await runSingleAccountTest(acc)
      else if (acc.created_at && acc.created_at !== '获取中') { const t = new Date(acc.created_at.replace(' ', 'T')); const diff = Math.floor((new Date() - t) / (1000 * 60 * 60 * 24)); acc.alive_days = diff <= 0 ? 1 : diff }
    })
  } catch(e) {}
}

const batchTest = async () => { if (accounts.value.length === 0) return; alert('🚀 强制同步官方体征...'); for (const acc of accounts.value) { await runSingleAccountTest(acc) } }
const deleteAccount = async (acc) => { if (!confirm(`确定要彻底删除 [${acc.alias}] 吗？`)) return; try { const res = await axios.post('/api/accounts/delete', { id: acc.id }); if (res.data && res.data.status === 'success') fetchAccounts() } catch (e) { alert('删除失败') } }

const checkSystemStatus = async () => {
  try {
    const res = await axios.get('/api/status')
    needInit.value = !!res.data?.need_init
    if (!needInit.value) { 
      const savedToken = localStorage.getItem('big_inspector_auth')
      if (savedToken) { axios.defaults.headers.common['Authorization'] = 'Basic ' + savedToken; try { await axios.get('/api/login'); isLoggedIn.value = true; startTimers(); fetchAccounts() } catch(e) { isLoggedIn.value = false } }
    }
  } catch(e) { needInit.value = false } finally { loading.value = false }
}

const submitInit = async () => { submitting.value = true; try { const res = await axios.post('/api/system/init', initForm.value); if (res.data && res.data.status === 'success') window.location.reload() } catch(e) { alert('初始化失败') } finally { submitting.value = false } }
const submitLogin = async () => { submitting.value = true; const token = btoa(loginForm.value.username + ':' + loginForm.value.password); axios.defaults.headers.common['Authorization'] = 'Basic ' + token; try { await axios.get('/api/login'); localStorage.setItem('big_inspector_auth', token); isLoggedIn.value = true; startTimers(); fetchAccounts() } catch(e) { alert('账号或密码错误！'); delete axios.defaults.headers.common['Authorization'] } finally { submitting.value = false } }
const submitAddAccount = async () => { submitting.value = true; try { await axios.post('/api/accounts/add', addForm.value); showModal.value = false; fetchAccounts() } catch(e) { alert('凭证保存失败') } finally { submitting.value = false } }
const logout = () => { isLoggedIn.value = false; localStorage.removeItem('big_inspector_auth'); delete axios.defaults.headers.common['Authorization']; window.location.reload() }

watch(currentTab, (newTab) => { if (needInit.value || !isLoggedIn.value) return; if (newTab === 'monitor') fetchMonitorData(); if (newTab === 'tenant') fetchAccounts() })
watch(() => addForm.value.raw_config, (val) => { if (!val) return; const lines = val.split('\n'); lines.forEach(line => { const parts = line.split('='); if (parts.length >= 2) { const k = parts[0].trim().toLowerCase(); const v = parts.slice(1).join('=').trim(); if (k === 'user') addForm.value.user_id = v; if (k === 'tenancy') addForm.value.tenancy_id = v; if (k === 'region') addForm.value.region = v; if (k === 'fingerprint') addForm.value.fingerprint = v } }) })

const formatTime = (t) => t && t !== '获取中' ? t.substring(0, 10) : '获取中'
onMounted(() => checkSystemStatus())
onBeforeUnmount(() => { clearInterval(clockTimer); clearInterval(monitorTimer) })
</script>

<style>
body { background-color: #0b0f19; color: #cbd5e1; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; }
.fade-in { animation: fadeIn 0.25s ease-in; } .fade-in-up { animation: fadeInUp 0.3s ease-out forwards; }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } } @keyframes fadeInUp { from { opacity: 0; transform: translateY(15px); } to { opacity: 1; transform: translateY(0); } }
.init-pure-wrapper { min-height: 100vh; background-color: #0b0f19; display: flex; justify-content: center; align-items: center; padding: 20px; width: 100vw; box-sizing: border-box; }
.init-card { background: #111827; padding: 35px; border-radius: 12px; width: 100%; max-width: 480px; border: 1px solid #1f2937; box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5); }
.init-card h2 { margin-top: 0; margin-bottom: 10px; font-size: 20px; color: #fff; text-align: center; }
.init-card p { text-align: center; margin-bottom: 25px; line-height: 1.5; }
.app-layout { display: flex; min-height: 100vh; }
.sidebar { width: 250px; background: #111827; border-right: 1px solid #1f2937; display: flex; flex-direction: column; flex-shrink: 0; position: fixed; height: 100vh; z-index: 10; }
.main-wrapper { flex: 1; margin-left: 250px; min-width: 0; display: flex; flex-direction: column; position: relative; }
.main-content { padding: 25px; flex: 1; }
.sidebar-brand { padding: 24px 20px; display: flex; align-items: center; gap: 12px; border-bottom: 1px solid #1f2937; font-weight: 700; font-size: 15px; color: #fff; letter-spacing: 0.5px; }
.sidebar-menu { padding: 15px 10px; display: flex; flex-direction: column; gap: 2px; overflow-y: auto; flex: 1; }
.menu-group { font-size: 11px; text-transform: uppercase; color: #4b5563; font-weight: 700; padding: 15px 12px 6px 12px; letter-spacing: 0.5px; }
.menu-item { display: flex; align-items: center; gap: 12px; padding: 10px 14px; border-radius: 6px; color: #9ca3af; text-decoration: none; font-size: 13px; font-weight: 500; transition: all 0.15s; }
.menu-item i { width: 16px; font-size: 14px; text-align: center; color: #6b7280; } .menu-item:hover { background: #161e2e; color: #f3f4f6; } .menu-item.active { background: #1e293b; color: #38bdf8; font-weight: 600; } .menu-item.active i { color: #38bdf8; }
.monitor-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 25px; background: #111827; padding: 15px 20px; border-radius: 8px; border: 1px solid #1f2937; }
.monitor-header .m-title { display: flex; align-items: center; gap: 10px; } .monitor-header .m-title h2 { margin: 0; font-size: 18px; color: #fff; }
.realtime-clock { color: #10b981; font-size: 14px; font-weight: bold; background: #0b0f19; padding: 6px 14px; border-radius: 20px; border: 1px solid #1f2937; }
.monitor-grid-top { display: grid; grid-template-columns: repeat(3, 1fr); gap: 15px; margin-bottom: 25px; }
.m-card-mini { background: #111827; border: 1px solid #1f2937; padding: 15px; border-radius: 8px; display: flex; align-items: center; gap: 15px; }
.mini-icon { width: 45px; height: 45px; border-radius: 8px; display: flex; align-items: center; justify-content: center; font-size: 18px; }
.mini-icon.blue { background: rgba(56,189,248,0.1); color: #38bdf8; } .mini-icon.green { background: rgba(16,185,129,0.1); color: #10b981; } .mini-icon.orange { background: rgba(245,158,11,0.1); color: #f59e0b; }
.mini-info { display: flex; flex-direction: column; gap: 2px; } .mini-info .title { font-size: 12px; color: #9ca3af; } .mini-info .value { font-size: 20px; font-weight: 700; color: #fff; }
.monitor-grid-main { display: grid; grid-template-columns: repeat(2, 1fr); gap: 20px; margin-bottom: 25px; }
.m-box { background: #111827; border: 1px solid #1f2937; border-radius: 8px; padding: 20px; } .box-head { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 600; color: #fff; padding-bottom: 15px; border-bottom: 1px solid #1f2937; margin-bottom: 15px; }
.split-layout { display: flex; align-items: center; justify-content: space-between; gap: 15px; }
.details-list { flex: 1; display: flex; flex-direction: column; gap: 8px; font-size: 12px; } .details-list .item { display: flex; justify-content: space-between; border-bottom: 1px dashed #1f2937; padding-bottom: 4px; } .details-list .lbl { color: #64748b; } .details-list .val { color: #f8fafc; font-weight: 500; }
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; background: #111827; padding: 12px 20px; border-radius: 8px; border: 1px solid #1f2937; }
.logo-area { display: flex; align-items: center; } .logo-area h2 { margin: 0; font-size: 16px; color: #f8fafc; }
.btn-group { display: flex; gap: 6px; }
.table-container { background: #111827; border-radius: 8px; border: 1px solid #1f2937; overflow: hidden; }
table { width: 100%; border-collapse: collapse; text-align: left; } th { background: #1f2937; color: #9ca3af; font-size: 13px; font-weight: 500; padding: 14px 16px; } td { padding: 14px 16px; border-bottom: 1px solid #1f2937; font-size: 13px; vertical-align: middle; } tr:hover { background: #161e2e; }
.btn { border: none; padding: 8px 14px; border-radius: 6px; font-size: 13px; cursor: pointer; display: inline-flex; align-items: center; gap: 6px; font-weight: 500; }
.btn-api { background: #10b981; color: white; } .btn-export { background: #1f2937; color: #9ca3af; border: 1px solid #374151; } .btn-check { background: #2563eb; color: white; }
.btn-create-spec { background: #f59e0b; color: #000; font-weight: 700; border: none; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 12px; margin-right: 8px; }
.btn-delete-spec { background: #ef4444; color: #fff; font-weight: 600; border: none; padding: 6px 12px; border-radius: 6px; cursor: pointer; font-size: 12px; transition: background 0.2s; } .btn-delete-spec:hover { background: #dc2626; }
.badge { padding: 3px 6px; border-radius: 4px; font-size: 11px; font-weight: 600; } .badge-info { background: rgba(56,189,248,0.1); color: #38bdf8; } .badge-success { background: rgba(16,185,129,0.1); color: #10b981; } .badge-neutral { background: #1f2937; color: #9ca3af; border: 1px solid #374151; } .badge-warning { background: rgba(245,158,11,0.1); color: #f59e0b; }
.code-font { background: #1f2937; padding: 2px 6px; border-radius: 4px; color: #e2e8f0; }
.action-cell { display: flex; align-items: center; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.75); backdrop-filter: blur(4px); display: flex; justify-content: center; align-items: center; z-index: 200; }
.modal-content { background: #111827; padding: 30px; border-radius: 12px; width: 620px; border: 1px solid #1f2937; }
.form-group { margin-bottom: 16px; } .form-group label { display: block; margin-bottom: 6px; font-size: 12px; color: #9ca3af; font-weight: 600; } .form-group input, .form-group select, .form-group textarea { width: 100%; box-sizing: border-box; background: #0b0f19; border: 1px solid #1f2937; color: #fff; padding: 10px; border-radius: 6px; outline: none; font-size: 13px; } .code-input { font-family: monospace; color: #10b981 !important; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 25px; }
.loading-screen-full { position: fixed; top:0; left:0; right:0; bottom:0; background: #0b0f19; display: flex; flex-direction: column; align-items: center; justify-content: center; z-index: 999; }
.spinner { border: 4px solid rgba(255,255,255,0.1); border-top: 4px solid #38bdf8; border-radius: 50%; width: 35px; height: 35px; animation: spin 1s linear infinite; margin-bottom: 15px; }
@keyframes spin { 100% { transform: rotate(360deg); } }
</style>
