<template>
  <div>
    <div v-if="loading" class="loading-screen">
      <div class="spinner"></div>
      <h2>系统启动中...</h2>
    </div>

    <div v-else-if="needInit" class="init-screen">
      <div class="init-card fade-in">
        <h2>🚀 大探长面板初始化</h2>
        <form @submit.prevent="submitInit">
          <div class="form-group"><label>管理员账号</label><input v-model="initForm.username" type="text" required /></div>
          <div class="form-group"><label>管理员密码</label><input v-model="initForm.password" type="password" required /></div>
          <button type="submit" :disabled="submitting" class="btn btn-check" style="width:100%">保存配置</button>
        </form>
      </div>
    </div>

    <div v-else class="dashboard fade-in">
      
      <header class="dash-header">
        <div class="logo-area">
          <i class="fa-solid fa-key" style="color: #38bdf8; margin-right: 10px; font-size: 24px;"></i>
          <h2>租户管理</h2>
        </div>
        
        <div class="search-bar">
          <input type="text" placeholder="输入租户名或区域进行搜索..." />
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
              <th>账号成本</th>
              <th>存活天数</th>
              <th>开机任务</th>
              <th>主区域</th>
              <th>是否多区</th>
              <th>账号类型</th>
              <th>创建时间</th>
              <th>账号状态</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="accounts.length === 0">
              <td colspan="12" class="text-center" style="padding: 40px; color: #64748b;">暂无数据，请点击右上角 API导入</td>
            </tr>
            <tr v-for="(acc, index) in accounts" :key="acc.id">
              <td class="text-muted">{{ index + 1 }}</td>
              
              <td>
                <span class="truncate-text bg-dark-badge" :title="acc.tenancy_id">
                  {{ acc.tenancy_id ? acc.tenancy_id.substring(0, 10) + '...' : '获取中' }}
                </span>
              </td>

              <td class="font-bold text-primary" style="border-bottom: 1px dashed #38bdf8; display: inline-block; margin-top: 15px;">
                {{ acc.alias || 'DEFAULT' }}
              </td>

              <td class="text-muted">0</td>

              <td><span class="badge badge-info">{{ acc.alive_days || 318 }}</span></td>

              <td>
                <span v-if="acc.has_boot_task" class="text-warning">执行中</span>
                <span v-else class="text-muted">无任务</span>
              </td>

              <td class="text-primary">{{ acc.region }}</td>

              <td>
                <span v-if="acc.is_multi_region" class="badge badge-success">● 是</span>
                <span v-else class="text-success" style="font-size: 12px;">● 否</span>
              </td>

              <td><span class="badge badge-outline text-success">个人免费账号</span></td>

              <td class="text-sm text-muted">{{ acc.created_at || '2026-06-09 21:09' }}</td>

              <td>
                <span v-if="acc.status === 'active' || !acc.status" class="badge badge-success"><i class="fa-solid fa-check-circle"></i> 有效</span>
                <span v-else class="badge badge-danger">失效</span>
              </td>

              <td class="action-cell">
                <button class="btn btn-icon btn-action" @click="viewDetails(acc)" title="账户详情">
                  <i class="fa-solid fa-ellipsis"></i>
                </button>
              </td>
            </tr>
          </tbody>
        </table>
        
        <div class="pagination-footer text-muted">
          <span>共 {{ accounts.length }} 条 第 1 / 1 页</span>
        </div>
      </div>

      <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
        <div class="modal-content fade-in-up">
          <h3><i class="fa-solid fa-bolt" style="color:#22c55e;"></i> API配置快速导入</h3>
          <p class="text-sm text-muted" style="margin-bottom: 20px;">API信息 (注意: 新生成的API请勿立即导入，等待10分钟后再执行)</p>
          
          <form @submit.prevent="submitAddAccount">
            <div class="form-group">
              <label>API配置快速导入 (粘贴 OCI Config 文本)</label>
              <textarea v-model="addForm.raw_config" rows="5" class="code-input" placeholder="[DEFAULT]&#10;user=ocid1.user.oc1...&#10;fingerprint=f4:9e...&#10;tenancy=ocid1.tenancy...&#10;region=mx-monterrey-1"></textarea>
            </div>
            
            <div class="form-group">
              <label>自定义名称 (UserName)</label>
              <input v-model="addForm.alias" type="text" required placeholder="例如: 墨西哥蒙特雷A" />
            </div>

            <div class="form-group">
              <label>密钥文件 (.pem 上传或粘贴)</label>
              <div class="file-upload-wrapper">
                <input type="file" @change="handleFileUpload" accept=".pem,.key" id="file-upload" class="hidden-file-input" />
                <label for="file-upload" class="file-upload-btn"><i class="fa-solid fa-upload"></i> 选择文件</label>
                <span class="text-sm" style="margin-left: 10px; color: #38bdf8;">{{ uploadedFileName }}</span>
              </div>
              <textarea v-model="addForm.private_key" rows="2" placeholder="如果不上传，也可直接在此粘贴私钥文本..." style="margin-top:10px;"></textarea>
            </div>
            
            <input type="hidden" v-model="addForm.tenancy_id">
            <input type="hidden" v-model="addForm.user_id">
            <input type="hidden" v-model="addForm.region">
            <input type="hidden" v-model="addForm.fingerprint">

            <div class="modal-actions">
              <button type="button" class="btn btn-export" @click="showModal = false"><i class="fa-solid fa-xmark"></i> 取消</button>
              <button type="submit" :disabled="submitting" class="btn btn-check"><i class="fa-solid fa-save"></i> 保存</button>
            </div>
          </form>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import axios from 'axios'

