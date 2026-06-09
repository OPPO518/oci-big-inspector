<template>
  <div>
    <div v-if="loading" class="container">
      <div class="card text-center">
        <div class="spinner"></div>
        <h2>系统握手中...</h2>
        <p>正在与 Go 核心引擎建立安全连接</p>
      </div>
    </div>

    <div v-else-if="needInit" class="container">
      <div class="card fade-in">
        <div class="header">
          <h2>🚀 大探长面板初始化</h2>
          <p>首次运行，请设置您的最高管理员安全凭证。</p>
        </div>
        <form @submit.prevent="submitInit">
          <div class="form-group">
            <label>设置管理员账号</label>
            <input v-model="initForm.username" type="text" required placeholder="例如: admin" />
          </div>
          <div class="form-group">
            <label>设置高强度密码</label>
            <input v-model="initForm.password" type="password" required placeholder="字母与数字组合" />
          </div>
          <button type="submit" :disabled="submitting" class="btn-primary">
            {{ submitting ? '配置安全加密中...' : '保存并初始化系统' }}
          </button>
          <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
        </form>
      </div>
    </div>

    <div v-else class="dashboard fade-in">
      <header class="dash-header">
        <div class="dash-title">
          <h2>🛡️ OCI 大探长控制台</h2>
          <p>核心加密引擎运行中</p>
        </div>
        <button @click="showModal = true" class="btn-primary">+ 纳管新账号</button>
      </header>

      <div class="account-grid">
        <div v-if="accounts.length === 0" class="empty-state">
          <p>📭 暂无纳管的甲骨文账号</p>
          <span>请点击右上角按钮添加你的第一个 API 凭证</span>
        </div>
        
        <div v-for="acc in accounts" :key="acc.id" class="account-card">
          <div class="card-head">
            <h3>{{ acc.alias }}</h3>
            <span class="region-badge">{{ acc.region }}</span>
          </div>
          <div class="card-body">
            <p><strong>租户 ID:</strong> <span class="truncate">{{ acc.tenancy_id }}</span></p>
            <p><strong>指纹:</strong> <span class="truncate">{{ acc.fingerprint }}</span></p>
            
            <p v-if="acc.testResult" class="test-feedback" :class="acc.testStatus">
              {{ acc.testResult }}
            </p>
          </div>
          
          <div class="card-actions">
            <button class="btn-text" @click="testConnection(acc)" :disabled="acc.isTesting">
              {{ acc.isTesting ? '正在握手...' : '测试连接' }}
            </button>
            <button class="btn-text">实例管理</button>
          </div>
        </div>
      </div>

      <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
        <div class="modal-content fade-in-up">
          <div class="modal-header">
            <h3>添加甲骨文 API 凭证</h3>
            <button class="close-btn" @click="showModal = false">×</button>
          </div>
          <form @submit.prevent="submitAddAccount">
            <div class="form-group"><label>账号别名</label><input v-model="addForm.alias" type="text" required /></div>
            <div class="form-group"><label>主区域 (Region)</label><input v-model="addForm.region" type="text" required /></div>
            <div class="form-group"><label>租户 OCID</label><input v-model="addForm.tenancy_id" type="text" required /></div>
            <div class="form-group"><label>用户 OCID</label><input v-model="addForm.user_id" type="text" required /></div>
            <div class="form-group"><label>公钥指纹</label><input v-model="addForm.fingerprint" type="text" required /></div>
            <div class="form-group"><label>私钥文本</label><textarea v-model="addForm.private_key" rows="5" required></textarea></div>
            <button type="submit" :disabled="submitting" class="btn-primary mt-2">
              {{ submitting ? '加密入库中...' : '安全保存凭证' }}
            </button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const loading = ref(true)
const needInit = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const showModal = ref(false)
const accounts = ref([])
const initForm = ref({ username: '', password: '' })
const addForm = ref({ alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '' })

const checkSystemStatus = async () => {
  try {
    const res = await axios.get('/api/status')
    needInit.value = res.data && res.data.need_init
    if (!needInit.value) fetchAccounts()
  } catch (error) {
    if (error.response?.status === 401) needInit.value = false
  } finally {
    setTimeout(() => { loading.value = false }, 500)
  }
}

const submitInit = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/system/init', initForm.value)
    if (res.data.status === 'success') {
      alert('面板初始化成功！请刷新页面以加载底层安全拦截。')
      window.location.reload()
    }
  } catch (error) {
    errorMessage.value = error.response?.data?.error || '初始化失败'
  } finally {
    submitting.value = false
  }
}

const fetchAccounts = async () => {
  try {
    const res = await axios.get('/api/accounts/list')
    // 将拿到的账号列表加上用于测试状态的空字段
    accounts.value = (res.data || []).map(acc => ({
      ...acc,
      isTesting: false,
      testResult: '',
      testStatus: ''
    }))
  } catch (error) {
    console.error('拉取账号失败:', error)
  }
}

