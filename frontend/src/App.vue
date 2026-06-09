<template>
  <div class="container">
    <div v-if="loading" class="card text-center">
      <div class="spinner"></div>
      <h2>系统握手中...</h2>
      <p>正在与 Go 核心引擎建立安全连接</p>
    </div>

    <div v-else-if="needInit" class="card fade-in">
      <div class="header">
        <h2>🚀 大探长面板初始化</h2>
        <p>首次运行，请设置您的最高管理员安全凭证。</p>
      </div>
      <form @submit.prevent="submitInit">
        <div class="form-group">
          <label>设置管理员账号</label>
          <input 
            v-model="form.username" 
            type="text" 
            required 
            placeholder="例如: admin" 
            autocomplete="username"
          />
        </div>
        <div class="form-group">
          <label>设置高强度密码</label>
          <input 
            v-model="form.password" 
            type="password" 
            required 
            placeholder="请使用大小写字母与数字组合" 
            autocomplete="new-password"
          />
        </div>
        <button type="submit" :disabled="submitting">
          {{ submitting ? '配置安全加密中...' : '保存并初始化系统' }}
        </button>
        <p v-if="errorMessage" class="error">{{ errorMessage }}</p>
      </form>
    </div>

    <div v-else class="card dashboard fade-in">
      <h2>✅ 验证通过，欢迎进入控制台</h2>
      <p>基础通信与安全拦截模块已就绪。</p>
      </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

// 页面核心状态
const loading = ref(true)
const needInit = ref(false)
const submitting = ref(false)
const errorMessage = ref('')

// 表单数据
const form = ref({
  username: '',
  password: ''
})

// 探测后端引擎状态
const checkSystemStatus = async () => {
  try {
    // 页面加载即向后端请求状态
    const res = await axios.get('/api/status')
    
    // 如果后端的 BasicAuth 拦截器返回了待初始化指令
    if (res.data && res.data.need_init) {
      needInit.value = true
    } else {
      needInit.value = false
    }
  } catch (error) {
    // 【核心机制】如果系统已初始化，后端会直接返回 401 Unauthorized。
    // 这时浏览器原生的 Basic Auth 弹窗会自动接管，要求用户输入账号密码。
    if (error.response && error.response.status === 401) {
      needInit.value = false
    } else {
      errorMessage.value = '无法连接到后端引擎，请检查容器日志。'
    }
  } finally {
    // 延迟 500ms 移除 Loading 状态，让界面过渡更平滑
    setTimeout(() => { loading.value = false }, 500)
  }
}

// 提交初始化注册
const submitInit = async () => {
  submitting.value = true
  errorMessage.value = ''
  
  try {
    const res = await axios.post('/api/system/init', form.value)
    if (res.data.status === 'success') {
      alert('面板初始化成功！系统即将重载以应用底层安全拦截机制。')
      // 重载页面，强制触发并唤起浏览器的 Basic Auth 账号密码登录输入框
      window.location.reload()
    }
  } catch (error) {
    errorMessage.value = error.response?.data?.error || '初始化请求失败，请检查系统日志。'
  } finally {
    submitting.value = false
  }
}

// Vue 挂载时立即执行探测
onMounted(() => {
  checkSystemStatus()
})
</script>

<style>
/* 极简无依赖的纯 CSS 样式库，保障前端体积轻如鸿毛 */
body {
  background-color: #f4f4f5;
  color: #27272a;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  margin: 0;
  font-family: system-ui, -apple-system, sans-serif;
}

.container {
  width: 100%;
  max-width: 420px;
  padding: 20px;
  box-sizing: border-box;
}

.card {
  background: #ffffff;
  padding: 32px;
  border-radius: 16px;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
}

.header {
  text-align: center;
  margin-bottom: 28px;
}

.header h2 {
  margin: 0 0 8px;
  font-size: 24px;
  color: #18181b;
}

.header p {
  margin: 0;
  color: #71717a;
  font-size: 14px;
}

.form-group {
  margin-bottom: 18px;
}

label {
  display: block;
  margin-bottom: 6px;
  font-weight: 600;
  color: #3f3f46;
  font-size: 14px;
}

input {
  width: 100%;
  padding: 12px;
  border: 1px solid #d4d4d8;
  border-radius: 8px;
  box-sizing: border-box;
  font-size: 14px;
  background-color: #fafafa;
  transition: all 0.2s;
}

input:focus {
  outline: none;
  border-color: #000000;
  background-color: #ffffff;
  box-shadow: 0 0 0 3px rgba(0, 0, 0, 0.05);
}

button {
  width: 100%;
  padding: 14px;
  background-color: #000000;
  color: white;
  border: none;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  margin-top: 10px;
  transition: background-color 0.2s;
}

button:hover:not(:disabled) {
  background-color: #27272a;
}

button:disabled {
  background-color: #a1a1aa;
  cursor: not-allowed;
}

.error {
  color: #ef4444;
  font-size: 14px;
  margin-top: 16px;
  text-align: center;
  font-weight: 500;
}

.text-center {
  text-align: center;
}

.dashboard h2 {
  color: #10b981;
}

/* 小动画：加载 Spinner 和界面淡入 */
.spinner {
  border: 3px solid #f3f3f3;
  border-top: 3px solid #000000;
  border-radius: 50%;
  width: 32px;
  height: 32px;
  animation: spin 1s linear infinite;
  margin: 0 auto 16px;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}

.fade-in {
  animation: fadeIn 0.4s ease-out forwards;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