const loading = ref(true)
const needInit = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const accounts = ref([])
const uploadedFileName = ref('未选择任何文件')

const initForm = ref({ username: '', password: '' })
const addForm = ref({ alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '' })

// 智能解析器
watch(() => addForm.value.raw_config, (val) => {
  if (!val) return
  const lines = val.split('\n')
  lines.forEach(line => {
    const parts = line.split('=')
    if (parts.length >= 2) {
      const k = parts[0].trim().toLowerCase()
      const v = parts.slice(1).join('=').trim() // 防止指纹中的等号被切断
      if (k === 'user') addForm.value.user_id = v
      if (k === 'tenancy') addForm.value.tenancy_id = v
      if (k === 'region') addForm.value.region = v
      if (k === 'fingerprint') addForm.value.fingerprint = v
    }
  })
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
    // 将后端返回的基础数据，映射到前端丰富的表格字段中 (后端未提供的字段暂时使用展示默认值)
    accounts.value = (res.data || []).map(acc => ({
      ...acc,
      alive_days: acc.alive_days || Math.floor(Math.random() * 300) + 10,
      status: 'active'
    }))
  } catch(e) { console.error('获取列表失败', e) }
}

const submitInit = async () => {
  submitting.value = true
  try {
    await axios.post('/api/system/init', initForm.value)
    window.location.reload()
  } catch(e) { alert('初始化失败') }
  finally { submitting.value = false }
}

const submitAddAccount = async () => {
  submitting.value = true
  try {
    await axios.post('/api/accounts/add', addForm.value)
    showModal.value = false
    addForm.value = { alias: '', tenancy_id: '', user_id: '', fingerprint: '', region: '', private_key: '', raw_config: '' }
    uploadedFileName.value = '未选择任何文件'
    fetchAccounts()
  } catch(e) { alert(e.response?.data?.error || '保存失败，请检查配置') }
  finally { submitting.value = false }
}

const batchTest = () => {
  alert('批量检测功能正在对接后端队列...')
}

const viewDetails = (acc) => {
  alert(`打开 [${acc.alias}] 的账户详情页 (即将上线)...`)
}

onMounted(() => checkSystemStatus())
</script>

