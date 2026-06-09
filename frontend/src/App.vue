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
          </div>
          <div class="card-actions">
            <button class="btn-text">测试连接</button>
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
            <div class="form-group">
              <label>账号别名 (如: 首尔主号)</label>
              <input v-model="addForm.alias" type="text" required />
            </div>
            <div class="form-group">
              <label>主区域 (Region, 如: ap-seoul-1)</label>
              <input v-model="addForm.region" type="text" required />
            </div>
            <div class="form-group">
              <label>租户 OCID (Tenancy ID)</label>
              <input v-model="addForm.tenancy_id" type="text" required />
            </div>
            <div class="form-group">
              <label>用户 OCID (User ID)</label>
              <input v-model="addForm.user_id" type="text" required />
            </div>
            <div class="form-group">
              <label>公钥指纹 (Fingerprint)</label>
              <input v-model="addForm.fingerprint" type="text" required />
            </div>
            <div class="form-group">
              <label>私钥文本 (Private Key, 自动进行 AES 高强度加密)</label>
              <textarea v-model="addForm.private_key" rows="5" required placeholder="-----BEGIN PRIVATE KEY-----&#10;...&#10;-----END PRIVATE KEY-----"></textarea>
            </div>
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

// 状态控制
const loading = ref(true)
const needInit = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const showModal = ref(false)

// 数据集
const accounts = ref([])

// 表单对象
const initForm = ref({ username: '', password: '' })
const addForm = ref({
  alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: ''
})

// 系统握手探测
const checkSystemStatus = async () => {
  try {
    const res = await axios.get('/api/status')
    needInit.value = res.data && res.data.need_init
    if (!needInit.value) {
      fetchAccounts() // 如果已初始化并登录成功，拉取账号列表
    }
  } catch (error) {
    if (error.response && error.response.status === 401) {
      needInit.value = false // 浏览器会弹出 Basic Auth
    }
  } finally {
    setTimeout(() => { loading.value = false }, 500)
  }
}

// 提交初始化
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

// 获取已绑定的账号列表
const fetchAccounts = async () => {
  try {
    const res = await axios.get('/api/accounts/list')
    accounts.value = res.data || []
  } catch (error) {
    console.error('拉取账号失败:', error)
  }
}

// 提交新增账号
const submitAddAccount = async () => {
  submitting.value = true
  try {
    const res = await axios.post('/api/accounts/add', addForm.value)
    if (res.data.status === 'success') {
      showModal.value = false
      addForm.value = { alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '' }
      fetchAccounts() // 刷新列表
    }
  } catch (error) {
    alert(error.response?.data?.error || '添加失败，请检查输入')
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  checkSystemStatus()
})
</script>

<style>
/* 全局与布局基础 */
body {
  background-color: #f4f4f5;
  color: #27272a;
  margin: 0;
  font-family: system-ui, -apple-system, sans-serif;
}
.container { display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; padding: 32px; border-radius: 16px; width: 100%; max-width: 420px; box-shadow: 0 4px 6px -2px rgba(0,0,0,0.05); }

/* 控制台布局 */
.dashboard { max-width: 1200px; margin: 0 auto; padding: 40px 20px; }
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px; }
.dash-title h2 { margin: 0; color: #18181b; }
.dash-title p { margin: 4px 0 0; color: #10b981; font-size: 14px; font-weight: 500; }

/* 账号卡片网格 */
.account-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 20px; }
.empty-state { grid-column: 1 / -1; text-align: center; padding: 60px 0; background: #fff; border-radius: 12px; border: 2px dashed #e4e4e7; color: #71717a; }
.account-card { background: #fff; border-radius: 12px; padding: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.02); border
