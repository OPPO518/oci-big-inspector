<template>
  <div>
    <div v-if="loading" class="container">
      <div class="card text-center">
        <div class="spinner"></div>
        <h2>系统启动中...</h2>
      </div>
    </div>

    <div v-else-if="needInit" class="container">
      <div class="card fade-in">
        <h2>🚀 大探长面板初始化</h2>
        <form @submit.prevent="submitInit">
          <div class="form-group"><label>管理员账号</label><input v-model="initForm.username" type="text" required /></div>
          <div class="form-group"><label>管理员密码</label><input v-model="initForm.password" type="password" required /></div>
          <button type="submit" :disabled="submitting" class="btn-primary">保存配置</button>
        </form>
      </div>
    </div>

    <div v-else class="dashboard">
      <header class="dash-header">
        <h2>🛡️ OCI 大探长控制台</h2>
        <button @click="showModal = true" class="btn-primary">+ 纳管新账号</button>
      </header>

      <div class="account-grid">
        <div v-for="acc in accounts" :key="acc.id" class="account-card">
          <div class="card-head"><h3>{{ acc.alias }}</h3><span class="region-badge">{{ acc.region }}</span></div>
          <div class="card-body">
            <p><strong>租户:</strong> {{ acc.tenancy_id }}</p>
            <p><strong>指纹:</strong> {{ acc.fingerprint }}</p>
            <p v-if="acc.testResult" :class="acc.testStatus">{{ acc.testResult }}</p>
          </div>
          <div class="card-actions">
            <button class="btn-text" @click="testConnection(acc)" :disabled="acc.isTesting">
              {{ acc.isTesting ? '握手中...' : '测试连接' }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
        <div class="modal-content">
          <h3>添加 OCI 凭证</h3>
          <form @submit.prevent="submitAddAccount">
            <div class="form-group">
              <label>配置粘贴 (粘贴 user=... 那一段，自动填充以下字段)</label>
              <textarea v-model="addForm.raw_config" rows="3" placeholder="粘贴内容..."></textarea>
            </div>
            
            <div class="form-group"><label>账号别名</label><input v-model="addForm.alias" type="text" required /></div>
            <div class="form-group"><label>公钥指纹</label><input v-model="addForm.fingerprint" type="text" required /></div>
            
            <div class="form-group">
              <label>私钥文本</label>
              <textarea v-model="addForm.private_key" rows="4" required></textarea>
            </div>
            
            <input type="hidden" v-model="addForm.tenancy_id">
            <input type="hidden" v-model="addForm.user_id">
            <input type="hidden" v-model="addForm.region">

            <button type="submit" :disabled="submitting" class="btn-primary" style="width:100%">保存凭证</button>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import axios from 'axios'

const loading = ref(true); const needInit = ref(false); const submitting = ref(false); const showModal = ref(false); const accounts = ref([])
const initForm = ref({ username: '', password: '' })
const addForm = ref({ alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '' })

// 自动解析器
watch(() => addForm.value.raw_config, (val) => {
  if (!val) return
  const lines = val.split('\n')
  lines.forEach(line => {
    const [k, v] = line.split('=').map(s => s.trim())
    if (k === 'user') addForm.value.user_id = v
    if (k === 'tenancy') addForm.value.tenancy_id = v
    if (k === 'region') addForm.value.region = v
    if (k === 'fingerprint') addForm.value.fingerprint = v
  })
})

const checkSystemStatus = async () => {
  try {
    const res = await axios.get('/api/status')
    needInit.value = res.data?.need_init
    if (!needInit.value) fetchAccounts()
  } catch(e) { needInit.value = false }
  finally { loading.value = false }
}

const fetchAccounts = async () => {
  const res = await axios.get('/api/accounts/list')
  accounts.value = res.data || []
}

const submitInit = async () => {
  await axios.post('/api/system/init', initForm.value)
  window.location.reload()
}

const submitAddAccount = async () => {
  submitting.value = true
  try {
    await axios.post('/api/accounts/add', addForm.value)
    showModal.value = false
    fetchAccounts()
  } finally { submitting.value = false }
}

const testConnection = async (acc) => {
  acc.isTesting = true
  try {
    const res = await axios.post('/api/accounts/test', { id: acc.id })
    acc.testResult = `✅ ${res.data.tenant_name}`; acc.testStatus = 'text-success'
  } catch(e) { acc.testResult = '❌ 失败'; acc.testStatus = 'text-danger' }
  finally { acc.isTesting = false }
}

onMounted(() => checkSystemStatus())
</script>

<style>
/* 极简基础样式 */
.container { display: flex; justify-content: center; align-items: center; min-height: 100vh; }
.card { background: #fff; padding: 20px; border-radius: 12px; width: 400px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
.dashboard { max-width: 900px; margin: 40px auto; padding: 20px; }
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.account-grid { display: grid; gap: 20px; }
.account-card { background: #fff; padding: 20px; border-radius: 8px; border: 1px solid #eee; }
.modal-overlay { position: fixed; top:0; left:0; right:0; bottom:0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; }
.modal-content { background: #fff; padding: 24px; border-radius: 12px; width: 450px; }
.form-group { margin-bottom: 12px; }
label { display: block; font-size: 12px; font-weight: bold; margin-bottom: 4px; }
input, textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
.btn-primary { background: #000; color: #fff; border: none; padding: 10px; border-radius: 6px; cursor: pointer; }
.text-success { color: green; font-size: 12px; }
.text-danger { color: red; font-size: 12px; }
</style>