const submitAddAccount = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/accounts/add', addForm.value)
    if (res.data.status === 'success') {
      showModal.value = false
      addForm.value = { alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '' }
      fetchAccounts()
    }
  } catch (error) {
    alert(error.response?.data?.error || '添加失败')
  } finally {
    submitting.value = false
  }
}

// 【新增核心功能】调用后端测试该账号凭证的连通性
const testConnection = async (acc) => {
  acc.isTesting = true
  acc.testResult = ''
  
  try {
    const res = await axios.post('/api/accounts/test', { id: acc.id })
    if (res.data.status === 'success') {
      acc.testResult = `✅ API 通道正常 | 真实租户名: ${res.data.tenant_name}`
      acc.testStatus = 'text-success'
    }
  } catch (error) {
    acc.testResult = `❌ ${error.response?.data?.error || '网络超时或配置错误'}`
    acc.testStatus = 'text-danger'
  } finally {
    acc.isTesting = false
  }
}

onMounted(() => checkSystemStatus())
</script>

<style>
/* ... (保留之前的所有样式，只需在底部追加下面这两行反馈文本颜色) ... */
body { background-color: #f4f4f5; color: #27272a; margin: 0; font-family: system-ui, -apple-system, sans-serif; }
.container { display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; padding: 32px; border-radius: 16px; width: 100%; max-width: 420px; box-shadow: 0 4px 6px -2px rgba(0,0,0,0.05); }
.dashboard { max-width: 1200px; margin: 0 auto; padding: 40px 20px; }
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.dash-title h2 { margin: 0; color: #18181b; }
.dash-title p { margin: 4px 0 0; color: #10b981; font-size: 14px; font-weight: 500; }
.account-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 20px; }
.empty-state { grid-column: 1 / -1; text-align: center; padding: 60px 0; background: #fff; border-radius: 12px; border: 2px dashed #e4e4e7; color: #71717a; }
.account-card { background: #fff; border-radius: 12px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.02); border: 1px solid #f4f4f5; transition: transform 0.2s; }
.account-card:hover { transform: translateY(-2px); box-shadow: 0 10px 15px -3px rgba(0,0,0,0.05); }
.card-head { display: flex; justify-content: space-between; align-items: center; border-bottom: 1px solid #f4f4f5; padding-bottom: 12px; margin-bottom: 12px; }
.card-head h3 { margin: 0; font-size: 18px; }
.region-badge { background: #e0e7ff; color: #4338ca; padding: 4px 8px; border-radius: 6px; font-size: 12px; font-weight: 600; }
.card-body p { margin: 8px 0; font-size: 13px; color: #52525b; display: flex; flex-direction: column; }
.truncate { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; background: #f4f4f5; padding: 4px; border-radius: 4px; margin-top: 4px; font-family: monospace; }
.card-actions { margin-top: 16px; display: flex; gap: 10px; }
.form-group { margin-bottom: 16px; }
label { display: block; margin-bottom: 6px; font-size: 13px; font-weight: 600; color: #3f3f46; }
input, textarea { width: 100%; padding: 10px; border: 1px solid #d4d4d8; border-radius: 8px; font-size: 14px; box-sizing: border-box; background: #fafafa; }
input:focus, textarea:focus { outline: none; border-color: #000; background: #fff; }
textarea { resize: vertical; font-family: monospace; }
.btn-primary { background: #000; color: #fff; border: none; padding: 10px 16px; border-radius: 8px; font-weight: 600; cursor: pointer; transition: background 0.2s; }
.btn-primary:hover:not(:disabled) { background: #27272a; }
.btn-primary:disabled { background: #a1a1aa; cursor: not-allowed; }
.btn-text { background: transparent; border: 1px solid #e4e4e7; padding: 6px 12px; border-radius: 6px; font-size: 13px; cursor: pointer; color: #18181b; }
.btn-text:hover:not(:disabled) { background: #f4f4f5; }
.mt-2 { margin-top: 16px; width: 100%; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.4); display: flex; justify-content: center; align-items: center; z-index: 50; backdrop-filter: blur(2px); }
.modal-content { background: #fff; width: 100%; max-width: 500px; padding: 24px; border-radius: 16px; max-height: 90vh; overflow-y: auto; }
.modal-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.modal-header h3 { margin: 0; }
.close-btn { background: transparent; border: none; font-size: 24px; cursor: pointer; color: #71717a; }
.spinner { border: 3px solid #f3f3f3; border-top: 3px solid #000; border-radius: 50%; width: 32px; height: 32px; animation: spin 1s linear infinite; margin: 0 auto 16px; }
@keyframes spin { 100% { transform: rotate(360deg); } }
.fade-in { animation: fadeIn 0.3s ease-out forwards; }
.fade-in-up { animation: fadeInUp 0.3s ease-out forwards; }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes fadeInUp { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }

/* 这是为 API 测试结果专门加的颜色 */
.test-feedback { margin-top: 12px !important; padding: 8px !important; border-radius: 6px; font-weight: 500; font-size: 12px !important; background: #fafafa; border-left: 4px solid #d4d4d8; }
.text-success { color: #059669; border-left-color: #10b981; }
.text-danger { color: #dc2626; border-left-color: #ef4444; }
</style>