<style>
/* 字体与全局深色配置 */
@import url('https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css');
body { background-color: #111827; color: #cbd5e1; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; margin: 0; }
.fade-in { animation: fadeIn 0.3s ease-in; }
.fade-in-up { animation: fadeInUp 0.3s ease-out forwards; }
@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
@keyframes fadeInUp { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }

/* 通用组件 */
.text-muted { color: #64748b; }
.text-primary { color: #38bdf8; }
.text-success { color: #10b981; }
.text-warning { color: #f59e0b; }
.text-sm { font-size: 13px; }
.font-bold { font-weight: 600; }
.text-center { text-align: center; }

/* 按钮组 */
.btn { border: none; padding: 8px 16px; border-radius: 6px; font-size: 13px; cursor: pointer; transition: all 0.2s; font-weight: 500; display: inline-flex; align-items: center; gap: 6px; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-api { background: #22c55e; color: #fff; }
.btn-api:hover { background: #16a34a; }
.btn-export { background: #334155; color: #f8fafc; }
.btn-export:hover { background: #475569; }
.btn-check { background: #3b82f6; color: #fff; }
.btn-check:hover { background: #2563eb; }
.btn-icon { padding: 8px 12px; background: #1e293b; color: #cbd5e1; }
.btn-icon:hover { background: #334155; color: #fff; }
.btn-action { background: transparent; border: 1px solid #334155; color: #38bdf8; border-radius: 4px; }
.btn-action:hover { background: #1e293b; border-color: #38bdf8; }

/* 布局 */
.loading-screen, .init-screen { display: flex; justify-content: center; align-items: center; height: 100vh; flex-direction: column; }
.init-card { background: #1f2937; padding: 30px; border-radius: 12px; width: 400px; box-shadow: 0 10px 25px rgba(0,0,0,0.5); }
.dashboard { padding: 20px; max-width: 1400px; margin: 0 auto; }

/* 头部 */
.dash-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; background: #1f2937; padding: 15px 20px; border-radius: 8px; border: 1px solid #374151; }
.logo-area { display: flex; align-items: center; }
.logo-area h2 { margin: 0; font-size: 20px; color: #f8fafc; }
.search-bar { display: flex; width: 350px; }
.search-bar input { flex: 1; background: #111827; border: 1px solid #374151; padding: 8px 12px; border-radius: 6px 0 0 6px; color: #fff; outline: none; }
.search-bar input:focus { border-color: #3b82f6; }
.btn-search { background: #3b82f6; border: none; color: white; padding: 8px 15px; border-radius: 0 6px 6px 0; cursor: pointer; }
.btn-group { display: flex; gap: 10px; }

/* 表格 */
.table-container { background: #1f2937; border-radius: 8px; border: 1px solid #374151; overflow-x: auto; }
table { width: 100%; border-collapse: collapse; text-align: left; }
th { background: #374151; color: #9ca3af; font-size: 13px; font-weight: 500; padding: 12px 15px; white-space: nowrap; }
td { padding: 15px; border-bottom: 1px solid #374151; font-size: 14px; vertical-align: middle; }
tr:hover { background: #111827; }

/* 标签与截断 */
.badge { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: 500; }
.badge-info { background: rgba(56, 189, 248, 0.1); color: #38bdf8; border: 1px solid rgba(56, 189, 248, 0.3); }
.badge-success { background: rgba(16, 185, 129, 0.1); color: #10b981; }
.badge-danger { background: rgba(239, 68, 68, 0.1); color: #ef4444; border: 1px solid rgba(239, 68, 68, 0.3); }
.badge-outline { border: 1px solid #10b981; background: transparent; }
.bg-dark-badge { background: #111827; padding: 4px 6px; border-radius: 4px; font-family: monospace; }
.truncate-text { display: inline-block; max-width: 120px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }

/* 底部与表单 */
.pagination-footer { padding: 15px; text-align: right; border-top: 1px solid #374151; font-size: 13px; }
.modal-overlay { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: rgba(0,0,0,0.7); backdrop-filter: blur(3px); display: flex; justify-content: center; align-items: center; z-index: 100; }
.modal-content { background: #1f2937; padding: 30px; border-radius: 12px; width: 600px; border: 1px solid #374151; }
.modal-content h3 { margin-top: 0; color: #f8fafc; }
.form-group { margin-bottom: 20px; }
.form-group label { display: block; margin-bottom: 8px; font-size: 13px; color: #cbd5e1; }
.form-group input[type="text"], .form-group input[type="password"], .form-group textarea { width: 100%; box-sizing: border-box; background: #111827; border: 1px solid #374151; color: #fff; padding: 10px; border-radius: 6px; outline: none; transition: border 0.2s; }
.form-group input:focus, .form-group textarea:focus { border-color: #3b82f6; }
.code-input { font-family: monospace; font-size: 13px; color: #10b981 !important; }
.file-upload-wrapper { display: flex; align-items: center; }
.hidden-file-input { display: none; }
.file-upload-btn { background: #374151; color: #f8fafc; padding: 8px 16px; border-radius: 6px; cursor: pointer; font-size: 13px; transition: 0.2s; }
.file-upload-btn:hover { background: #475569; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 30px; }
.spinner { border: 4px solid rgba(255,255,255,0.1); border-top: 4px solid #38bdf8; border-radius: 50%; width: 40px; height: 40px; animation: spin 1s linear infinite; margin-bottom: 20px; }
@keyframes spin { 100% { transform: rotate(360deg); } }
</style>
